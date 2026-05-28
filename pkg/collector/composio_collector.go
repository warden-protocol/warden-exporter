package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	composioUsageQuantityMetricName       = "composio_usage_quantity"
	composioUsageEventsMetricName         = "composio_usage_events"
	composioUsageQuantityByToolMetricName = "composio_usage_quantity_by_tool"
	composioProjectsTotalMetricName       = "composio_projects_total"
	composioAPIURL                        = "https://backend.composio.dev/api/v3.1"
	composioBreakdownLimit                = 100
	composioBreakdownEntityType           = "tool_calls"
	composioBreakdownGroupBy              = "tool_slug"
)

type composioUsageEntity struct {
	Unit          string `json:"unit"`
	TotalQuantity string `json:"total_quantity"`
	EventCount    int64  `json:"event_count"`
}

type composioUsageSummaryResponse struct {
	Entities map[string]composioUsageEntity `json:"entities"`
}

type composioUsageGroup struct {
	Key           string `json:"key"`
	TotalQuantity string `json:"total_quantity"`
	EventCount    int64  `json:"event_count"`
}

type composioUsageBreakdownResponse struct {
	EntityType string               `json:"entity_type"`
	Unit       string               `json:"unit"`
	Groups     []composioUsageGroup `json:"groups"`
}

type composioProjectListResponse struct {
	TotalItems int64 `json:"total_items"`
}

//nolint:gochecknoglobals // descriptors are referenced from both Describe and Collect
var (
	composioUsageQuantity = prometheus.NewDesc(
		composioUsageQuantityMetricName,
		"Composio org metering quantity month-to-date, by entity_type",
		[]string{"entity_type", "unit", "status"},
		nil,
	)

	composioUsageEvents = prometheus.NewDesc(
		composioUsageEventsMetricName,
		"Composio org metering event count month-to-date, by entity_type",
		[]string{"entity_type", "status"},
		nil,
	)

	composioUsageQuantityByTool = prometheus.NewDesc(
		composioUsageQuantityByToolMetricName,
		"Composio org tool_calls metering quantity month-to-date, grouped by tool_slug (top 100)",
		[]string{"tool_slug", "unit", "status"},
		nil,
	)

	composioProjectsTotal = prometheus.NewDesc(
		composioProjectsTotalMetricName,
		"Total number of projects in the Composio organization",
		[]string{"status"},
		nil,
	)
)

type ComposioCollector struct {
	Cfg config.Config
}

func (c ComposioCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- composioUsageQuantity
	ch <- composioUsageEvents
	ch <- composioUsageQuantityByTool
	ch <- composioProjectsTotal
}

func (c ComposioCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(c.Cfg.Timeout)*time.Second,
	)
	defer cancel()

	fromMs, toMs := composioMTDWindow(time.Now().UTC())

	c.collectUsageSummary(ctx, ch, fromMs, toMs)
	c.collectUsageBreakdown(ctx, ch, fromMs, toMs)
	c.collectProjectsTotal(ctx, ch)
}

func (c ComposioCollector) collectUsageSummary(
	ctx context.Context,
	ch chan<- prometheus.Metric,
	fromMs, toMs int64,
) {
	status := successStatus
	resp, err := c.fetchUsageSummary(ctx, fromMs, toMs)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting Composio usage summary %s", err))
		status = errorStatus
		resp = composioUsageSummaryResponse{}
	}

	for entityType, entity := range resp.Entities {
		ch <- prometheus.MustNewConstMetric(
			composioUsageQuantity,
			prometheus.GaugeValue,
			parseFloatOrZero(entity.TotalQuantity),
			[]string{entityType, entity.Unit, status}...,
		)
		ch <- prometheus.MustNewConstMetric(
			composioUsageEvents,
			prometheus.GaugeValue,
			float64(entity.EventCount),
			[]string{entityType, status}...,
		)
	}
}

func (c ComposioCollector) collectUsageBreakdown(
	ctx context.Context,
	ch chan<- prometheus.Metric,
	fromMs, toMs int64,
) {
	status := successStatus
	resp, err := c.fetchUsageBreakdown(ctx, fromMs, toMs)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting Composio usage breakdown %s", err))
		status = errorStatus
		resp = composioUsageBreakdownResponse{}
	}

	for _, group := range resp.Groups {
		ch <- prometheus.MustNewConstMetric(
			composioUsageQuantityByTool,
			prometheus.GaugeValue,
			parseFloatOrZero(group.TotalQuantity),
			[]string{group.Key, resp.Unit, status}...,
		)
	}
}

func (c ComposioCollector) collectProjectsTotal(
	ctx context.Context,
	ch chan<- prometheus.Metric,
) {
	status := successStatus
	resp, err := c.fetchProjectsTotal(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting Composio projects total %s", err))
		status = errorStatus
		resp = composioProjectListResponse{}
	}

	ch <- prometheus.MustNewConstMetric(
		composioProjectsTotal,
		prometheus.GaugeValue,
		float64(resp.TotalItems),
		[]string{status}...,
	)
}

func (c ComposioCollector) fetchUsageSummary(
	ctx context.Context,
	fromMs, toMs int64,
) (composioUsageSummaryResponse, error) {
	body := map[string]int64{"from": fromMs, "to": toMs}
	data, err := composioRequest(ctx, http.MethodPost, composioAPIURL+"/org/usage/summary", c.Cfg.ComposioAPIKey, body, c.Cfg.HTTPTimeout)
	if err != nil {
		return composioUsageSummaryResponse{}, err
	}

	var resp composioUsageSummaryResponse
	if err = json.Unmarshal(data, &resp); err != nil {
		return composioUsageSummaryResponse{}, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info("Composio usage summary request successful")
	return resp, nil
}

func (c ComposioCollector) fetchUsageBreakdown(
	ctx context.Context,
	fromMs, toMs int64,
) (composioUsageBreakdownResponse, error) {
	body := map[string]any{
		"from":     fromMs,
		"to":       toMs,
		"group_by": composioBreakdownGroupBy,
		"limit":    composioBreakdownLimit,
	}
	url := composioAPIURL + "/org/usage/" + composioBreakdownEntityType
	data, err := composioRequest(ctx, http.MethodPost, url, c.Cfg.ComposioAPIKey, body, c.Cfg.HTTPTimeout)
	if err != nil {
		return composioUsageBreakdownResponse{}, err
	}

	var resp composioUsageBreakdownResponse
	if err = json.Unmarshal(data, &resp); err != nil {
		return composioUsageBreakdownResponse{}, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info("Composio usage breakdown request successful")
	return resp, nil
}

func (c ComposioCollector) fetchProjectsTotal(
	ctx context.Context,
) (composioProjectListResponse, error) {
	url := composioAPIURL + "/org/owner/project/list?limit=1"
	data, err := composioRequest(ctx, http.MethodGet, url, c.Cfg.ComposioAPIKey, nil, c.Cfg.HTTPTimeout)
	if err != nil {
		return composioProjectListResponse{}, err
	}

	var resp composioProjectListResponse
	if err = json.Unmarshal(data, &resp); err != nil {
		return composioProjectListResponse{}, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info("Composio project list request successful")
	return resp, nil
}

func composioRequest(
	ctx context.Context,
	method, url, apiKey string,
	body any,
	timeoutSeconds int,
) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("x-org-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK response: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	return data, nil
}

func composioMTDWindow(now time.Time) (int64, int64) {
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return from.UnixMilli(), now.UnixMilli()
}

func parseFloatOrZero(s string) float64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
