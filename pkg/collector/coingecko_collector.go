package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	coinGeckoRateLimitMetricName         = "coingecko_rate_limit_per_minute"
	coinGeckoMonthlyCallCreditMetricName = "coingecko_monthly_call_credit"
	coinGeckoRemainingCallsMetricName    = "coingecko_current_remaining_monthly_calls"
	coinGeckoTotalMonthlyCallsMetricName = "coingecko_current_total_monthly_calls"
	coinGeckoAPIURL                      = "https://pro-api.coingecko.com/api/v3"
)

type CoinGeckoUsageResponse struct {
	Plan                         string  `json:"plan"`
	RateLimitRequestPerMinute    float64 `json:"rate_limit_request_per_minute"`
	MonthlyCallCredit            float64 `json:"monthly_call_credit"`
	CurrentTotalMonthlyCalls     float64 `json:"current_total_monthly_calls"`
	CurrentRemainingMonthlyCalls float64 `json:"current_remaining_monthly_calls"`
}

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var (
	coinGeckoRateLimit = prometheus.NewDesc(
		coinGeckoRateLimitMetricName,
		"Returns CoinGecko API Key rate limit per minute",
		[]string{
			"plan",
			"status",
		},
		nil,
	)

	coinGeckoMonthlyCallCredit = prometheus.NewDesc(
		coinGeckoMonthlyCallCreditMetricName,
		"Returns CoinGecko API Key monthly call credit",
		[]string{
			"plan",
			"status",
		},
		nil,
	)

	coinGeckoRemainingCalls = prometheus.NewDesc(
		coinGeckoRemainingCallsMetricName,
		"Returns CoinGecko API Key remaining monthly calls",
		[]string{
			"plan",
			"status",
		},
		nil,
	)

	coinGeckoTotalMonthlyCalls = prometheus.NewDesc(
		coinGeckoTotalMonthlyCallsMetricName,
		"Returns CoinGecko API Key current total monthly calls",
		[]string{
			"plan",
			"status",
		},
		nil,
	)
)

type CoinGeckoCollector struct {
	Cfg config.Config
}

func (c CoinGeckoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- coinGeckoRateLimit
	ch <- coinGeckoMonthlyCallCredit
	ch <- coinGeckoRemainingCalls
	ch <- coinGeckoTotalMonthlyCalls
}

func (c CoinGeckoCollector) Collect(ch chan<- prometheus.Metric) {
	var err error
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(c.Cfg.Timeout)*time.Second,
	)
	defer cancel()

	status := successStatus

	response, err := c.coinGeckoCollectUsage(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting CoinGecko usage %s", err))
		status = errorStatus
		response = CoinGeckoUsageResponse{}
	}

	ch <- prometheus.MustNewConstMetric(
		coinGeckoRateLimit,
		prometheus.GaugeValue,
		response.RateLimitRequestPerMinute,
		[]string{
			response.Plan,
			status,
		}...,
	)

	ch <- prometheus.MustNewConstMetric(
		coinGeckoMonthlyCallCredit,
		prometheus.GaugeValue,
		response.MonthlyCallCredit,
		[]string{
			response.Plan,
			status,
		}...,
	)

	ch <- prometheus.MustNewConstMetric(
		coinGeckoRemainingCalls,
		prometheus.GaugeValue,
		response.CurrentRemainingMonthlyCalls,
		[]string{
			response.Plan,
			status,
		}...,
	)

	ch <- prometheus.MustNewConstMetric(
		coinGeckoTotalMonthlyCalls,
		prometheus.GaugeValue,
		response.CurrentTotalMonthlyCalls,
		[]string{
			response.Plan,
			status,
		}...,
	)
}

func (c CoinGeckoCollector) coinGeckoCollectUsage(
	ctx context.Context,
) (CoinGeckoUsageResponse, error) {
	url := fmt.Sprintf("%s/key", coinGeckoAPIURL)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return CoinGeckoUsageResponse{}, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("x-cg-pro-api-key", c.Cfg.CoinGeckoAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return CoinGeckoUsageResponse{}, fmt.Errorf("error performing request: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return CoinGeckoUsageResponse{}, fmt.Errorf("received non-OK response: %d", resp.StatusCode)
	}

	var coinGeckoResponse CoinGeckoUsageResponse
	if err = json.NewDecoder(resp.Body).Decode(&coinGeckoResponse); err != nil {
		return CoinGeckoUsageResponse{}, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info("CoinGecko API usage request successful")

	return coinGeckoResponse, nil
}
