// This is the collector of the xiraid raid information.
// The equivalent of this collector from CLI is:
// 'xicli raid show'

package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
	// Xiraid exporter
	xrprotos "github.com/ironcub3/xiraid_exporter/protos"
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
            []string{"hwkey", "license_key", "license_version", "license_crypto_version",
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
    // Convert the raw data into a JSON string
    jsonData, err := textToJSON(data)
    if err != nil {
        return fmt.Errorf("could not convert data to JSON: %w", err)
    }
    // Hardware key
    hwkey := getOrDefault(jsonData, "hwkey")
    // license info
    licenseKey := getOrDefault(jsonData, "license_key")
    licenseVersion := getOrDefault(jsonData,"version")
    licenseCryptoVersion := getOrDefault(jsonData, "crypto_version")
    licenseStatus := getOrDefault(jsonData, "status")
    licenseCreation := getOrDefault(jsonData, "created")
    licenseExpiration := getOrDefault(jsonData, "expired")
    ch <- prometheus.MustNewConstMetric(collector.licenseInfo, prometheus.GaugeValue, 1, hwkey, licenseVersion, 
        licenseCryptoVersion, licenseKey, licenseStatus, licenseCreation, licenseExpiration)
    //levels
    levels := gjson.Get(jsonData, "levels").String()
    if levels != "" {
        levelsVal := gjson.Get(jsonData, "levels").Float()
        ch <- prometheus.MustNewConstMetric(collector.levels, prometheus.GaugeValue, levelsVal, hwkey)  
    }
    // total disks
    disks := gjson.Get(jsonData, "disks").String()
    if disks != "" {
        disksVal := gjson.Get(jsonData, "disks").Float()
        ch <- prometheus.MustNewConstMetric(collector.disks, prometheus.GaugeValue, disksVal, hwkey)
    }
    // disks in use
    disksInUse := gjson.Get(jsonData, "disks_in_use").String()
    if disksInUse != "" {
        disksInUseVal := gjson.Get(jsonData, "disks_in_use").Float()
        ch <- prometheus.MustNewConstMetric(collector.disksInUse, prometheus.GaugeValue, disksInUseVal, hwkey)
    }
    return nil
}

// textToJSON function that reads lines and creates a JSON structure
func textToJSON(data string) (string, error) {
    jsonMap := make(map[string]string)
    lines := strings.Split(data, "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if strings.Contains(line, ":") {
            parts := strings.SplitN(line, ":", 2)
            key := strings.TrimSpace(parts[0])
            value := strings.TrimSpace(parts[1])
            // Convert key to lowercase and replace spaces with underscores
            formattedKey := strings.ToLower(strings.ReplaceAll(key, " ", "_"))
            jsonMap[formattedKey] = value
        }
    }
    jsonData, err := json.Marshal(jsonMap)
    if err != nil {
        return "", fmt.Errorf("could not marshal to JSON: %w", err)
    }
    return string(jsonData), nil
}

func getOrDefault(jsonData, key string) string { 
    value := gjson.Get(jsonData, key)
    if !value.Exists() {
        return ""
    }
    return value.String()
}
