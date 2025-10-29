//nolint:dupl // similar code needed for multiple collectors
package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	bnbBalanceMetricName = "bnb_wallet_balance"
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var bnbBalance = prometheus.NewDesc(
	bnbBalanceMetricName,
	"Returns the wallet balance on BNB blockchain",
	[]string{
		"account",
		"symbol",
		"status",
	},
	nil,
)

type BnbCollector struct {
	Cfg config.Config
}

func (bn BnbCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- bnbBalance
}

func (bn BnbCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(bn.Cfg.Timeout)*time.Second,
	)
	defer cancel()

	addresses := strings.Split(bn.Cfg.BnbAddresses, ",")
	for _, addr := range addresses {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}

		status := successStatus
		balance, err := getBalance(ctx, bn.Cfg.BnbRPCURL, addr, bn.Cfg.Timeout)
		if err != nil {
			log.Error(fmt.Sprintf("error getting balance for address %s: %s", addr, err))
			status = errorStatus
			balance = 0
		}

		ch <- prometheus.MustNewConstMetric(
			bnbBalance,
			prometheus.GaugeValue,
			balance,
			[]string{
				addr,
				"BNB",
				status,
			}...,
		)
	}
}
