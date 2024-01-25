package collector

import (
	"flag"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

const (
	diskSubsystem = "disk"
)

var (
	diskstatsDeviceExclude = flag.String(
		"collector.diskstats.device-exclude",
		diskstatsDefaultIgnoredDevices,
		"Regexp of diskstats devices to exclude (mutually exclusive to device-include).",
	)
	diskstatsDeviceInclude = flag.String(
		"collector.diskstats.device-include",
		"",
		"Regexp of diskstats devices to include (mutually exclusive to device-exclude).")
)

var (
	diskLabelNames      = []string{"device"}
	readTimeSecondsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, diskSubsystem, "read_time_seconds_total"),
		"The total number of seconds spent by all reads.",
		diskLabelNames,
		nil,
	)

	writeTimeSecondsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, diskSubsystem, "write_time_seconds_total"),
		"This is the total number of seconds spent by all writes.",
		diskLabelNames,
		nil,
	)
)

func newDiskstatsDeviceFilter() (deviceFilter, error) {

	if *diskstatsDeviceExclude != "" && *diskstatsDeviceInclude != "" {
		return deviceFilter{}, errors.New("device-exclude & device-include are mutually exclusive")
	}

	if *diskstatsDeviceExclude != "" {
		log.Info().Msgf("Parsed flag --collector.diskstats.device-exclude: %s", *diskstatsDeviceExclude)
	}

	if *diskstatsDeviceInclude != "" {
		log.Info().Msgf("Parsed Flag --collector.diskstats.device-include: %s", *diskstatsDeviceInclude)
	}

	return newDeviceFilter(*diskstatsDeviceExclude, *diskstatsDeviceInclude), nil
}
