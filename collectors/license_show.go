// This is the collector of the xiraid raid information.
// The equivalent of this collector from CLI is:
// 'xicli raid show'

package collector

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"

	// Xiraid exporter
	xrprotos "github.com/ithilbor/xiraid_exporter/protos"
	// Prometheus
	"github.com/prometheus/client_golang/prometheus"
	// Gjson
	"github.com/tidwall/gjson"
)

// licenseShowCollector is modified to work with raw text data instead of JSON.
// hwkey --> Static
// Kernel version --> Changeable
// license_key --> Changeable
// version --> Changeable
// crypto_version --> Changeable
// created --> Changeable
// expired --> Changeable
// disks --> Changeable
// levels --> Changeable
// type --> Changeable
// disks_in_use --> Changeable
// status --> Changeable

type licenseShowCollector struct {
    licenseInfo   *prometheus.Desc
    levels        *prometheus.Desc
    disks         *prometheus.Desc
    disksInUse    *prometheus.Desc
    logger        *slog.Logger
    xrClient      xrprotos.XRAIDServiceClient
}

//TODO: use the logger

func init() {
	registerCollector("license_show", defaultEnabled, NewLicenseShowCollector)
}

func NewLicenseShowCollector(logger *slog.Logger, xrClient xrprotos.XRAIDServiceClient) (Collector, error) {
    return &licenseShowCollector{
        licenseInfo: prometheus.NewDesc(
            prometheus.BuildFQName(namespace, "license", "info"),
            "License information",
            []string{"hwkey", "kernel_version", "license_key", "license_version", "license_crypto_version",
             "license_status", "license_creation", "license_expiration"}, nil,
        ),
        levels: prometheus.NewDesc(
            prometheus.BuildFQName(namespace, "license", "levels"),
            "Maximum RAID level",
            []string{"hwkey"}, nil,
        ),
        disks: prometheus.NewDesc(
            prometheus.BuildFQName(namespace, "license", "disks"),
            "Maximum number of drives",
            []string{"hwkey"}, nil,
        ),
        disksInUse: prometheus.NewDesc(
            prometheus.BuildFQName(namespace, "license", "disks_in_use"),
            "Number of used drives in the system",
            []string{"hwkey"}, nil,
        ),
        logger:   logger,
        xrClient: xrClient,
    }, nil
}

func fetchLicenseShowData(xrClient xrprotos.XRAIDServiceClient) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	licenseShowRequest := &xrprotos.LicenseShow{}
	r1, err := xrClient.LicenseShow(ctx, licenseShowRequest)
	if err != nil {
		return "", fmt.Errorf("could not get RAID entries: %w", err)
	}
	return r1.GetMessage(), nil
}

func (collector *licenseShowCollector) Update(ch chan<- prometheus.Metric) error {
    data, err := fetchLicenseShowData(collector.xrClient)
    if err != nil {
        return fmt.Errorf("could not get RAID entries: %w", err)
    }
    // Hardware key
    hwkey := getOrDefault(data, "hwkey")
    // license info
    kernelVersion := getOrDefault(data, "Kernel version")
    log.Print(kernelVersion)
    licenseKey := getOrDefault(data, "license_key")
    licenseVersion := getOrDefault(data,"version")
    licenseCryptoVersion := getOrDefault(data, "crypto_version")
    licenseStatus := getOrDefault(data, "status")
    licenseCreation := getOrDefault(data, "created")
    licenseExpiration := getOrDefault(data, "expired")
    ch <- prometheus.MustNewConstMetric(collector.licenseInfo, prometheus.GaugeValue, 1, hwkey, kernelVersion, licenseVersion, 
        licenseCryptoVersion, licenseKey, licenseStatus, licenseCreation, licenseExpiration)
    // levels
    levels := gjson.Get(data, "levels").String()
    if levels != "" {
        levelsVal := gjson.Get(data, "levels").Float()
        ch <- prometheus.MustNewConstMetric(collector.levels, prometheus.GaugeValue, levelsVal, hwkey)  
    }
    // total disks
    disks := gjson.Get(data, "disks").String()
    if disks != "" {
        disksVal := gjson.Get(data, "disks").Float()
        ch <- prometheus.MustNewConstMetric(collector.disks, prometheus.GaugeValue, disksVal, hwkey)
    }
    // disks in use
    disksInUse := gjson.Get(data, "disks_in_use").String()
    if disksInUse != "" {
        disksInUseVal := gjson.Get(data, "disks_in_use").Float()
        ch <- prometheus.MustNewConstMetric(collector.disksInUse, prometheus.GaugeValue, disksInUseVal, hwkey)
    }
    return nil
}

func getOrDefault(data, key string) string { 
    value := gjson.Get(data, key)
    if !value.Exists() {
        return "Not found"
    }
    return value.String()
}
