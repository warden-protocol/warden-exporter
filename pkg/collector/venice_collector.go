package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	"github.com/warden-protocol/warden-exporter/pkg/http"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	veniceBillingMetricName = "venice_funds"
	veniceUsageMetricName   = "venice_api_key_usage"
	veniceAPIURL            = "https://api.venice.ai/api/v1"
)

type VeniceUsageResponse struct {
	Data []struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Usage       struct {
			TrailingSevenDays struct {
				USD  string `json:"usd"`
				VCU  string `json:"vcu"`
				DIEM string `json:"diem"`
			} `json:"trailingSevenDays"`
		} `json:"usage"`
	} `json:"data"`
}

type VeniceBalanceResponse struct {
	Data struct {
		AccessPermitted bool `json:"accessPermitted"`
		Balances        struct {
			USD  float64 `json:"USD"`
			VCU  float64 `json:"VCU"`
			DIEM float64 `json:"DIEM"`
		} `json:"balances"`
	} `json:"data"`
}

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var veniceBilling = prometheus.NewDesc(
	veniceBillingMetricName,
	"Returns Venice Billing information",
	[]string{
		"account",
		"symbol",
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var veniceUsage = prometheus.NewDesc(
	veniceUsageMetricName,
	"Returns Venice API Key usage information",
	[]string{
		"account",
		"id",
		"description",
		"symbol",
		"status",
	},
	nil,
)

type VeniceCollector struct {
	Cfg config.Config
}

func (v VeniceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- veniceBilling
	ch <- veniceUsage
}

func (v VeniceCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(v.Cfg.Timeout)*time.Second,
	)
	defer cancel()

	for _, apiKey := range splitCommaList(v.Cfg.VeniceAPIKey) {
		v.collectAccount(ctx, ch, apiKey)
	}
}

func (v VeniceCollector) collectAccount(
	ctx context.Context,
	ch chan<- prometheus.Metric,
	apiKey string,
) {
	account := redactKey(apiKey)
	status := successStatus

	diemBalance, usdBalance, err := v.veniceCollectBalance(ctx, apiKey)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting Venice balance: %s", err))
		status = errorStatus
	}

	ch <- prometheus.MustNewConstMetric(
		veniceBilling,
		prometheus.GaugeValue,
		diemBalance,
		[]string{
			account,
			"DIEM",
			status,
		}...,
	)

	ch <- prometheus.MustNewConstMetric(
		veniceBilling,
		prometheus.GaugeValue,
		usdBalance,
		[]string{
			account,
			"USD",
			status,
		}...,
	)

	if !v.Cfg.VeniceUsageMetrics {
		return
	}

	usage, err := v.veniceCollectUsage(ctx, apiKey)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting Venice usage: %s", err))
		status = errorStatus
	}

	for _, data := range usage.Data {
		diemCount, _ := strconv.ParseFloat(data.Usage.TrailingSevenDays.DIEM, 64)
		usdCount, _ := strconv.ParseFloat(data.Usage.TrailingSevenDays.USD, 64)
		ch <- prometheus.MustNewConstMetric(
			veniceUsage,
			prometheus.GaugeValue,
			diemCount,
			account,
			data.ID,
			data.Description,
			"DIEM",
			status,
		)
		ch <- prometheus.MustNewConstMetric(
			veniceUsage,
			prometheus.GaugeValue,
			usdCount,
			account,
			data.ID,
			data.Description,
			"USD",
			status,
		)
	}
}

func (v VeniceCollector) veniceCollectUsage(
	ctx context.Context,
	apiKey string,
) (VeniceUsageResponse, error) {
	url := fmt.Sprintf("%s/api_keys", veniceAPIURL)

	data, err := http.GetRequest(ctx, url, apiKey, v.Cfg.HTTPTimeout)
	if err != nil {
		return VeniceUsageResponse{}, err
	}

	var veniceResponse VeniceUsageResponse
	if err = json.NewDecoder(bytes.NewReader(data)).Decode(&veniceResponse); err != nil {
		return VeniceUsageResponse{}, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info("Venice API usage request successful")

	return veniceResponse, nil
}

func (v VeniceCollector) veniceCollectBalance(
	ctx context.Context,
	apiKey string,
) (float64, float64, error) {
	url := fmt.Sprintf("%s/api_keys/rate_Limits", veniceAPIURL)

	data, err := http.GetRequest(ctx, url, apiKey, v.Cfg.HTTPTimeout)
	if err != nil {
		return 0, 0, err
	}
	var veniceResponse VeniceBalanceResponse
	if err = json.NewDecoder(bytes.NewReader(data)).Decode(&veniceResponse); err != nil {
		return 0, 0, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info("Venice Billing API request successful")

	return veniceResponse.Data.Balances.DIEM,
		veniceResponse.Data.Balances.USD, nil
}
