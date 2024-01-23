package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	promcollector "github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/zhaoqiang0201/node_exporter/collector"
	"github.com/zhaoqiang0201/node_exporter/version"
	"gopkg.in/natefinch/lumberjack.v2"
	"net/http"
	"os/user"
	"runtime"
	"runtime/debug"
)

type handler struct {
	unfilteredHandler       http.Handler
	exporterMetricsRegistry *prometheus.Registry
	includeExporterMetrics  bool
	maxRequests             int
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
			promcollector.NewGoCollector(),
		)
	}

	return h
}

func (h *handler) innerHandler(filters ...string) (http.Handler, error) {
	nc, err := collector.NewNodeCollector(filters...)
	if err != nil {
		return
	}
}

var cmd = &cobra.Command{
	Use:   "node_exporter",
	Short: "node_exporter 1s scrape",
	RunE: func(cmd *cobra.Command, args []string) error {
		buildInfo, ok := debug.ReadBuildInfo()
		fmt.Println(ok)
		bytes, err := json.MarshalIndent(buildInfo, "", "\t")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", bytes)
		return nil
	},
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

func main() {
	cmdPflagSet := cmd.PersistentFlags()
	cmdPflagSet.AddGoFlagSet(flag.CommandLine)
	var (
		logpath                = cmdPflagSet.String("log.path", "node_exporter.log", "日志路径")
		logMaxSize             = cmdPflagSet.Int("log.maxSize", 10, "日志轮转文件大小")
		logMaxBackups          = cmdPflagSet.Int("log.maxBackups", 3, "日志轮转文件保留")
		metricsPath            = cmdPflagSet.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		maxRequests            = cmdPflagSet.Int("web.max-requests", 40, "Maximum number of parallel scrape requests. Use 0 to disable.")
		maxProcs               = cmdPflagSet.Int("runtime.gomaxprocs", 1, "The target number of CPUs Go will run on (GOMAXPROCS)")
		disableExporterMetrics = cmdPflagSet.Bool("web.disable-exporter-metrics", false, "Exclude metrics about the exporter itself (promhttp_*, process_*, go_*).")

		//webListenAddresses = cmdPflagSet.String("web.listen-address", ":9100", "Addresses on which to expose metrics and web interface. Repeatable for multiple addresses.")
	)
	initLog(*logpath, *logMaxSize, *logMaxBackups)
	log.Info().Msgf("Starting node_exporter version %s", version.Info())
	log.Info().Msgf("Build context build_context " + version.BuildContext())
	if user, err := user.Current(); err == nil && user.Uid == "0" {
		log.Warn().Msg("Node Exporter is running as root user. This exporter is designed to run as unprivileged user, root is not required.")
	}
	runtime.GOMAXPROCS(*maxProcs)
	log.Info().Msgf("Go MAXPROCS=%d", runtime.GOMAXPROCS(0))

	http.Handle(*metricsPath, newHandler(!*disableExporterMetrics, *maxRequests))

	if err := cmd.Execute(); err != nil {
		log.Panic().Err(err).Send()
	}
}

func initLog(path string, maxSize, maxBackups int) {
	logFD := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    maxSize,
		MaxAge:     0,
		MaxBackups: maxBackups,
		LocalTime:  false,
		Compress:   false,
	}

	log.Logger = log.With().CallerWithSkipFrameCount(2).Logger().Output(logFD)
}
