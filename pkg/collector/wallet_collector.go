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
		balanceBigInt := new(big.Int)
		balanceBigInt.SetString(balanceRaw.String(), 10)

		// Adjust based on your denomination
		denomString := fmt.Sprintf("1e%d", w.Cfg.Exponent) // w.Cfg.Exponent assumed to be 18
		denomFactor, ok := new(big.Float).SetString(denomString)
		if !ok {
			log.Error("Error parsing denominator factor")
		}

		// Create a *big.Float from the balance raw value
		balanceBig := new(big.Float).SetInt(balanceBigInt)

		// Perform the division
		result := new(big.Float).Quo(balanceBig, denomFactor)

		// Convert the result to float64 if necessary (note: this might still lead to a float64 approximation)
		balance, _ := result.Float64()

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
