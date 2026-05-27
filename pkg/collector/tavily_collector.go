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
	tavilyPlanUsageMetricName = "tavily_plan_usage"
	tavilyPlanLimitMetricName = "tavily_plan_limit"
	tavilyAPIURL              = "https://api.tavily.com"
)

type TavilyUsageResponse struct {
	Account struct {
		CurrentPlan string  `json:"current_plan"`
		PlanUsage   float64 `json:"plan_usage"`
		PlanLimit   float64 `json:"plan_limit"`
	} `json:"account"`
}

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var (
	tavilyPlanUsage = prometheus.NewDesc(
		tavilyPlanUsageMetricName,
		"Returns Tavily API account plan usage (credits consumed in current billing cycle)",
		[]string{
			"plan",
			"unit",
			"status",
		},
		nil,
	)

	tavilyPlanLimit = prometheus.NewDesc(
		tavilyPlanLimitMetricName,
		"Returns Tavily API account plan limit (credit ceiling for current billing cycle)",
		[]string{
			"plan",
			"unit",
			"status",
		},
		nil,
	)
)

type TavilyCollector struct {
	Cfg config.Config
}

func (c TavilyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- tavilyPlanUsage
	ch <- tavilyPlanLimit
}

func (c TavilyCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(c.Cfg.Timeout)*time.Second,
	)
	defer cancel()

	status := successStatus

	response, err := c.tavilyCollectUsage(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting Tavily usage %s", err))
		status = errorStatus
		response = TavilyUsageResponse{}
	}

	ch <- prometheus.MustNewConstMetric(
		tavilyPlanUsage,
		prometheus.GaugeValue,
		response.Account.PlanUsage,
		[]string{
			response.Account.CurrentPlan,
			"credits",
			status,
		}...,
	)

	ch <- prometheus.MustNewConstMetric(
		tavilyPlanLimit,
		prometheus.GaugeValue,
		response.Account.PlanLimit,
		[]string{
			response.Account.CurrentPlan,
			"credits",
			status,
		}...,
	)
}

func (c TavilyCollector) tavilyCollectUsage(ctx context.Context) (TavilyUsageResponse, error) {
	url := fmt.Sprintf("%s/usage", tavilyAPIURL)

	data, err := http.GetRequest(ctx, url, c.Cfg.TavilyAPIKey, c.Cfg.HTTPTimeout)
	if err != nil {
		return TavilyUsageResponse{}, err
	}

	var tavilyResponse TavilyUsageResponse
	if err = json.Unmarshal(data, &tavilyResponse); err != nil {
		return TavilyUsageResponse{}, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info("Tavily API usage request successful")

	return tavilyResponse, nil
}
