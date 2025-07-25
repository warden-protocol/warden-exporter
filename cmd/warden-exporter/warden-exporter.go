package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/warden-protocol/warden-exporter/pkg/collector"
	"github.com/warden-protocol/warden-exporter/pkg/config"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err.Error())
	}
	logLevel := log.LevelFlag()

	flag.Parse()

	log.SetLevel(*logLevel)

	if cfg.WardenMetrics {
		wardenCollector := collector.WardenCollector{
			Cfg: cfg,
		}
		actionCollector := collector.ActionCollector{
			Cfg: cfg,
		}
		authCollector := collector.AuthCollector{
			Cfg: cfg,
		}

		go prometheus.MustRegister(wardenCollector)
		go prometheus.MustRegister(actionCollector)
		go prometheus.MustRegister(authCollector)
	}

	if cfg.WalletAddresses != "" {
		walletCollector := collector.WalletBalanceCollector{
			Cfg: cfg,
		}

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

	if cfg.VeniceMetrics {
		veniceCollector := collector.VeniceCollector{
			Cfg: cfg,
		}
		go prometheus.MustRegister(veniceCollector)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", healthCheckHandler)

	addr := fmt.Sprintf(":%s", cfg.Port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.TTL) * time.Second,
		WriteTimeout: time.Duration(cfg.TTL) * time.Second,
	}

	log.Info(fmt.Sprintf("Starting server on addr: %s", addr))

	if err = srv.ListenAndServe(); err != nil {
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
