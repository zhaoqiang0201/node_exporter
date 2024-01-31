package collector

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

type processCollector struct {
	fs      procfs.FS
	pidUsed *prometheus.Desc
	pidMax  *prometheus.Desc
}

func init() {
	registerCollector("processes", defaultEnabled, NewProcessStatCollector)
}

func NewProcessStatCollector() (Collector, error) {
	fs, err := procfs.NewFS(*procPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open procfs: %v", *procPath)
	}
	subsystem := "processes"
	return &processCollector{
		fs: fs,
		pidUsed: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "pids"),
			"Number of PIDs", nil, nil,
		),
		// cat /proc/sys/kernel/pid_max
		//pidMax: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "max_processes"),
		//	"Number of max PIDs limit", nil, nil,
		//),
	}, nil
}

func (c *processCollector) Update(ch chan<- prometheus.Metric) error {
	procs, err := c.fs.AllProcs()
	if err != nil {
		return errors.Wrapf(err, "unable to list all processes")
	}
	ch <- prometheus.MustNewConstMetric(c.pidUsed, prometheus.GaugeValue, float64(len(procs)))
	return nil
}
