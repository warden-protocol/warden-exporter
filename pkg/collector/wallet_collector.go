package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	"github.com/warden-protocol/warden-exporter/pkg/grpc"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	walletBalanceMetricName = "cosmos_wallet_balance"
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var walletBalance = prometheus.NewDesc(
	walletBalanceMetricName,
	"Returns the wallet balance of account",
	[]string{
		"chain_id",
		"account",
		"denom",
		"status",
	},
	nil,
)

type WalletBalanceCollector struct {
	Cfg config.Config
}

func (w WalletBalanceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- walletBalance
}

func (w WalletBalanceCollector) Collect(ch chan<- prometheus.Metric) {
	var balance uint64
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(w.Cfg.Timeout)*time.Second,
	)

	defer cancel()

	status := successStatus

	client, err := grpc.NewClient(w.Cfg)
	if err != nil {
		log.Error(fmt.Sprintf("error getting wallet balance metrics: %s", err))
	}

	addresses := strings.Split(w.Cfg.WalletAddresses, ",")
	for _, addr := range addresses {
		balance, err = client.Balance(ctx, addr, w.Cfg.Denom)
		if err != nil {
			log.Error(err.Error())
			status = errorStatus
		}
		ch <- prometheus.MustNewConstMetric(
			walletBalance,
			prometheus.GaugeValue,
			float64(balance),
			[]string{
				w.Cfg.ChainID,
				addr,
				w.Cfg.Denom,
				status,
			}...,
		)
	}
}
