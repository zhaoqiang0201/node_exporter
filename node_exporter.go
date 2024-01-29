package main

import (
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	promcollector "github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/zhaoqiang0201/node_exporter/collector"
	"github.com/zhaoqiang0201/node_exporter/version"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	stdlog "log"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"sort"
	"time"
)

type handler struct {
	unfilteredHandler       http.Handler
	exporterMetricsRegistry *prometheus.Registry
	includeExporterMetrics  bool
	maxRequests             int
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

func newHandler(includeExporterMetrics bool, maxRequests int) *handler {
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

var cmd = &cobra.Command{
	Use:   "node_exporter",
	Short: "node_exporter 1s scrape",
	RunE: func(cmd *cobra.Command, args []string) error {
		f := cmd.PersistentFlags().Lookup("web.listen-address")
		if f == nil {
			return errors.New("web.listen-address flags is nil")
		}
		log.Info().Msgf("node_exporter listen %s", f.Value.String())
		server := &http.Server{
			Addr:         f.Value.String(),
			ReadTimeout:  time.Second * 5,
			WriteTimeout: time.Second * 5,
			IdleTimeout:  time.Second * 10,
		}

		return server.ListenAndServe()
	},
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

func main() {
	cmdPflagSet := cmd.PersistentFlags()
	cmdPflagSet.AddGoFlagSet(flag.CommandLine)
	var (
		logpath                = cmdPflagSet.String("log.path", "", "日志路径")
		logMaxSize             = cmdPflagSet.Int("log.maxSize", 10, "日志轮转文件大小")
		logMaxBackups          = cmdPflagSet.Int("log.maxBackups", 3, "日志轮转文件保留")
		metricsPath            = cmdPflagSet.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		maxRequests            = cmdPflagSet.Int("web.max-requests", 40, "Maximum number of parallel scrape requests. Use 0 to disable.")
		maxProcs               = cmdPflagSet.Int("runtime.gomaxprocs", 1, "The target number of CPUs Go will run on (GOMAXPROCS)")
		disableExporterMetrics = cmdPflagSet.Bool("web.disable-exporter-metrics", false, "Exclude metrics about the exporter itself (promhttp_*, process_*, go_*).")
	)
	cmdPflagSet.String("web.listen-address", ":9111", "Addresses on which to expose metrics and web interface. Repeatable for multiple addresses.")

	initLog(*logpath, *logMaxSize, *logMaxBackups)
	log.Info().Msgf("Starting node_exporter version %v", version.Info())
	log.Info().Msgf("Build context build_context " + version.BuildContext())
	if user, err := user.Current(); err == nil && user.Uid == "0" {
		log.Warn().Msg("Node Exporter is running as root user. This exporter is designed to run as unprivileged user, root is not required.")
	}
	runtime.GOMAXPROCS(*maxProcs)
	log.Info().Msgf("Go MAXPROCS=%d", runtime.GOMAXPROCS(0))

	http.Handle(*metricsPath, newHandler(!*disableExporterMetrics, *maxRequests))
	if *metricsPath != "/" {
		landingConfig := web.LandingConfig{
			Name:        "Node Exporter",
			Description: "Prometheus Node Exporter",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{
					Address: *metricsPath,
					Text:    "Metrics",
				},
			},
		}
		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			log.Error().Err(err).Send()
			os.Exit(1)
		}
		http.Handle("/", landingPage)
		http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("TONG"))
		})
	}

	if err := cmd.Execute(); err != nil {
		log.Panic().Err(err).Send()
	}
}

func initLog(path string, maxSize, maxBackups int) {
	var logFD io.Writer
	if path == "" {
		logFD = os.Stdout
	} else {
		logFD = &lumberjack.Logger{
			Filename:   path,
			MaxSize:    maxSize,
			MaxAge:     0,
			MaxBackups: maxBackups,
			LocalTime:  false,
			Compress:   false,
		}
	}

	log.Logger = log.With().CallerWithSkipFrameCount(2).Logger().Output(logFD)
}
