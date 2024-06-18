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
		walletCollector := collector.WalletBalanceCollector{
			Cfg: cfg,
		}

		go prometheus.MustRegister(wardenCollector)
		go prometheus.MustRegister(intentCollector)
		go prometheus.MustRegister(authCollector)
		go prometheus.MustRegister(walletCollector)
	}

	if cfg.ValidatorMetrics {
		validatorCollector := collector.ValidatorsCollector{
			Cfg: cfg,
		}
		go prometheus.MustRegister(validatorCollector)
	}

	if cfg.WarpMetrics {
		warpCollector := collector.WarpCollector{
			Cfg: cfg,
		}
		go prometheus.MustRegister(warpCollector)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", healthCheckHandler)

	addr := fmt.Sprintf(":%d", *port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.TTL) * time.Second,
		WriteTimeout: time.Duration(cfg.TTL) * time.Second,
	}

	log.Info(fmt.Sprintf("Starting server on addr: %s", addr))

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}
}

// healthCheckHandler handles the /healthz endpoint.
func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Error(err.Error())
	}
}
