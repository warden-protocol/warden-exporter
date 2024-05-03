package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/warden-protocol/warden-exporter/pkg/collector"
	"github.com/warden-protocol/warden-exporter/pkg/config"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	defaultPort = 8081
	timeout     = 10
)

func main() {
	port := flag.Int("p", defaultPort, "Server port")
	logLevel := log.LevelFlag()

	flag.Parse()

	log.SetLevel(*logLevel)

	cfg := config.Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err.Error())
	}

	if cfg.WardenMetrics {
		wardenCollector := collector.WardenCollector{
			Cfg: cfg,
		}
		intentCollector := collector.IntentCollector{
			Cfg: cfg,
		}
		authCollector := collector.AuthCollector{
			Cfg: cfg,
		}

		prometheus.MustRegister(wardenCollector)
		prometheus.MustRegister(intentCollector)
		prometheus.MustRegister(authCollector)
	}

	if cfg.ValidatorMetrics {
		validatorCollector := collector.ValidatorsCollector{
			Cfg: cfg,
		}
		prometheus.MustRegister(validatorCollector)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	addr := fmt.Sprintf(":%d", *port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  timeout * time.Second,
		WriteTimeout: timeout * time.Second,
	}

	log.Info(fmt.Sprintf("Starting server on addr: %s", addr))

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}
}
