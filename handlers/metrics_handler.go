package handlers

import (
	// Go
	"fmt"
	"sort"
	"slices"
	"net/http"
	"log/slog"
	// Xiraid exporter
	"github.com/ironcub3/xiraid_exporter/collectors"
	// Prometheus
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	promcollectors "github.com/prometheus/client_golang/prometheus/collectors"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
)

type metricsHandler struct {
	httpHandler http.Handler
	enabledCollectors []string 
	exporterMetricsRegistry *prometheus.Registry
	prometheusDefaultMetrics  bool
	maxConcurrentRequests int
	logger *slog.Logger
}

func NewMetricsHandler(prometheusDefaultMetrics bool, maxConcurrentRequests int, logger *slog.Logger) *metricsHandler {
	h := &metricsHandler{
		exporterMetricsRegistry: prometheus.NewRegistry(),
		prometheusDefaultMetrics: prometheusDefaultMetrics,
		maxConcurrentRequests: maxConcurrentRequests,
		logger: logger,
	}
	// Enables the Prometheus default metrics collector
	if h.prometheusDefaultMetrics {
		h.exporterMetricsRegistry.MustRegister(
			promcollectors.NewProcessCollector(promcollectors.ProcessCollectorOpts{}),
			promcollectors.NewGoCollector(),
		)
	}
	if innerHandler, err := h.innerHandler(); err != nil {
		panic(fmt.Sprintf("Couldn't create metrics handler: %s", err))
	} else {
		h.httpHandler = innerHandler
	}
	return h
}

// ServeHTTP implements http.Handler.
func (h *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	collects := r.URL.Query()["collect[]"]
	h.logger.Debug("collect query:", "collects", collects)
	excludes := r.URL.Query()["exclude[]"]
	h.logger.Debug("exclude query:", "excludes", excludes)
	if len(collects) == 0 && len(excludes) == 0 {
		// No filters, use the prepared unfiltered handler.
		h.httpHandler.ServeHTTP(w, r)
		return
	}
	if len(collects) > 0 && len(excludes) > 0 {
		h.logger.Debug("rejecting combined collect and exclude queries")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Combined collect and exclude queries are not allowed."))
		return
	}
	filters := &collects
	if len(excludes) > 0 {
		// In exclude mode, filtered collectors = enabled - excludeed.
		f := []string{}
		for _, c := range h.enabledCollectors {
			if (slices.Index(excludes, c)) == -1 {
				f = append(f, c)
			}
		}
		filters = &f
	}
	// To serve filtered metrics, we create a filtering handler on the fly.
	filteredHandler, err := h.innerHandler(*filters...)
	if err != nil {
		h.logger.Warn("Couldn't create filtered metrics handler", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Couldn't create filtered metrics handler: %s", err)))
		return
	}
	filteredHandler.ServeHTTP(w, r)
}

// innerHandler creates the filtered and unfilterd http.Handler
func (h *metricsHandler) innerHandler(filters ...string) (http.Handler, error) {
	// Create the collector of collectors filtered or not
	nc, err := collector.NewCollectorRegistry(h.logger, filters...)
	if err != nil {
		return nil, fmt.Errorf("couldn't create collector: %s", err)
	}
	if len(filters) == 0 {
		h.logger.Info("Enabled collectors:")
		for n := range nc.Collectors {
			h.enabledCollectors = append(h.enabledCollectors, n)
		}
		sort.Strings(h.enabledCollectors)
		for _, c := range h.enabledCollectors {
			h.logger.Info(c)
		}
	}
	// Creating the base collector
	r := prometheus.NewRegistry()
	r.MustRegister(versioncollector.NewCollector("xiraid_exporter"))
	if err := r.Register(nc); err != nil {
		return nil, fmt.Errorf("couldn't register xiraid version collector: %s", err)
	}
	var handler http.Handler
	// Create the collector with prometheus default metrics
	if h.prometheusDefaultMetrics {
		handler = promhttp.HandlerFor(
			prometheus.Gatherers{h.exporterMetricsRegistry, r},
			promhttp.HandlerOpts{
				ErrorLog:            slog.NewLogLogger(h.logger.Handler(), slog.LevelError),
				ErrorHandling:       promhttp.ContinueOnError,
				MaxRequestsInFlight: h.maxConcurrentRequests,
				Registry:            h.exporterMetricsRegistry,
			},
		)
		// Note that we have to use h.exporterMetricsRegistry here to
		// use the same promhttp metrics for all expositions.
		handler = promhttp.InstrumentMetricHandler(
			h.exporterMetricsRegistry, handler,
		)
	} else {
		// Create the collecctor without prometheus default metrics
		handler = promhttp.HandlerFor(
			r,
			promhttp.HandlerOpts{
				ErrorLog:            slog.NewLogLogger(h.logger.Handler(), slog.LevelError),
				ErrorHandling:       promhttp.ContinueOnError,
				MaxRequestsInFlight: h.maxConcurrentRequests,
			},
		)
	}
	return handler, nil
}