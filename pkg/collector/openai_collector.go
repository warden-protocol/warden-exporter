package collector

import (
	"bytes"
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
	openAICostMetricName = "openai_cost"
	openAIAPIURL         = "https://api.openai.com/v1"
)

type OpenAICostsResponse struct {
	Data []struct {
		Results []struct {
			Amount struct {
				Value float64 `json:"value"`
			} `json:"amount"`
		} `json:"results"`
	} `json:"data"`
}

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var openAICost = prometheus.NewDesc(
	openAICostMetricName,
	"Returns OpenAI API costs",
	[]string{
		"currency",
		"status",
	},
	nil,
)

type OpenAICollector struct {
	Cfg config.Config
}

func (o OpenAICollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- openAICost
}

func (o OpenAICollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(o.Cfg.Timeout)*time.Second,
	)
	defer cancel()

	var errors []string

	errors = o.collectCostMetrics(ctx, ch, errors)

	if len(errors) > 0 {
		log.Info(fmt.Sprintf("OpenAI metrics collection completed with errors: %v", errors))
	} else {
		log.Info("OpenAI metrics collection completed successfully")
	}
}

func (o OpenAICollector) collectCostMetrics(
	ctx context.Context,
	ch chan<- prometheus.Metric,
	errors []string,
) []string {
	monthlyCostStatus := successStatus
	monthlyCost, err := o.openAICollectCostsMonthly(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting OpenAI monthly costs: %s", err))
		errors = append(errors, "monthly costs")
		monthlyCostStatus = errorStatus
		monthlyCost = 0
	}

	ch <- prometheus.MustNewConstMetric(
		openAICost,
		prometheus.GaugeValue,
		monthlyCost,
		[]string{"USD", monthlyCostStatus}...,
	)

	return errors
}

func (o OpenAICollector) openAICollectCostsMonthly(ctx context.Context) (float64, error) {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	return o.openAICollectCosts(ctx, monthStart, now)
}

func (o OpenAICollector) openAICollectCosts(
	ctx context.Context,
	startTime, endTime time.Time,
) (float64, error) {
	days := int(endTime.Sub(startTime).Hours()/24) + 1

	url := fmt.Sprintf("%s/organization/costs?start_time=%d&limit=%d",
		openAIAPIURL, startTime.Unix(), days)

	data, err := http.GetRequest(ctx, url, o.Cfg.OpenAIAPIKey)
	if err != nil {
		return 0, err
	}

	var costsResponse OpenAICostsResponse
	if err = json.NewDecoder(bytes.NewReader(data)).Decode(&costsResponse); err != nil {
		return 0, fmt.Errorf("error decoding response: %w", err)
	}

	var total float64
	for _, bucket := range costsResponse.Data {
		for _, res := range bucket.Results {
			total += res.Amount.Value
		}
	}

	return total, nil
}
