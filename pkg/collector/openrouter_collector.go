package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	http "github.com/warden-protocol/warden-exporter/pkg/http"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	openRouterUsageMetricName          = "openrouter_usage"
	openRouterLimitMetricName          = "openrouter_limit"
	openRouterLimitRemainingMetricName = "openrouter_limit_remaining"
	openRouterCreditsTotalMetricName   = "openrouter_credits_total"
	openRouterCreditsUsageMetricName   = "openrouter_credits_usage"
	openRouterAPIURL                   = "https://openrouter.ai/api/v1"
)

type OpenRouterKeyResponse struct {
	Data struct {
		Label          string  `json:"label"`
		LimitReset     string  `json:"limit_reset"`
		Limit          float64 `json:"limit"`
		LimitRemaining float64 `json:"limit_remaining"`
		Usage          float64 `json:"usage"`
		UsageDaily     float64 `json:"usage_daily"`
		UsageWeekly    float64 `json:"usage_weekly"`
		UsageMonthly   float64 `json:"usage_monthly"`
	} `json:"data"`
}

type OpenRouterCreditsResponse struct {
	Data struct {
		TotalCredits float64 `json:"total_credits"`
		TotalUsage   float64 `json:"total_usage"`
	} `json:"data"`
}

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var (
	openRouterUsage = prometheus.NewDesc(
		openRouterUsageMetricName,
		"Returns OpenRouter API key usage in USD over the given period (total/daily/weekly/monthly)",
		[]string{
			"key",
			"period",
			"unit",
			"status",
		},
		nil,
	)

	openRouterLimit = prometheus.NewDesc(
		openRouterLimitMetricName,
		"Returns OpenRouter API key spending limit in USD for the configured period",
		[]string{
			"key",
			"period",
			"unit",
			"status",
		},
		nil,
	)

	openRouterLimitRemaining = prometheus.NewDesc(
		openRouterLimitRemainingMetricName,
		"Returns OpenRouter API key remaining spending allowance in USD for the configured period",
		[]string{
			"key",
			"period",
			"unit",
			"status",
		},
		nil,
	)

	openRouterCreditsTotal = prometheus.NewDesc(
		openRouterCreditsTotalMetricName,
		"Returns OpenRouter account total purchased credits in USD",
		[]string{
			"key",
			"unit",
			"status",
		},
		nil,
	)

	openRouterCreditsUsage = prometheus.NewDesc(
		openRouterCreditsUsageMetricName,
		"Returns OpenRouter account total credit usage in USD",
		[]string{
			"key",
			"unit",
			"status",
		},
		nil,
	)
)

type OpenRouterCollector struct {
	Cfg config.Config
}

func (c OpenRouterCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- openRouterUsage
	ch <- openRouterLimit
	ch <- openRouterLimitRemaining
	ch <- openRouterCreditsTotal
	ch <- openRouterCreditsUsage
}

func (c OpenRouterCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(c.Cfg.Timeout)*time.Second,
	)
	defer cancel()

	for _, apiKey := range splitCommaList(c.Cfg.OpenRouterAPIKey) {
		c.collectKey(ctx, ch, apiKey)
	}
}

func (c OpenRouterCollector) collectKey(
	ctx context.Context,
	ch chan<- prometheus.Metric,
	apiKey string,
) {
	keyStatus := successStatus
	keyResp, err := c.openRouterCollectKey(ctx, apiKey)
	keyLabel := keyResp.Data.Label
	if err != nil {
		log.Error(fmt.Sprintf("error collecting OpenRouter key info %s", err))
		keyStatus = errorStatus
		keyResp = OpenRouterKeyResponse{}
		keyLabel = redactKey(apiKey)
	}

	period := keyResp.Data.LimitReset
	if period == "" {
		period = "unknown"
	}

	usageBuckets := []struct {
		period string
		value  float64
	}{
		{"total", keyResp.Data.Usage},
		{"daily", keyResp.Data.UsageDaily},
		{"weekly", keyResp.Data.UsageWeekly},
		{"monthly", keyResp.Data.UsageMonthly},
	}
	for _, b := range usageBuckets {
		ch <- prometheus.MustNewConstMetric(
			openRouterUsage,
			prometheus.GaugeValue,
			b.value,
			[]string{keyLabel, b.period, "USD", keyStatus}...,
		)
	}

	ch <- prometheus.MustNewConstMetric(
		openRouterLimit,
		prometheus.GaugeValue,
		keyResp.Data.Limit,
		[]string{keyLabel, period, "USD", keyStatus}...,
	)

	ch <- prometheus.MustNewConstMetric(
		openRouterLimitRemaining,
		prometheus.GaugeValue,
		keyResp.Data.LimitRemaining,
		[]string{keyLabel, period, "USD", keyStatus}...,
	)

	creditsStatus := successStatus
	creditsResp, err := c.openRouterCollectCredits(ctx, apiKey)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting OpenRouter credits %s", err))
		creditsStatus = errorStatus
		creditsResp = OpenRouterCreditsResponse{}
	}

	ch <- prometheus.MustNewConstMetric(
		openRouterCreditsTotal,
		prometheus.GaugeValue,
		creditsResp.Data.TotalCredits,
		[]string{keyLabel, "USD", creditsStatus}...,
	)

	ch <- prometheus.MustNewConstMetric(
		openRouterCreditsUsage,
		prometheus.GaugeValue,
		creditsResp.Data.TotalUsage,
		[]string{keyLabel, "USD", creditsStatus}...,
	)
}

func (c OpenRouterCollector) openRouterCollectKey(
	ctx context.Context,
	apiKey string,
) (OpenRouterKeyResponse, error) {
	url := fmt.Sprintf("%s/auth/key", openRouterAPIURL)

	data, err := http.GetRequest(ctx, url, apiKey, c.Cfg.HTTPTimeout)
	if err != nil {
		return OpenRouterKeyResponse{}, err
	}

	var resp OpenRouterKeyResponse
	if err = json.Unmarshal(data, &resp); err != nil {
		return OpenRouterKeyResponse{}, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info("OpenRouter API key request successful")

	return resp, nil
}

func (c OpenRouterCollector) openRouterCollectCredits(
	ctx context.Context,
	apiKey string,
) (OpenRouterCreditsResponse, error) {
	url := fmt.Sprintf("%s/credits", openRouterAPIURL)

	data, err := http.GetRequest(ctx, url, apiKey, c.Cfg.HTTPTimeout)
	if err != nil {
		return OpenRouterCreditsResponse{}, err
	}

	var resp OpenRouterCreditsResponse
	if err = json.Unmarshal(data, &resp); err != nil {
		return OpenRouterCreditsResponse{}, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info("OpenRouter API credits request successful")

	return resp, nil
}
