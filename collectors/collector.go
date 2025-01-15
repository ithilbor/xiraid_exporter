// Package collector includes all individual collectors to gather and export system metrics.
package collector

import (
	// Go
	"fmt"
	"time"
	"sync"
	"errors"
	"log/slog"
	// Xiraid exporter
	xrprotos "github.com/ironcub3/xiraid_exporter/protos"
	xrgrpc "github.com/ironcub3/xiraid_exporter/connections/grpc"
	// Kingpin
	"github.com/alecthomas/kingpin/v2"
	// Prometheus
	"github.com/prometheus/client_golang/prometheus"
)

// CollectorRegistry implements the prometheus.Collector interface.
type CollectorRegistry struct {
	Collectors map[string]Collector
	logger     *slog.Logger
}

// Collector is the interface a collector has to implement.
type Collector interface {
	// Get new metrics and expose them via prometheus registry.
	Update(ch chan<- prometheus.Metric) error
}

// Namespace defines the common namespace to be used by all metrics.
const namespace = "xiraid"

// Used to enable or disable a new collector by default
// can be used only one variable but for better readability
// of the code used two
const (
	defaultEnabled  = true
	defaultDisabled = false
)

// Variables inherited by all collectors
var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"xiraid_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"xiraid_exporter: Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
)

// Collectors variables
var (
	factories              = make(map[string]func(logger *slog.Logger, xrClient xrprotos.XRAIDServiceClient) (Collector, error))
	initiatedCollectorsMtx = sync.Mutex{}
	initiatedCollectors    = make(map[string]Collector)
	collectorState         = make(map[string]*bool)
	forcedCollectors       = map[string]bool{} // collectors which have been explicitly enabled or disabled
)

// ErrNoData indicates the collector found no data to collect, but had no other error.
var ErrNoData = errors.New("collector returned no data")

// NewXrCollector creates a new collector registry that contains all the collectors.
func NewCollectorRegistry(logger *slog.Logger, filters ...string) (*CollectorRegistry, error) {
	// This connection is execute only one since the collector of collectors runs only one time
	xrClient:= xrgrpc.NewXiraidClient(logger)
	f := make(map[string]bool)
	for _, filter := range filters {
		enabled, exist := collectorState[filter]
		if !exist {
			return nil, fmt.Errorf("missing collector: %s", filter)
		}
		if !*enabled {
			return nil, fmt.Errorf("disabled collector: %s", filter)
		}
		f[filter] = true
	}
	collectors := make(map[string]Collector)
	initiatedCollectorsMtx.Lock()
	defer initiatedCollectorsMtx.Unlock()
	for key, enabled := range collectorState {
		if !*enabled || (len(f) > 0 && !f[key]) {
			continue
		}
		if collector, ok := initiatedCollectors[key]; ok {
			collectors[key] = collector
		} else {
			collector, err := factories[key](logger.With("collector", key), xrClient)
			if err != nil {
				return nil, err
			}
			collectors[key] = collector
			initiatedCollectors[key] = collector
		}
	}
	return &CollectorRegistry{Collectors: collectors, logger: logger}, nil
}

// Function that register the collector only 
func registerCollector(collector string, isDefaultEnabled bool, factory func(logger *slog.Logger, xrClient xrprotos.XRAIDServiceClient) (Collector, error)) {
	var helpDefaultState string
	if isDefaultEnabled {
		helpDefaultState = "enabled"
	} else {
		helpDefaultState = "disabled"
	}
	flagName := fmt.Sprintf("collector.%s_%s", namespace, collector)
	flagHelp := fmt.Sprintf("Enable/disable the %s collector (default: %s).", collector, helpDefaultState)
	// used to parse the true vaule as a string
	defaultValue := fmt.Sprintf("%v", isDefaultEnabled)
	flag := kingpin.Flag(flagName, flagHelp).Default(defaultValue).Action(collectorFlagAction(collector)).Bool()
	collectorState[collector] = flag
	factories[collector] = factory
}

// DisableDefaultCollectors sets the collector state to false for all collectors which
// have not been explicitly enabled on the command line.
func DisableDefaultCollectors() {
	for c := range collectorState {
		if _, ok := forcedCollectors[c]; !ok {
			*collectorState[c] = false
		}
	}
}

// Describe implements the prometheus.Collector interface.
func (n CollectorRegistry) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (n CollectorRegistry) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(n.Collectors))
	for name, c := range n.Collectors {
		go func(name string, c Collector) {
			execute(name, c, ch, n.logger)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
}

func execute(name string, c Collector, ch chan<- prometheus.Metric, logger *slog.Logger) {
	begin := time.Now()
	err := c.Update(ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		if IsNoDataError(err) {
			logger.Debug("collector returned no data", "name", name, "duration_seconds", duration.Seconds(), "err", err)
		} else {
			logger.Error("collector failed", "name", name, "duration_seconds", duration.Seconds(), "err", err)
		}
		success = 0
	} else {
		logger.Debug("collector succeeded", "name", name, "duration_seconds", duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

func IsNoDataError(err error) bool {
	return err == ErrNoData
}

// collectorFlagAction generates a new action function for the given collector
// to track whether it has been explicitly enabled or disabled from the command line.
// A new action function is needed for each collector flag because the ParseContext
// does not contain information about which flag called the action.
// See: https://github.com/alecthomas/kingpin/issues/294
func collectorFlagAction(collector string) func(ctx *kingpin.ParseContext) error {
	return func(ctx *kingpin.ParseContext) error {
		forcedCollectors[collector] = true
		return nil
	}
}