package collector

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"os"
	"strconv"
	"strings"
)

type loadavgCollector struct {
	metric []typedDesc
}

func init() {
	registerCollector("loadavg", defaultEnabled, NewLoadavgCollector)
}

func NewLoadavgCollector() (Collector, error) {
	return &loadavgCollector{
		metric: []typedDesc{
			{prometheus.NewDesc(namespace+"_load1", "1m load average.", nil, nil), prometheus.GaugeValue},
			{prometheus.NewDesc(namespace+"_load5", "5m load average.", nil, nil), prometheus.GaugeValue},
			{prometheus.NewDesc(namespace+"_load15", "5m load average.", nil, nil), prometheus.GaugeValue},
		},
	}, nil
}

func (c *loadavgCollector) Update(ch chan<- prometheus.Metric) error {
	loads, err := getLoad()
	if err != nil {
		return fmt.Errorf("couldn't get load: %w", err)
	}
	for i, load := range loads {
		log.Debug().Msgf("return load. index: %d, load: %v", i, load)
		ch <- c.metric[i].mustNewConstMetric(load)
	}
	return nil
}

func getLoad() (loads []float64, err error) {
	data, err := os.ReadFile(procFilePath("loadavg"))
	if err != nil {
		return nil, err
	}
	loads, err = parseLoad(string(data))
	if err != nil {
		return nil, err
	}
	return loads, nil
}

func parseLoad(data string) (loads []float64, err error) {
	loads = make([]float64, 3)
	parts := strings.Fields(data)
	if len(parts) < 3 {
		return nil, fmt.Errorf("unexpected content in %s", procFilePath("loadavg"))
	}
	for i, load := range parts[0:3] {
		loads[i], err = strconv.ParseFloat(load, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse load '%s': %w", load, err)
		}
	}
	return loads, nil
}
