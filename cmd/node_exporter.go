package cmd

import (
	"flag"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/zhaoqiang0201/node_exporter/handler"
	locallog "github.com/zhaoqiang0201/node_exporter/log"
	"github.com/zhaoqiang0201/node_exporter/version"
	"net/http"
	"os/user"
	"runtime"
	"time"
)

var (
	logpath                string
	logMaxSize             int
	logMaxBackups          int
	metricsPath            string
	maxRequests            int
	maxProcs               int
	disableExporterMetrics bool
	webAddr                string
	versions               bool
)

var cmd = &cobra.Command{
	Use:   "node_exporter",
	Short: "node_exporter 1s scrape",
	RunE: func(cmd *cobra.Command, args []string) error {
		if versions {
			version.Print("node_exporter_1s")
			return nil
		}
		return run()
	},
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

func init() {
	cobra.OnInitialize(locallog.InitZeroLog(logpath, logMaxSize, logMaxBackups))

	cmdPflagSet := cmd.PersistentFlags()
	cmdPflagSet.StringVar(&logpath, "log.path", "", "日志路径")
	cmdPflagSet.IntVar(&logMaxSize, "log.maxSize", 10, "日志轮转文件大小")
	cmdPflagSet.IntVar(&logMaxBackups, "log.maxBackups", 3, "日志轮转文件保留")
	cmdPflagSet.StringVar(&metricsPath, "web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	cmdPflagSet.IntVar(&maxRequests, "web.max-requests", 40, "Maximum number of parallel scrape requests. Use 0 to disable.")
	cmdPflagSet.IntVar(&maxProcs, "runtime.gomaxprocs", 1, "The target number of CPUs Go will run on (GOMAXPROCS)")
	cmdPflagSet.BoolVar(&disableExporterMetrics, "web.disable-exporter-metrics", false, "Exclude metrics about the exporter itself (promhttp_*, process_*, go_*).")
	cmdPflagSet.StringVar(&webAddr, "web.listen-address", ":9111", "Addresses on which to expose metrics and web interface. Repeatable for multiple addresses.")
	cmdPflagSet.BoolVarP(&versions, "version", "v", false, "node版本信息")
	cmdPflagSet.AddGoFlagSet(flag.CommandLine)
}

func Execute() error {
	return cmd.Execute()
}

func run() error {
	log.Info().Msgf("Starting node_exporter version %v", version.Info())
	log.Info().Msgf("Build context build_context " + version.BuildContext())
	if user, err := user.Current(); err == nil && user.Uid == "0" {
		log.Warn().Msg("Node Exporter is running as root user. This exporter is designed to run as unprivileged user, root is not required.")
	}
	runtime.GOMAXPROCS(maxProcs)
	log.Info().Msgf("Go MAXPROCS=%d", runtime.GOMAXPROCS(0))

	http.Handle(metricsPath, handler.MetricsHandler(!disableExporterMetrics, maxRequests))
	http.HandleFunc("/ping", handler.Ping)
	http.Handle("/", handler.RootHandler(metricsPath))

	log.Info().Msgf("node_exporter listen %s", webAddr)
	server := &http.Server{
		Addr:         webAddr,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		IdleTimeout:  time.Second * 10,
	}

	return server.ListenAndServe()
}
