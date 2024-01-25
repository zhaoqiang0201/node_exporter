package collector

import (
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"sync"
)

func init() {
	registerCollector("netdev", defaultEnabled, NewNetDevCollector)
}

var (
	netdevDeviceInclude = flag.String("collector.netdev.device-include", "", "Regexp of net devices to include (mutually exclusive to device-exclude).")
	netdevDeviceExclude = flag.String("collector.netdev.device-exclude", "", "Regexp of net devices to exclude (mutually exclusive to device-include).")
)

type netDevStats map[string]map[string]uint64

type netDevCollector struct {
	subsystem        string
	deviceFilter     deviceFilter
	metricDescsMutex sync.Mutex
	metricDescs      map[string]*prometheus.Desc
}

func NewNetDevCollector() (Collector, error) {
	if *netdevDeviceExclude != "" && *netdevDeviceInclude != "" {
		return nil, errors.New("device-exclude & device-include are mutually exclusive")
	}
	if *netdevDeviceExclude != "" {
		log.Info().Msgf("Parsed flag --collector.netdev.device-exclude = %v", *netdevDeviceExclude)
	}

	if *netdevDeviceInclude != "" {
		log.Info().Msgf("Parsed Flag --collector.netdev.device-include = %v", *netdevDeviceInclude)
	}
	return &netDevCollector{
		subsystem:    "network",
		deviceFilter: newDeviceFilter(*netdevDeviceExclude, *netdevDeviceInclude),
		metricDescs:  map[string]*prometheus.Desc{},
	}, nil
}

func (c *netDevCollector) Update(ch chan<- prometheus.Metric) error {
	netDev, err := getNetDevStats(&c.deviceFilter)
	if err != nil {
		return fmt.Errorf("couldn't get netstats: %w", err)
	}

	for dev, devStats := range netDev {
		for key, value := range devStats {
			desc := c.metricDesc(key)
			ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, float64(value), dev)
		}
	}
	return nil
}

func (c *netDevCollector) metricDesc(key string) *prometheus.Desc {
	c.metricDescsMutex.Lock()
	defer c.metricDescsMutex.Unlock()

	if _, ok := c.metricDescs[key]; !ok {
		c.metricDescs[key] = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, c.subsystem, key+"_total"),
			fmt.Sprintf("Network device statistic %s.", key),
			[]string{"device"},
			nil,
		)
	}

	return c.metricDescs[key]
}
