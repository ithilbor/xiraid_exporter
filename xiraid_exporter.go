package main

import (
	// Go
	"os"
	"os/user"
	"runtime"
	"net/http"
	// Xiraid exporter
	"github.com/ironcub3/xiraid_exporter/handlers"
	// Kingpin
	"github.com/alecthomas/kingpin/v2"
	// Prometheus
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

func main() {
	var (
		metricsEndpoint = kingpin.Flag(
			"metrics-endpoint",
			"URL path where metrics will be exposed for scraping.",
		).Default("/metrics").String()
		prometheusDefaultMetrics = kingpin.Flag(
			"prometheus-default-metrics",
			"Enable metrics about the exporter itself like: promhttp_*, process_*, go_* (default: disabled).",
		).Default("false").Bool()
		maxConcurrentRequests = kingpin.Flag(
			"max-concurrent-requests",
			"Maximum number of parallel scrape requests. Use 0 to disable.",
		).Default("40").Int()
		goMaxProcs = kingpin.Flag(
			"gomaxprocs",
			"The target number of cores Go will run on (environmen variable GOMAXPROCS)",
		).Envar("GOMAXPROCS").Default("1").Int()
		toolkitFlags = kingpinflag.AddFlags(kingpin.CommandLine, ":9505")
	)
	// Set up the prometheus logger and kingpin
	promslogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promslogConfig)
	kingpin.Version(version.Print("xiraid_exporter"))
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.HelpFlag.Short('h')
	// Parsing here and after because of the dependancie of toolkitFlags from xrExpPort
	kingpin.Parse()
	logger := promslog.New(promslogConfig)
	logger.Info("Starting xiraid_exporter", "version", version.Info())
	logger.Info("Build context", "build_context", version.BuildContext())
	if user, err := user.Current(); err == nil && user.Uid == "0" {
		logger.Warn(
			"You're running the xiraid_exporter as root user." + 
			"This exporter is designed to run as unprivileged user," +
			"therfore root is not required.")
	}
	runtime.GOMAXPROCS(*goMaxProcs)
	logger.Debug("Go maximum processors", "procs", runtime.GOMAXPROCS(0))
	// Create the handler to expose the metrics
	http.Handle(*metricsEndpoint, handlers.NewMetricsHandler(*prometheusDefaultMetrics, *maxConcurrentRequests, logger))
	if *metricsEndpoint != "/" {
		landingConfig := web.LandingConfig{
			Name:        "xiraid_exporter",
			Description: "Xinnor xiRAID metrics exporter for Prometheus",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{
					Address: *metricsEndpoint,
					Text:    "Metrics",
				},
			},
		}
		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			logger.Error("Failed to initiate the http handler", "error", err.Error())
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}
	// Start the http server
	server := &http.Server{}
	if err := web.ListenAndServe(server, toolkitFlags, logger); err != nil {
		logger.Error("Error starting the http server", "error", err.Error())
		os.Exit(1)
	}
}