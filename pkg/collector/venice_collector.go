package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	veniceBillingMetricName = "venice_funds"
	veniceUsageMetricName   = "venice_api_key_usage"
	veniceAPIURL            = "https://api.venice.ai/api/v1"
)

type VeniceUsageResponse struct {
	Data []struct {
		ID    string `json:"id"`
		Usage struct {
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
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var veniceUsage = prometheus.NewDesc(
	veniceUsageMetricName,
	"Returns Venice API Key usage information",
	[]string{
		"id",
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
	var err error
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(v.Cfg.Timeout)*time.Second,
	)
	defer cancel()

	status := successStatus

	balance, err := v.veniceCollectBalance(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting Venice balance: %s", err))
		status = errorStatus
	}

	ch <- prometheus.MustNewConstMetric(
		veniceBilling,
		prometheus.GaugeValue,
		balance,
		[]string{
			status,
		}...,
	)

	usage, err := v.veniceCollectUsage(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting Venice usage: %s", err))
		status = errorStatus
	}

	for _, data := range usage.Data {
		diemCount, _ := strconv.ParseFloat(data.Usage.TrailingSevenDays.DIEM, 64)
		ch <- prometheus.MustNewConstMetric(
			veniceUsage,
			prometheus.GaugeValue,
			diemCount,
			data.ID,
			status,
		)
	}
}

func (v VeniceCollector) veniceCollectUsage(ctx context.Context) (VeniceUsageResponse, error) {
	// Perform HTTP GET request to Venice API
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api_keys", veniceAPIURL),
		nil,
	)
	if err != nil {
		return VeniceUsageResponse{}, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+v.Cfg.VeniceAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return VeniceUsageResponse{}, fmt.Errorf("error performing request: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return VeniceUsageResponse{}, fmt.Errorf("received non-OK response: %d", resp.StatusCode)
	}

	var veniceResponse VeniceUsageResponse
	if err = json.NewDecoder(resp.Body).Decode(&veniceResponse); err != nil {
		return VeniceUsageResponse{}, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info("Venice API usage request successful")

	return veniceResponse, nil
}

func (v VeniceCollector) veniceCollectBalance(ctx context.Context) (float64, error) {
	// Perform HTTP GET request to Venice API
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api_keys/rate_Limits", veniceAPIURL),
		nil,
	)
	if err != nil {
		return 0, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+v.Cfg.VeniceAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error performing request: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("received non-OK response: %d", resp.StatusCode)
	}

	var veniceResponse VeniceBalanceResponse
	if err = json.NewDecoder(resp.Body).Decode(&veniceResponse); err != nil {
		return 0, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info("Venice Billing API request successful")

	return veniceResponse.Data.Balances.DIEM, nil
}
