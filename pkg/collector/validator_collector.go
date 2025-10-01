package collector

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	"github.com/warden-protocol/warden-exporter/pkg/grpc"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
	types "github.com/warden-protocol/warden-exporter/pkg/types"
)

const (
	missedBlocksMetricName   = "cosmos_validator_missed_blocks"
	blocksProposedMetricName = "cosmos_validator_blocks_proposed"
	avgBlockTimeMetricName   = "cosmos_chain_avg_block_time_seconds"
	tokensMetricName         = "cosmos_validator_tokens"
	delegatorSharesMetric    = "cosmos_validator_delegator_shares"
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var missedBlocks = prometheus.NewDesc(
	missedBlocksMetricName,
	"Returns missed blocks for a validator.",
	[]string{
		"chain_id",
		"valcons",
		"valoper",
		"moniker",
		"jailed",
		"tombstoned",
		"bond_status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var blocksProposed = prometheus.NewDesc(
	blocksProposedMetricName,
	"Returns number of blocks proposed by validator in recent window.",
	[]string{
		"chain_id",
		"valcons",
		"valoper",
		"moniker",
		"jailed",
		"tombstoned",
		"bond_status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var avgBlockTime = prometheus.NewDesc(
	avgBlockTimeMetricName,
	"Returns average block time in seconds for the chain.",
	[]string{
		"chain_id",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var tokens = prometheus.NewDesc(
	tokensMetricName,
	"Returns total bonded tokens for validator.",
	[]string{
		"chain_id",
		"valcons",
		"valoper",
		"moniker",
		"jailed",
		"tombstoned",
		"bond_status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var delegatorShares = prometheus.NewDesc(
	delegatorSharesMetric,
	"Returns total delegator shares for validator.",
	[]string{
		"chain_id",
		"valcons",
		"valoper",
		"moniker",
		"jailed",
		"tombstoned",
		"bond_status",
	},
	nil,
)

type ValidatorsCollector struct {
	Cfg config.Config
}

func (vc ValidatorsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- missedBlocks
	ch <- blocksProposed
	ch <- avgBlockTime
	ch <- tokens
	ch <- delegatorShares
}

func (vc ValidatorsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(vc.Cfg.Timeout)*time.Second,
	)

	defer cancel()

	vals, err := grpc.SigningValidators(ctx, vc.Cfg)
	if err != nil {
		log.Error(fmt.Sprintf("error getting signing validators: %s", err))
	} else {
		// Get block proposer counts
		proposerCounts, err := grpc.BlockProposers(ctx, vc.Cfg, vc.Cfg.BlockWindow)
		if err != nil {
			log.Error(fmt.Sprintf("error getting block proposers: %s", err))
			proposerCounts = make(map[string]int64)
		}

		// Merge proposer counts into validator data
		for i := range vals {
			if count, ok := proposerCounts[vals[i].ConsAddress]; ok {
				vals[i].BlocksProposed = count
			}
		}

		for _, m := range vc.missedBlocksMetrics(vals) {
			ch <- m
		}

		for _, m := range vc.blocksProposedMetrics(vals) {
			ch <- m
		}

		for _, m := range vc.tokensMetrics(vals) {
			ch <- m
		}

		for _, m := range vc.delegatorSharesMetrics(vals) {
			ch <- m
		}
	}

	// Get average block time
	blockTime, err := grpc.AverageBlockTime(ctx, vc.Cfg, 100)
	if err != nil {
		log.Error(fmt.Sprintf("error getting average block time: %s", err))
	} else {
		ch <- prometheus.MustNewConstMetric(
			avgBlockTime,
			prometheus.GaugeValue,
			blockTime,
			vc.Cfg.ChainID,
		)
	}
}

func (vc ValidatorsCollector) missedBlocksMetrics(vals []types.Validator) []prometheus.Metric {
	metrics := []prometheus.Metric{}

	for _, val := range vals {
		metrics = append(
			metrics,
			prometheus.MustNewConstMetric(
				missedBlocks,
				prometheus.GaugeValue,
				float64(val.MissedBlocks),
				[]string{
					vc.Cfg.ChainID,
					val.ConsAddress,
					val.OperatorAddress,
					val.Moniker,
					strconv.FormatBool(val.Jailed),
					strconv.FormatBool(val.Tombstoned),
					val.BondStatus,
				}...,
			),
		)
	}

	return metrics
}

func (vc ValidatorsCollector) blocksProposedMetrics(vals []types.Validator) []prometheus.Metric {
	metrics := []prometheus.Metric{}

	for _, val := range vals {
		metrics = append(
			metrics,
			prometheus.MustNewConstMetric(
				blocksProposed,
				prometheus.GaugeValue,
				float64(val.BlocksProposed),
				[]string{
					vc.Cfg.ChainID,
					val.ConsAddress,
					val.OperatorAddress,
					val.Moniker,
					strconv.FormatBool(val.Jailed),
					strconv.FormatBool(val.Tombstoned),
					val.BondStatus,
				}...,
			),
		)
	}

	return metrics
}

func (vc ValidatorsCollector) tokensMetrics(vals []types.Validator) []prometheus.Metric {
	metrics := []prometheus.Metric{}

	for _, val := range vals {
		metrics = append(
			metrics,
			prometheus.MustNewConstMetric(
				tokens,
				prometheus.GaugeValue,
				val.Tokens,
				[]string{
					vc.Cfg.ChainID,
					val.ConsAddress,
					val.OperatorAddress,
					val.Moniker,
					strconv.FormatBool(val.Jailed),
					strconv.FormatBool(val.Tombstoned),
					val.BondStatus,
				}...,
			),
		)
	}

	return metrics
}

func (vc ValidatorsCollector) delegatorSharesMetrics(vals []types.Validator) []prometheus.Metric {
	metrics := []prometheus.Metric{}

	for _, val := range vals {
		metrics = append(
			metrics,
			prometheus.MustNewConstMetric(
				delegatorShares,
				prometheus.GaugeValue,
				val.DelegatorShares,
				[]string{
					vc.Cfg.ChainID,
					val.ConsAddress,
					val.OperatorAddress,
					val.Moniker,
					strconv.FormatBool(val.Jailed),
					strconv.FormatBool(val.Tombstoned),
					val.BondStatus,
				}...,
			),
		)
	}

	return metrics
}
