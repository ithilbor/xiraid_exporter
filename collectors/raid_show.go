// This is the collector of the xiraid raid information.
// The equivalent of this collector from CLI is:
// 'xicli raid show'

package collector

import (
	// Go
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"
	// Xiraid exporter
	xrprotos "github.com/ironcub3/xiraid_exporter/protos"
	// Prometheus
	"github.com/prometheus/client_golang/prometheus"
	// Gjson
	"github.com/tidwall/gjson"
)

type raidShowCollector struct {
	raidInfo      *prometheus.Desc
	state         *prometheus.Desc
	stateDetail   *prometheus.Desc
	active        *prometheus.Desc
	config        *prometheus.Desc
	blockSize     *prometheus.Desc
	groupSize     *prometheus.Desc
	stripSize     *prometheus.Desc
	size          *prometheus.Desc
	memoryUsage   *prometheus.Desc
	deviceState   *prometheus.Desc
	logger        *slog.Logger
	xrClient      xrprotos.XRAIDServiceClient
}

//TODO: use the logger

func init() {
	registerCollector("raid_show", defaultEnabled, NewRaidShowCollector)
}

func NewRaidShowCollector(logger *slog.Logger, xrClient xrprotos.XRAIDServiceClient) (Collector, error) {
	namespace := "xiraid"
	return &raidShowCollector{
		// Static information of the RAID
		raidInfo: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "raid", "info"),
			"Informations about the RAID",
			[]string{"raid_name", "uuid", "level", "sparepool_name"}, nil,
		),
		state: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "raid", "state"),
			"RAID status (1 = online, 0 = offline, 2 = all other states)",
			[]string{"raid_name", "uuid", "state"}, nil,
		),
		stateDetail: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "raid", "state_detail"),
			"Details about the RAID status",
			[]string{"raid_name", "uuid", "state_detail"}, nil,
		),
		active: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "raid", "active"),
			"RAID block device active (1 = True, 0 = False)",
			[]string{"raid_name", "uuid"}, nil,
		),
		config: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "raid", "config"),
			"RAID presence in the configuration file (1 = True, 0 = False)",
			[]string{"raid_name", "uuid"}, nil,
		),
		blockSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "raid", "block_size"),
			"RAID block size",
			[]string{"raid_name", "uuid"}, nil,
		),
		groupSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "raid", "group_size"),
			"RAID group size",
			[]string{"raid_name", "uuid"}, nil,
		),
		stripSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "raid", "strip_size"),
			"RAID strip size",
			[]string{"raid_name", "uuid"}, nil,
		),
		size: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "raid", "size_gib"),
			"Size of the RAID in GiB",
			[]string{"raid_name", "uuid"}, nil,
		),
		memoryUsage: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "raid", "memory_usage_mb"),
			"Memory usage of the RAID in MB (0 = no memory limit setted)",
			[]string{"raid_name", "uuid"}, nil,
		),
		deviceState: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "raid", "device_state"),
			"State of each device in the RAID array (1 = online, 0 = offline)",
			[]string{"raid_name", "uuid", "device", "device_serial"}, nil,
		),
		logger:   logger,
		xrClient: xrClient,
	}, nil
}

// fetchXiraidData retrieves and parses RAID data using gjson.
func fetchXiraidData(xrClient xrprotos.XRAIDServiceClient) (gjson.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	raidShowRequest := &xrprotos.RaidShow{}
	r1, err := xrClient.RaidShow(ctx, raidShowRequest)
	if err != nil {
		return gjson.Result{}, fmt.Errorf("could not get RAID entries: %w", err)
	}
	return gjson.Parse(r1.GetMessage()), nil
}

// Update gathers and exports metrics.
func (collector *raidShowCollector) Update(ch chan<- prometheus.Metric) error {
	data, err := fetchXiraidData(collector.xrClient)
	if err != nil {
		return fmt.Errorf("could not get RAID entries: %w", err)
	}
	data.ForEach(func(key, value gjson.Result) bool {
		// Static data
		raidName := key.String()
		uuid := value.Get("uuid").String()
		level := value.Get("level").String()
		sparepoolName := value.Get("sparepool").String()
		// RAID info ( static data )
		ch <- prometheus.MustNewConstMetric(collector.raidInfo, prometheus.GaugeValue,
			 1, raidName, uuid, level, sparepoolName)
		// RAID state
		// Check the array length
		mainStateArray := value.Get("state").Array()
		if len(mainStateArray) > 0 {
			mainState := mainStateArray[0].String()
			if mainState == "" {
				mainState = "unknown"
			}
			stateVal := 0.0
			if mainState == "online" {
			    stateVal = 1.0
			} else if mainState == "offline" {
			    stateVal = 0.0
			} else {
				// for all other states
			    stateVal = 2.0
			}
			ch <- prometheus.MustNewConstMetric(collector.state, prometheus.GaugeValue, stateVal, raidName, uuid, mainState)
		} else {
			collector.logger.Warn("Array length < 0")
		}
		// RAID state detail
		// Here we take the state detail from the state array
		stateDetailArray := value.Get("state").Array()
		if len(stateDetailArray) > 1 {
			stateDetail := value.Get("state").Array()[1].String()
			if stateDetail == "" {
				stateDetail = "unknown"
			}
			ch <- prometheus.MustNewConstMetric(collector.stateDetail, prometheus.GaugeValue, 1, raidName, uuid, stateDetail)
		}
		// Active status
		active := 0.0
		if value.Get("active").Bool() {
			active = 1.0
		}
		ch <- prometheus.MustNewConstMetric(collector.active, prometheus.GaugeValue, active, raidName, uuid)
		// Config status
		config := 0.0
		if value.Get("config").Bool() {
			config = 1.0
		}
		ch <- prometheus.MustNewConstMetric(collector.config, prometheus.GaugeValue, config, raidName, uuid)
		// Block size, group size, and strip size
		ch <- prometheus.MustNewConstMetric(collector.blockSize, prometheus.GaugeValue, value.Get("block_size").Float(), raidName, uuid)
		ch <- prometheus.MustNewConstMetric(collector.groupSize, prometheus.GaugeValue, value.Get("group_size").Float(), raidName, uuid)
		ch <- prometheus.MustNewConstMetric(collector.stripSize, prometheus.GaugeValue, value.Get("strip_size").Float(), raidName, uuid)
		// Size in GiB
		size := value.Get("size").String()
		if sizeVal, err := parseSize(size); err == nil {
			ch <- prometheus.MustNewConstMetric(collector.size, prometheus.GaugeValue, sizeVal, raidName, uuid)
		}
		// Memory usage (handle "-" value gracefully)
		// TODO: scrivi la funzione parseMem
		memoryUsage := value.Get("memory_usage_mb").String()
		if memVal, err := strconv.ParseFloat(memoryUsage, 64); err != nil {
			ch <- prometheus.MustNewConstMetric(collector.memoryUsage, prometheus.GaugeValue, 0, raidName, uuid)
		} else {
			ch <- prometheus.MustNewConstMetric(collector.memoryUsage, prometheus.GaugeValue, memVal, raidName, uuid)
		}
		// State of each device in the RAID array
		serials := value.Get("serials").Array()
		devices := value.Get("devices").Array()
		for i, device := range devices {
		    deviceSerial := "unknown_serial"
			if i < len(serials) {
				deviceSerial = serials[i].String()
			}
		    deviceName := device.Array()[1].String()
		    state := device.Array()[2].Array()[0].String()
		    stateVal := 0.0
		    if state == "online" {
		        stateVal = 1.0
		    }
		    ch <- prometheus.MustNewConstMetric(
		        collector.deviceState, prometheus.GaugeValue, stateVal, raidName, 
				uuid, deviceName, deviceSerial)
		}
		return true
	})
	return nil
}

func parseSize(size string) (float64, error) {
	sizeVal := 0.0
	_, err := fmt.Sscanf(size, "%f GiB", &sizeVal)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size: %w", err)
	} else if size == "-" {
		return 0, nil
	}
	return sizeVal, nil
}