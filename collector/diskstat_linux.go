package collector

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/blockdevice"
)

const (
	secondsPerTick = 1.0 / 1000.0
)
const (
	diskstatsDefaultIgnoredDevices = "^(ram|loop|fd|(h|s|v|xv)d[a-z]|nvme\\d+n\\d+p)\\d+$"
)

type diskstatsCollector struct {
	deviceFilter         deviceFilter
	fs                   blockdevice.FS
	readTimeSecondsDesc  *prometheus.Desc
	writeTimeSecondsDesc *prometheus.Desc
}

func init() {
	registerCollector("diskstats", defaultEnabled, NewDiskstatsCollector)
}

// NewDiskstatsCollector returns a new Collector exposing disk device stats.
// Docs from https://www.kernel.org/doc/Documentation/iostats.txt
func NewDiskstatsCollector() (Collector, error) {
	//var diskLabelNames = []string{"device"}
	fs, err := blockdevice.NewFS(*procPath, *sysPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sysfs: %w", err)
	}

	deviceFilter, err := newDiskstatsDeviceFilter()
	if err != nil {
		return nil, fmt.Errorf("failed to parse device filter flags: %w", err)
	}

	collector := diskstatsCollector{
		deviceFilter:         deviceFilter,
		fs:                   fs,
		readTimeSecondsDesc:  readTimeSecondsDesc,
		writeTimeSecondsDesc: writeTimeSecondsDesc,
	}

	return &collector, nil
}

func (c diskstatsCollector) Update(ch chan<- prometheus.Metric) error {
	diskStats, err := c.fs.ProcDiskstats()
	if err != nil {
		return fmt.Errorf("couldn't get diskstats: %w", err)
	}

	for _, stats := range diskStats {
		dev := stats.DeviceName
		if c.deviceFilter.ignored(dev) {
			continue
		}

		ch <- prometheus.MustNewConstMetric(readTimeSecondsDesc, prometheus.CounterValue, float64(stats.ReadTicks)*float64(secondsPerTick), dev)
		ch <- prometheus.MustNewConstMetric(writeTimeSecondsDesc, prometheus.CounterValue, float64(stats.WriteTicks)*float64(secondsPerTick), dev)
	}
	return nil
}
