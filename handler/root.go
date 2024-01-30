package handler

import (
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/rs/zerolog/log"
	"github.com/zhaoqiang0201/node_exporter/version"
	"net/http"
	"os"
)

func RootHandler(path string) http.Handler {
	landingConfig := web.LandingConfig{
		Name:        "Node Exporter",
		Description: "Prometheus Node Exporter",
		Version:     version.Info(),
		Links: []web.LandingLinks{
			{
				Address: path,
				Text:    "Metrics",
			},
		},
	}
	landingPage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		log.Error().Err(err).Send()
		os.Exit(1)
	}
	return landingPage
}
