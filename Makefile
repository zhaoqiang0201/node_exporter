GO           ?= go
GOHOSTOS     ?= $(shell $(GO) env GOHOSTOS)
GOHOSTARCH   ?= $(shell $(GO) env GOHOSTARCH)
VERSION      ?= $(shell git describe --tags --always)
GOMODULE     ?= $(shell $(GO) list)

default: build

.phony: build
build:
	CGO_ENABLED=0 go build -ldflags="-X $(GOMODULE)/version.Version=$(VERSION)" node_exporter.go