package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	cpuCollectorSubsystem = "cpu"
)

var (
	nodeCPUSecondsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "seconds_total"),
		"Seconds the CPUs spent in each mode.",
		[]string{"cpu", "mode"},
		nil,
	)
	nodeCPULogicCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "logic_count"),
		"CPUs logic count.",
		nil,
		nil,
	)
)
