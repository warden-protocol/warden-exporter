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
	rulesMetricName   = "warden_rules"
	actionsMetricName = "warden_actions"
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var rules = prometheus.NewDesc(
	rulesMetricName,
	"Returns the number of Rules existing in chain",
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

type ActionCollector struct {
	Cfg config.Config
}

func (ic ActionCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- actions
	ch <- rules
}

func (ic ActionCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(ic.Cfg.Timeout)*time.Second,
	)

	defer cancel()

	status := successStatus

	client, err := grpc.NewClient(ic.Cfg)
	if err != nil {
		log.Error(fmt.Sprintf("error getting warden metrics: %s", err))
	}

	rulesAmount, err := client.Rules(ctx)
	if err != nil {
		status = errorStatus

		log.Error(err.Error())
	}

	ch <- prometheus.MustNewConstMetric(
		rules,
		prometheus.GaugeValue,
		float64(rulesAmount),
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
