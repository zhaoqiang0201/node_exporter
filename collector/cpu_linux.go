package collector

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"regexp"
	"strconv"
	"sync"
)

const jumpBackSeconds = 3.0

var (
	flagsInclude         = flag.String("collector.cpu.info.flags-include", "", "Filter the `flags` field in cpuInfo with a value that must be a regular expression")
	bugsInclude          = flag.String("collector.cpu.info.bugs-include", "", "Filter the `bugs` field in cpuInfo with a value that must be a regular expression")
	jumpBackDebugMessage = fmt.Sprintf("CPU Idle counter jumped backwards more than %f seconds, possible hotplug event, resetting CPU stats", jumpBackSeconds)
)

func init() {
	registerCollector("cpu", defaultEnabled, NewCPUCollector)
}

type cpuCollector struct {
	fs            procfs.FS
	cpu           *prometheus.Desc
	cpuLogicCount *prometheus.Desc

	cpuStats      map[int64]procfs.CPUStat
	cpuStatsMutex sync.Mutex

	cpuFlagsIncludeRegexp *regexp.Regexp
	cpuBugsIncludeRegexp  *regexp.Regexp
}

func (c *cpuCollector) compileIncludeFlags(flagsIncludeFlag *string, bugsIncludeFlag *string) error {
	//if (*flagsIncludeFlag != "" || *bugsIncludeFlag != "") && !*enableCPUInfo {
	//	*enableCPUInfo = true
	//	level.Info(c.logger).Log("msg", "--collector.cpu.info has been set to `true` because you set the following flags, like --collector.cpu.info.flags-include and --collector.cpu.info.bugs-include")
	//}
	var err error
	if *flagsIncludeFlag != "" {
		c.cpuFlagsIncludeRegexp, err = regexp.Compile(*flagsIncludeFlag)
		if err != nil {
			return err
		}
	}
	if *bugsIncludeFlag != "" {
		c.cpuBugsIncludeRegexp, err = regexp.Compile(*bugsIncludeFlag)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewCPUCollector() (Collector, error) {
	fs, err := procfs.NewFS(*procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open procfs: %w", err)
	}

	//sysfs, err := sysfs.NewFS(*sysPath)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to open sysfs: %w", err)
	//}

	c := &cpuCollector{
		fs:            fs,
		cpu:           nodeCPUSecondsDesc,
		cpuLogicCount: nodeCPULogicCount,
		cpuStats:      make(map[int64]procfs.CPUStat),
	}

	err = c.compileIncludeFlags(flagsInclude, bugsInclude)
	if err != nil {
		return nil, fmt.Errorf("fail to compile --collector.cpu.info.flags-include and --collector.cpu.info.bugs-include, the values of them must be regular expressions: %w", err)
	}
	return c, nil
}

func (c *cpuCollector) Update(ch chan<- prometheus.Metric) error {
	return c.updateStat(ch)
}

func (c *cpuCollector) updateStat(ch chan<- prometheus.Metric) error {
	stat, err := c.fs.Stat()
	if err != nil {
		return err
	}
	c.updateCPUStats(stat.CPU)

	c.cpuStatsMutex.Lock()
	defer c.cpuStatsMutex.Unlock()
	for cpuID, cpuStat := range c.cpuStats {
		cpuNum := strconv.Itoa(int(cpuID))
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.User, cpuNum, "user")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Nice, cpuNum, "nice")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.System, cpuNum, "system")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Idle, cpuNum, "idle")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Iowait, cpuNum, "iowait")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.IRQ, cpuNum, "irq")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.SoftIRQ, cpuNum, "softirq")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Steal, cpuNum, "steal")
	}
	ch <- prometheus.MustNewConstMetric(nodeCPULogicCount, prometheus.GaugeValue, float64(len(c.cpuStats)))
	return nil
}

func (c *cpuCollector) updateCPUStats(newStats map[int64]procfs.CPUStat) {
	c.cpuStatsMutex.Lock()
	defer c.cpuStatsMutex.Unlock()
	for i, n := range newStats {
		cpuStats := c.cpuStats[i]
		if (cpuStats.Idle - n.Idle) >= jumpBackSeconds {
			log.Debug().Msgf("%s. cpu: %d, old_value: %v, new_value: %v", jumpBackDebugMessage, i, cpuStats.Idle, n.Idle)
			cpuStats = procfs.CPUStat{}
		}

		if n.Idle >= cpuStats.Idle {
			cpuStats.Idle = n.Idle
		} else {
			log.Debug().Msgf("CPU Idle counter jumped backwards. cpu: %d, old_value: %v, new_value: %v", i, cpuStats.Idle, n.Idle)
		}

		if n.User >= cpuStats.User {
			cpuStats.User = n.User
		} else {
			log.Debug().Msgf("CPU User counter jumped backwards. cpu: %d, old_value: %v, new_value: %v", i, cpuStats.User, n.User)
		}

		if n.Nice >= cpuStats.Nice {
			cpuStats.Nice = n.Nice
		} else {
			log.Debug().Msgf("CPU Nice counter jumped backwards. cpu: %d, old_value: %v, new_value: %v", i, cpuStats.Nice, n.Nice)
		}

		if n.System >= cpuStats.System {
			cpuStats.System = n.System
		} else {
			log.Debug().Msgf("CPU System counter jumped backwards. cpu: %d, old_value: %v, new_value: %v", i, cpuStats.System, n.System)
		}

		if n.Iowait >= cpuStats.Iowait {
			cpuStats.Iowait = n.Iowait
		} else {
			log.Debug().Msgf("CPU Iowait counter jumped backwards. cpu: %d, old_value: %v, new_value: %v", i, cpuStats.Iowait, n.Iowait)
		}

		if n.IRQ >= cpuStats.IRQ {
			cpuStats.IRQ = n.IRQ
		} else {
			log.Debug().Msgf("CPU IRQ counter jumped backwards. cpu: %d, old_value: %v, new_value: %v", i, cpuStats.IRQ, n.IRQ)
		}

		if n.SoftIRQ >= cpuStats.SoftIRQ {
			cpuStats.SoftIRQ = n.SoftIRQ
		} else {
			log.Debug().Msgf("CPU SoftIRQ counter jumped backwards. cpu: %d, old_value: %v, new_value: %v", i, cpuStats.SoftIRQ, n.SoftIRQ)
		}

		if n.Steal >= cpuStats.Steal {
			cpuStats.Steal = n.Steal
		} else {
			log.Debug().Msgf("CPU Steal counter jumped backwards. cpu: %d, old_value: %v, new_value: %v", i, cpuStats.Steal, n.Steal)
		}

		if n.Guest >= cpuStats.Guest {
			cpuStats.Guest = n.Guest
		} else {
			log.Debug().Msgf("CPU Guest counter jumped backwards. cpu: %d, old_value: %v, new_value: %v", i, cpuStats.Guest, n.Guest)
		}

		if n.GuestNice >= cpuStats.GuestNice {
			cpuStats.GuestNice = n.GuestNice
		} else {
			log.Debug().Msgf("CPU GuestNice counter jumped backwards. cpu: %d, old_value: %v, new_value: %v", i, cpuStats.GuestNice, n.GuestNice)
		}

		c.cpuStats[i] = cpuStats
	}
	if len(newStats) != len(c.cpuStats) {
		onlineCPUIds := maps.Keys(newStats)
		maps.DeleteFunc(c.cpuStats, func(key int64, item procfs.CPUStat) bool {
			return !slices.Contains(onlineCPUIds, key)
		})
	}
}
