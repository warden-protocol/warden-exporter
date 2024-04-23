package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	"github.com/warden-protocol/warden-exporter/pkg/grpc"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	spacesMetricName      = "warden_spaces"
	keysEcdsaMetricName   = "warden_keys_ecdsa"
	keysEddsaMetricName   = "warden_keys_eddsa"
	keysPendingMetricName = "warden_keys_pending"
	keysChainsMetricName  = "warden_keys_keychains"
	successStatus         = "success"
	errorStatus           = "error"
)

var spaces = prometheus.NewDesc(
	spacesMetricName,
	"Returns the number of Spaces existing in chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

var ecdsaKeys = prometheus.NewDesc(
	keysEcdsaMetricName,
	"Returns the number of ECDSA keys existing in chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

var eddsaKeys = prometheus.NewDesc(
	keysEddsaMetricName,
	"Returns the number of EDDSA keys existing in chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

var pendingKeys = prometheus.NewDesc(
	keysPendingMetricName,
	"Returns the number of Pending keys existing in chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

var keyChains = prometheus.NewDesc(
	keysChainsMetricName,
	"Returns the number of Keychains existing in chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

type WardenCollector struct {
	Cfg config.Config
}

func (w WardenCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- spaces
	ch <- ecdsaKeys
	ch <- eddsaKeys
	ch <- pendingKeys
	ch <- keyChains
}

func (w WardenCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(w.Cfg.Timeout)*time.Second,
	)

	defer cancel()

	status := successStatus

	client, err := grpc.NewClient(w.Cfg)
	if err != nil {
		log.Error(fmt.Sprintf("error getting spaces metrics: %s", err))
	}

	spacesAmount, err := client.Spaces(ctx)
	if err != nil {
		status = errorStatus

		log.Error(err.Error())
	}

	ch <- prometheus.MustNewConstMetric(
		spaces,
		prometheus.GaugeValue,
		float64(spacesAmount),
		[]string{
			w.Cfg.ChainID,
			status,
		}...,
	)

	ecdsa, eddsa, pending, err := client.Keys(ctx)
	if err != nil {
		status = errorStatus

		log.Error(err.Error())
	}
	ch <- prometheus.MustNewConstMetric(
		ecdsaKeys,
		prometheus.GaugeValue,
		float64(ecdsa),
		[]string{
			w.Cfg.ChainID,
			status,
		}...,
	)
	ch <- prometheus.MustNewConstMetric(
		eddsaKeys,
		prometheus.GaugeValue,
		float64(eddsa),
		[]string{
			w.Cfg.ChainID,
			status,
		}...,
	)
	ch <- prometheus.MustNewConstMetric(
		pendingKeys,
		prometheus.GaugeValue,
		float64(pending),
		[]string{
			w.Cfg.ChainID,
			status,
		}...,
	)

	keyChainsAmount, err := client.KeyChains(ctx)
	if err != nil {
		status = errorStatus

		log.Error(err.Error())
	}

	ch <- prometheus.MustNewConstMetric(
		keyChains,
		prometheus.GaugeValue,
		float64(keyChainsAmount),
		[]string{
			w.Cfg.ChainID,
			status,
		}...,
	)

	log.Debug("Stop collecting", zap.String("metric", spacesMetricName))
}
