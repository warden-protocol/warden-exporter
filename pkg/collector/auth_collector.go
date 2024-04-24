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
	accountMetricName = "warden_accounts"
)

var accounts = prometheus.NewDesc(
	accountMetrciName,
	"Returns the number of accounts existing in chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

type AuthCollector struct {
	Cfg config.Config
}

func (ac AuthCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- accounts
}

func (ac AuthCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(ac.Cfg.Timeout)*time.Second,
	)

	defer cancel()

	status := successStatus

	client, err := grpc.NewClient(ac.Cfg)
	if err != nil {
		log.Error(fmt.Sprintf("error getting spaces metrics: %s", err))
	}

	accountsAmount, err := client.Accounts(ctx)
	if err != nil {
		status = errorStatus

		log.Error(err.Error())
	}

	ch <- prometheus.MustNewConstMetric(
		accounts,
		prometheus.GaugeValue,
		float64(accountsAmount),
		[]string{
			ac.Cfg.ChainID,
			status,
		}...,
	)
}
