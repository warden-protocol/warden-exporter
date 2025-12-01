package collector

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	"github.com/warden-protocol/warden-exporter/pkg/grpc"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	inflationMetricName        = "cosmos_mint_inflation"
	annualProvisionsMetricName = "cosmos_mint_annual_provisions"
	totalSupplyMetricName      = "cosmos_bank_total_supply"
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var inflation = prometheus.NewDesc(
	inflationMetricName,
	"Current inflation rate of the chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var annualProvisions = prometheus.NewDesc(
	annualProvisionsMetricName,
	"Annual provisions (tokens minted per year)",
	[]string{
		"chain_id",
		"denom",
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var totalSupply = prometheus.NewDesc(
	totalSupplyMetricName,
	"Total supply of the chain denomination",
	[]string{
		"chain_id",
		"denom",
		"status",
	},
	nil,
)

type MintCollector struct {
	Cfg config.Config
}

func (mc MintCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- inflation
	ch <- annualProvisions
	ch <- totalSupply
}

func (mc MintCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(mc.Cfg.Timeout)*time.Second,
	)

	defer cancel()

	client, err := grpc.NewClient(mc.Cfg)
	if err != nil {
		log.Error(fmt.Sprintf("error creating gRPC client for mint metrics: %s", err))
		return
	}

	defer func() {
		if tempErr := client.CloseConn(); tempErr != nil {
			log.Error(tempErr.Error())
		}
	}()

	// Collect inflation
	mc.collectInflation(ctx, client, ch)

	// Collect annual provisions
	mc.collectAnnualProvisions(ctx, client, ch)

	// Collect total supply
	mc.collectTotalSupply(ctx, client, ch)
}

func (mc MintCollector) collectInflation(
	ctx context.Context,
	client grpc.Client,
	ch chan<- prometheus.Metric,
) {
	status := successStatus

	inflationRaw, err := client.Inflation(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("error getting inflation: %s", err))
		status = errorStatus
		ch <- prometheus.MustNewConstMetric(
			inflation,
			prometheus.GaugeValue,
			0,
			mc.Cfg.ChainID,
			status,
		)
		return
	}

	// Parse decimal string from bytes
	inflationStr := string(inflationRaw)
	inflationBigInt := new(big.Int)
	_, ok := inflationBigInt.SetString(inflationStr, 10)
	if !ok {
		log.Error(fmt.Sprintf("error parsing inflation value: %s", inflationStr))
		status = errorStatus
		ch <- prometheus.MustNewConstMetric(
			inflation,
			prometheus.GaugeValue,
			0,
			mc.Cfg.ChainID,
			status,
		)
		return
	}

	// Convert to float64 (inflation is already a decimal ratio, not in base units)
	inflationBig := new(big.Float).SetInt(inflationBigInt)
	denomFactor := new(big.Float).SetInt64(1e18) // SDK uses 18 decimal places for LegacyDec
	result := new(big.Float).Quo(inflationBig, denomFactor)
	inflationFloat, _ := result.Float64()

	ch <- prometheus.MustNewConstMetric(
		inflation,
		prometheus.GaugeValue,
		inflationFloat,
		mc.Cfg.ChainID,
		status,
	)
}

func (mc MintCollector) collectAnnualProvisions(
	ctx context.Context,
	client grpc.Client,
	ch chan<- prometheus.Metric,
) {
	status := successStatus

	provisionsRaw, err := client.AnnualProvisions(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("error getting annual provisions: %s", err))
		status = errorStatus
		ch <- prometheus.MustNewConstMetric(
			annualProvisions,
			prometheus.GaugeValue,
			0,
			mc.Cfg.ChainID,
			mc.Cfg.Denom,
			status,
		)
		return
	}

	// Parse decimal string from bytes
	provisionsStr := string(provisionsRaw)
	provisionsBigInt := new(big.Int)
	_, ok := provisionsBigInt.SetString(provisionsStr, 10)
	if !ok {
		log.Error(fmt.Sprintf("error parsing annual provisions value: %s", provisionsStr))
		status = errorStatus
		ch <- prometheus.MustNewConstMetric(
			annualProvisions,
			prometheus.GaugeValue,
			0,
			mc.Cfg.ChainID,
			mc.Cfg.Denom,
			status,
		)
		return
	}

	// Adjust by exponent (provisions are in base units + 18 decimal LegacyDec precision)
	totalDecimals := mc.Cfg.Exponent + 18
	denomString := fmt.Sprintf("1e%d", totalDecimals)
	denomFactor, ok := new(big.Float).SetString(denomString)
	if !ok {
		log.Error("error parsing denominator factor for annual provisions")
		status = errorStatus
	}

	provisionsBig := new(big.Float).SetInt(provisionsBigInt)
	result := new(big.Float).Quo(provisionsBig, denomFactor)
	provisionsFloat, _ := result.Float64()

	ch <- prometheus.MustNewConstMetric(
		annualProvisions,
		prometheus.GaugeValue,
		provisionsFloat,
		mc.Cfg.ChainID,
		mc.Cfg.Denom,
		status,
	)
}

func (mc MintCollector) collectTotalSupply(
	ctx context.Context,
	client grpc.Client,
	ch chan<- prometheus.Metric,
) {
	status := successStatus

	supplyStr, err := client.TotalSupply(ctx, mc.Cfg.Denom)
	if err != nil {
		log.Error(fmt.Sprintf("error getting total supply: %s", err))
		status = errorStatus
		ch <- prometheus.MustNewConstMetric(
			totalSupply,
			prometheus.GaugeValue,
			0,
			mc.Cfg.ChainID,
			mc.Cfg.Denom,
			status,
		)
		return
	}

	// Convert to float64 and adjust by exponent
	supplyBigInt := new(big.Int)
	_, ok := supplyBigInt.SetString(supplyStr, 10)
	if !ok {
		log.Error(fmt.Sprintf("error parsing total supply value: %s", supplyStr))
		status = errorStatus
		ch <- prometheus.MustNewConstMetric(
			totalSupply,
			prometheus.GaugeValue,
			0,
			mc.Cfg.ChainID,
			mc.Cfg.Denom,
			status,
		)
		return
	}

	denomString := fmt.Sprintf("1e%d", mc.Cfg.Exponent)
	denomFactor, ok := new(big.Float).SetString(denomString)
	if !ok {
		log.Error("error parsing denominator factor for total supply")
		status = errorStatus
	}

	supplyBig := new(big.Float).SetInt(supplyBigInt)
	result := new(big.Float).Quo(supplyBig, denomFactor)
	supplyFloat, _ := result.Float64()

	ch <- prometheus.MustNewConstMetric(
		totalSupply,
		prometheus.GaugeValue,
		supplyFloat,
		mc.Cfg.ChainID,
		mc.Cfg.Denom,
		status,
	)
}
