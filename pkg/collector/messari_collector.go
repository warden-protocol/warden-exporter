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
	messariCreditMetricName = "messari_credit"
	messariAPIURL           = "https://api.messari.io"
)

type MessariCreditResponse struct {
	Error string `json:"error"`
	Data  struct {
		TeamID           uint64 `json:"teamId"`
		CreditsAllocated uint64 `json:"creditsAllocated"`
		StartDate        string `json:"startDate"`
		IsActive         bool   `json:"isActive"`
		EndDate          string `json:"endDate"`
		RemainingCredits uint64 `json:"remainingCredits"`
	} `json:"data"`
}

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var messariCredits = prometheus.NewDesc(
	messariCreditMetricName,
	"Returns Messari API Key credit information",
	[]string{
		"team_id",
		"is_active",
		"credits_allocated",
		"status",
	},
	nil,
)

type MessariCollector struct {
	Cfg config.Config
}

func (m MessariCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- messariCredits
}

func (m MessariCollector) Collect(ch chan<- prometheus.Metric) {
	var err error
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(m.Cfg.Timeout)*time.Second,
	)
	defer cancel()

	status := successStatus

	response, err := m.messariCollectCredits(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("error collecting Venice balance: %s", err))
		status = errorStatus
	}

	ch <- prometheus.MustNewConstMetric(
		messariCredits,
		prometheus.GaugeValue,
		float64(response.Data.RemainingCredits),
		[]string{
			fmt.Sprintf("%d", response.Data.TeamID),
			fmt.Sprintf("%t", response.Data.IsActive),
			fmt.Sprintf("%d", response.Data.CreditsAllocated),
			status,
		}...,
	)
}

func (m MessariCollector) messariCollectCredits(
	ctx context.Context,
) (MessariCreditResponse, error) {
	url := fmt.Sprintf("%s/user-management/v1/credits/allowance", messariAPIURL)
	data, err := http.GetRequest(ctx, url, m.Cfg.MessariAPIKey)
	if err != nil {
		return MessariCreditResponse{}, err
	}

	var messariResponse MessariCreditResponse
	if err = json.NewDecoder(bytes.NewReader(data)).Decode(&messariResponse); err != nil {
		return MessariCreditResponse{}, fmt.Errorf("error decoding response: %w", err)
	}

	log.Info("Messari API credits request successful")

	return messariResponse, nil
}
