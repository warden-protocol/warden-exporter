package collector

import (
	"bytes"
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
	xaiUsageMetricName         = "xai_usage"
	xaiSpendingLimitMetricName = "xai_postpaid_spending_limit"
	xaiBalanceMetricName       = "xai_prepaid_balance"
	xaiAPIURL                  = "https://management-api.x.ai/v1"
)

type XAIUsageRequest struct {
	AnalyticsRequest struct {
		TimeRange struct {
			StartTime string `json:"startTime"`
			EndTime   string `json:"endTime"`
			Timezone  string `json:"timezone"`
		} `json:"timeRange"`
		TimeUnit string `json:"timeUnit"`
		Values   []struct {
			Name        string `json:"name"`
			Aggregation string `json:"aggregation"`
		} `json:"values"`
		GroupBy []string      `json:"groupBy"`
		Filters []interface{} `json:"filters"`
	} `json:"analyticsRequest"`
}

type XAIUsageResponse struct {
	TimeSeries []struct {
		Group       []string `json:"group"`
		GroupLabels []string `json:"groupLabels"`
		DataPoints  []struct {
			Timestamp string    `json:"timestamp"`
			Values    []float64 `json:"values"`
		} `json:"dataPoints"`
	} `json:"timeSeries"`
	LimitReached bool `json:"limitReached"`
}

type XAISpendingLimitsResponse struct {
	SpendingLimits struct {
		HardSlAuto struct {
			Val string `json:"val"`
		} `json:"hardSlAuto"`
		EffectiveHardSl struct {
			Val string `json:"val"`
		} `json:"effectiveHardSl"`
		SoftSl struct {
			Val string `json:"val"`
		} `json:"softSl"`
		EffectiveSl struct {
			Val string `json:"val"`
		} `json:"effectiveSl"`
	} `json:"spendingLimits"`
}

type XAIBalanceResponse struct {
	Changes []struct {
		TeamID       string `json:"teamId"`
		ChangeOrigin string `json:"changeOrigin"`
		TopupStatus  string `json:"topupStatus,omitempty"`
		Amount       struct {
			Val string `json:"val"`
		} `json:"amount"`
		InvoiceID        string `json:"invoiceId"`
		InvoiceNumber    string `json:"invoiceNumber"`
		CreateTime       string `json:"createTime"`
		CreateTs         string `json:"createTs"`
		SpendBpKeyYear   int    `json:"spendBpKeyYear,omitempty"`
		SpendBpKeyMonth  int    `json:"spendBpKeyMonth,omitempty"`
		PaymentProcessor struct {
			Kind string `json:"kind"`
		} `json:"paymentProcessor"`
	} `json:"changes"`
	Total struct {
		Val string `json:"val"`
	} `json:"total"`
}

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var (
	xaiUsage = prometheus.NewDesc(
		xaiUsageMetricName,
		"Returns X.AI API usage cost in USD",
		[]string{
			"period",
			"currency",
			"status",
		},
		nil,
	)

	xaiSpendingLimit = prometheus.NewDesc(
		xaiSpendingLimitMetricName,
		"Returns X.AI postpaid spending limits information",
		[]string{
			"limit_type",
			"currency",
			"status",
		},
		nil,
	)

	xaiBalance = prometheus.NewDesc(
		xaiBalanceMetricName,
		"Returns X.AI prepaid balance information",
		[]string{
			"currency",
			"status",
		},
		nil,
	)
)

type XAICollector struct {
	Cfg config.Config
}

func (x XAICollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- xaiUsage
	ch <- xaiSpendingLimit
	ch <- xaiBalance
}

func (x XAICollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(x.Cfg.Timeout)*time.Second,
	)
	defer cancel()

	var errors []string

	errors = x.collectUsageMetrics(ctx, ch, errors)
	errors = x.collectSpendingLimitMetrics(ctx, ch, errors)
	errors = x.collectBalanceMetrics(ctx, ch, errors)

	if len(errors) > 0 {
		log.Info(fmt.Sprintf("X.AI metrics collection completed with errors: %v", errors))
	} else {
		log.Info("X.AI metrics collection completed successfully")
	}
}

func (x XAICollector) collectUsageMetrics(
	ctx context.Context,
	ch chan<- prometheus.Metric,
	errors []string,
) []string {
	monthlyUsageStatus := successStatus
	monthlyUsage, err := x.xaiCollectUsageMonthly(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting X.AI monthly usage: %s", err))
		errors = append(errors, "monthly usage")
		monthlyUsageStatus = errorStatus
		monthlyUsage = 0
	}

	ch <- prometheus.MustNewConstMetric(
		xaiUsage,
		prometheus.GaugeValue,
		monthlyUsage,
		[]string{"monthly", "USD", monthlyUsageStatus}...,
	)

	dailyUsageStatus := successStatus
	dailyUsage, errDaily := x.xaiCollectUsageDaily(ctx)
	if errDaily != nil {
		log.Error(fmt.Sprintf("error collecting X.AI daily usage: %s", errDaily))
		errors = append(errors, "daily usage")
		dailyUsageStatus = errorStatus
		dailyUsage = 0
	}

	ch <- prometheus.MustNewConstMetric(
		xaiUsage,
		prometheus.GaugeValue,
		dailyUsage,
		[]string{"daily", "USD", dailyUsageStatus}...,
	)

	return errors
}

func (x XAICollector) collectSpendingLimitMetrics(
	ctx context.Context,
	ch chan<- prometheus.Metric,
	errors []string,
) []string {
	spendingStatus := successStatus
	spendingLimits, err := x.xaiCollectSpendingLimits(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting X.AI spending limits: %s", err))
		errors = append(errors, "spending limits")
		spendingStatus = errorStatus
		spendingLimits = XAISpendingLimitsResponse{}
	}

	hardSlAuto := x.parseFloatValue(spendingLimits.SpendingLimits.HardSlAuto.Val) / 100.0
	effectiveHardSl := x.parseFloatValue(spendingLimits.SpendingLimits.EffectiveHardSl.Val) / 100.0
	softSl := x.parseFloatValue(spendingLimits.SpendingLimits.SoftSl.Val) / 100.0
	effectiveSl := x.parseFloatValue(spendingLimits.SpendingLimits.EffectiveSl.Val) / 100.0

	ch <- prometheus.MustNewConstMetric(
		xaiSpendingLimit,
		prometheus.GaugeValue,
		hardSlAuto,
		[]string{"hard_sl_auto", "USD", spendingStatus}...,
	)

	ch <- prometheus.MustNewConstMetric(
		xaiSpendingLimit,
		prometheus.GaugeValue,
		effectiveHardSl,
		[]string{"effective_hard_sl", "USD", spendingStatus}...,
	)

	ch <- prometheus.MustNewConstMetric(
		xaiSpendingLimit,
		prometheus.GaugeValue,
		softSl,
		[]string{"soft_sl", "USD", spendingStatus}...,
	)

	ch <- prometheus.MustNewConstMetric(
		xaiSpendingLimit,
		prometheus.GaugeValue,
		effectiveSl,
		[]string{"effective_sl", "USD", spendingStatus}...,
	)

	return errors
}

func (x XAICollector) collectBalanceMetrics(
	ctx context.Context,
	ch chan<- prometheus.Metric,
	errors []string,
) []string {
	balanceStatus := successStatus
	balance, err := x.xaiCollectBalance(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting X.AI balance: %s", err))
		errors = append(errors, "balance")
		balanceStatus = errorStatus
		balance = XAIBalanceResponse{}
	}

	totalBalanceCents := x.parseFloatValue(balance.Total.Val)
	totalBalance := totalBalanceCents / 100.0

	ch <- prometheus.MustNewConstMetric(
		xaiBalance,
		prometheus.GaugeValue,
		totalBalance,
		[]string{"USD", balanceStatus}...,
	)

	return errors
}

func (x XAICollector) xaiCollectUsageMonthly(ctx context.Context) (float64, error) {
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	return x.xaiCollectUsage(ctx, startOfMonth, now, "TIME_UNIT_MONTH")
}

func (x XAICollector) xaiCollectUsageDaily(ctx context.Context) (float64, error) {
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	return x.xaiCollectUsage(ctx, startOfDay, now, "TIME_UNIT_DAY")
}

func (x XAICollector) xaiCollectUsage(
	ctx context.Context,
	startTime, endTime time.Time,
	timeUnit string,
) (float64, error) {
	url := fmt.Sprintf("%s/billing/teams/%s/usage", xaiAPIURL, x.Cfg.XAITeamID)

	var requestBody XAIUsageRequest
	requestBody.AnalyticsRequest.TimeRange.StartTime = startTime.Format("2006-01-02 15:04:05")
	requestBody.AnalyticsRequest.TimeRange.EndTime = endTime.Format("2006-01-02 15:04:05")
	requestBody.AnalyticsRequest.TimeRange.Timezone = "Etc/GMT"
	requestBody.AnalyticsRequest.TimeUnit = timeUnit
	requestBody.AnalyticsRequest.Values = []struct {
		Name        string `json:"name"`
		Aggregation string `json:"aggregation"`
	}{
		{
			Name:        "usd",
			Aggregation: "AGGREGATION_SUM",
		},
	}
	requestBody.AnalyticsRequest.GroupBy = []string{"description"}
	requestBody.AnalyticsRequest.Filters = []interface{}{}

	data, err := x.makeXAIPostRequest(ctx, url, requestBody)
	if err != nil {
		return 0, err
	}

	var xaiResponse XAIUsageResponse
	if err = json.NewDecoder(bytes.NewReader(data)).Decode(&xaiResponse); err != nil {
		return 0, fmt.Errorf("error decoding response: %w", err)
	}

	var totalCost float64
	for _, series := range xaiResponse.TimeSeries {
		for _, dataPoint := range series.DataPoints {
			for _, value := range dataPoint.Values {
				totalCost += value
			}
		}
	}

	return totalCost, nil
}

func (x XAICollector) xaiCollectSpendingLimits(ctx context.Context) (XAISpendingLimitsResponse, error) {
	url := fmt.Sprintf("%s/billing/teams/%s/postpaid/spending-limits", xaiAPIURL, x.Cfg.XAITeamID)

	data, err := x.makeXAIRequest(ctx, url)
	if err != nil {
		return XAISpendingLimitsResponse{}, err
	}

	var xaiResponse XAISpendingLimitsResponse
	if err = json.NewDecoder(bytes.NewReader(data)).Decode(&xaiResponse); err != nil {
		return XAISpendingLimitsResponse{}, fmt.Errorf("error decoding response: %w", err)
	}

	return xaiResponse, nil
}

func (x XAICollector) xaiCollectBalance(ctx context.Context) (XAIBalanceResponse, error) {
	url := fmt.Sprintf("%s/billing/teams/%s/prepaid/balance", xaiAPIURL, x.Cfg.XAITeamID)

	data, err := x.makeXAIRequest(ctx, url)
	if err != nil {
		return XAIBalanceResponse{}, err
	}

	var xaiResponse XAIBalanceResponse
	if errDecode := json.NewDecoder(bytes.NewReader(data)).Decode(&xaiResponse); errDecode != nil {
		return XAIBalanceResponse{}, fmt.Errorf("error decoding response: %w", errDecode)
	}

	return xaiResponse, nil
}

func (x XAICollector) makeXAIRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+x.Cfg.XAIAPIKey)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Duration(x.Cfg.Timeout) * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	}

	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, errRead := buf.ReadFrom(resp.Body); errRead != nil {
		return nil, fmt.Errorf("error reading response body: %w", errRead)
	}

	if resp.StatusCode != http.StatusOK {
		log.Error(fmt.Sprintf("X.AI API error. Status: %d, URL: %s, Response: %s",
			resp.StatusCode, url, buf.String()))
		return nil, fmt.Errorf("received non-OK response: %d", resp.StatusCode)
	}

	return buf.Bytes(), nil
}

func (x XAICollector) parseFloatValue(val string) float64 {
	if val == "" {
		return 0
	}
	result, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0
	}
	return result
}

func (x XAICollector) makeXAIPostRequest(ctx context.Context, url string, body interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+x.Cfg.XAIAPIKey)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Duration(x.Cfg.Timeout) * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	}

	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, errRead := buf.ReadFrom(resp.Body); errRead != nil {
		return nil, fmt.Errorf("error reading response body: %w", errRead)
	}

	if resp.StatusCode != http.StatusOK {
		log.Error(fmt.Sprintf("X.AI API error. Status: %d, URL: %s, Response: %s",
			resp.StatusCode, url, buf.String()))
		return nil, fmt.Errorf("received non-OK response: %d", resp.StatusCode)
	}

	return buf.Bytes(), nil
}
