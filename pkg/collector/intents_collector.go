package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	"github.com/warden-protocol/warden-exporter/pkg/grpc"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	intentsMetricName = "warden_intents"
	actionsMetricName = "warden_actions"
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var intents = prometheus.NewDesc(
	intentsMetricName,
	"Returns the number of Intents existing in chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var actions = prometheus.NewDesc(
	actionsMetricName,
	"Returns the number of actions in the chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

type IntentCollector struct {
	Cfg config.Config
}

func (ic IntentCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- intents
	ch <- actions
}

func (ic IntentCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(ic.Cfg.Timeout)*time.Second,
	)

	defer cancel()

	status := successStatus

	client, err := grpc.NewClient(ic.Cfg)
	if err != nil {
		log.Error(fmt.Sprintf("error getting spaces metrics: %s", err))
	}

	intentsAmount, err := client.Intents(ctx)
	if err != nil {
		status = errorStatus

		log.Error(err.Error())
	}

	ch <- prometheus.MustNewConstMetric(
		intents,
		prometheus.GaugeValue,
		float64(intentsAmount),
		[]string{
			ic.Cfg.ChainID,
			status,
		}...,
	)

	actionsAmount, err := client.Actions(ctx)
	if err != nil {
		status = errorStatus

		log.Error(err.Error())
	}

	ch <- prometheus.MustNewConstMetric(
		actions,
		prometheus.GaugeValue,
		float64(actionsAmount),
		[]string{
			ic.Cfg.ChainID,
			status,
		}...,
	)
}
