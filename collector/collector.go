package collector

import (
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

var (
	factories              = make(map[string]func() (Collector, error))
	initiatedCollectorsMtx = sync.Mutex{}
	initiatedCollectors    = make(map[string]Collector)
	collectorState         = make(map[string]*bool)
)

type Collector interface {
	Update(ch chan<- prometheus.Metric) error
}

func registerCollector(collector string, isDefaultEnabled bool, factory func() (Collector, error)) {
	var helpDefaultState string
	if isDefaultEnabled {
		helpDefaultState = "enabled"
	} else {
		helpDefaultState = "disabled"
	}

	flagName := fmt.Sprintf("collector.%s", collector)
	flagHelp := fmt.Sprintf("Enable the %s collector (default: %s)", collector, helpDefaultState)
	flagValue := flag.Bool(flagName, isDefaultEnabled, flagHelp)

	collectorState[collector] = flagValue

	factories[collector] = factory
}

type NodeCollector struct {
	Collectors map[string]Collector
}

func NewNodeCollector(filters ...string) (*NodeCollector, error) {
	f := make(map[string]bool)
	for _, filter := range filters {
		enabled, exist := collectorState[filter]
		if !exist {
			return nil, errors.New(fmt.Sprintf("missing collector: %s", filter))
		}
		if !*enabled {
			return nil, errors.New(fmt.Sprintf("disabled collector: %s", filter))
		}
		f[filter] = true
	}
	collectors := make(map[string]Collector)
	initiatedCollectorsMtx.Lock()
	defer initiatedCollectorsMtx.Unlock()

	for key, enabled := range collectorState {
		if !*enabled || (len(f) > 0 && !f[key]) {
			continue
		}
		if collector, ok := initiatedCollectors[key]; ok {
			collectors[key] = collector
		} else {
			collector, err := factories[key]()
			if err != nil {
				return nil, err
			}
			collectors[key] = collector
			initiatedCollectors[key] = collector
		}
	}

	return &NodeCollector{
		Collectors: collectors,
	}, nil
}
