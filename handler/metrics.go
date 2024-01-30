package handler

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	promcollector "github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"github.com/zhaoqiang0201/node_exporter/collector"
	"github.com/zhaoqiang0201/node_exporter/version"
	stdlog "log"
	"net/http"
	"os"
	"sort"
)

type handler struct {
	unfilteredHandler       http.Handler
	exporterMetricsRegistry *prometheus.Registry
	includeExporterMetrics  bool
	maxRequests             int
}

func MetricsHandler(includeExporterMetrics bool, maxRequests int) *handler {
	h := &handler{
		exporterMetricsRegistry: prometheus.NewRegistry(),
		includeExporterMetrics:  includeExporterMetrics,
		maxRequests:             maxRequests,
	}
	if h.includeExporterMetrics {
		h.exporterMetricsRegistry.MustRegister(
			promcollector.NewProcessCollector(promcollector.ProcessCollectorOpts{}),
			promcollector.NewGoCollector(
				promcollector.WithGoCollectorRuntimeMetrics(promcollector.MetricsScheduler),
			),
		)
	}

	if innerHandler, err := h.innerHandler(); err != nil {
		panic(fmt.Sprintf("Couldn't create metrics handler: %s", err))
	} else {
		h.unfilteredHandler = innerHandler
	}

	return h
}

func (h *handler) innerHandler(filters ...string) (http.Handler, error) {
	nc, err := collector.NewNodeCollector(filters...)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("couldn't create collector. %v", err))
	}

	if len(filters) == 0 {
		log.Info().Msg("Enabled collectors")
		collectors := []string{}
		for n := range nc.Collectors {
			collectors = append(collectors, n)
		}
		sort.Strings(collectors)
		for _, c := range collectors {
			log.Info().Str("collector", c).Msg("Enable")
		}
	}
	r := prometheus.NewRegistry()
	r.MustRegister(version.NewCollector("node_exporter"))
	if err := r.Register(nc); err != nil {
		return nil, errors.New(fmt.Sprintf("couldn't register node collector: %s", err))
	}

	var handler http.Handler
	if h.includeExporterMetrics {
		handler = promhttp.HandlerFor(
			prometheus.Gatherers{h.exporterMetricsRegistry, r},
			promhttp.HandlerOpts{
				ErrorLog:            stdlog.New(os.Stdout, "", 0),
				ErrorHandling:       promhttp.ContinueOnError,
				Registry:            h.exporterMetricsRegistry,
				MaxRequestsInFlight: h.maxRequests,
			},
		)
		handler = promhttp.InstrumentMetricHandler(
			h.exporterMetricsRegistry, handler,
		)
	} else {
		handler = promhttp.HandlerFor(
			r,
			promhttp.HandlerOpts{
				ErrorLog:            stdlog.New(os.Stdout, "", 0),
				ErrorHandling:       promhttp.ContinueOnError,
				Registry:            h.exporterMetricsRegistry,
				MaxRequestsInFlight: h.maxRequests,
			},
		)
	}
	return handler, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filters := r.URL.Query()["collect[]"]
	log.Debug().Msgf("collect query: filters %v", filters)

	if len(filters) == 0 {
		h.unfilteredHandler.ServeHTTP(w, r)
		return
	}
	filterdHandler, err := h.innerHandler(filters...)
	if err != nil {
		log.Warn().Err(err).Msg("Couldn't create filtered metrics handler")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Couldn't create filtered metrics handler: %s", err)))
		return
	}
	filterdHandler.ServeHTTP(w, r)
}
