package collector

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	"github.com/warden-protocol/warden-exporter/pkg/grpc"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	walletBalanceMetricName = "cosmos_wallet_balance"
	award                   = 1e18
	uward                   = 1e6
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
	var balanceRaw math.Int
	denomsMap := map[string]float64{
		"award": award,
		"uward": uward,
	}

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
		balanceRaw, err = client.Balance(ctx, addr, w.Cfg.Denom)
		if err != nil {
			log.Error(err.Error())
			status = errorStatus
		}

		// Convert balance from atto to none
		if _, ok := denomsMap[w.Cfg.Denom]; !ok {
			log.Error("Denom not found")
			status = errorStatus
		}

		// Adjust based on your denomination
		denomFactor := new(
			big.Float,
		).SetFloat64(denomsMap[w.Cfg.Denom])
		balanceBig := new(big.Float).SetInt(balanceRaw.BigInt())
		balance, _ := new(big.Float).Quo(balanceBig, denomFactor).Float64()

		ch <- prometheus.MustNewConstMetric(
			walletBalance,
			prometheus.GaugeValue,
			balance,
			[]string{
				w.Cfg.ChainID,
				addr,
				w.Cfg.Denom,
				status,
			}...,
		)
	}
}
