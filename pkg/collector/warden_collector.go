package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/warden-protocol/wardenprotocol/warden/x/warden/types/v1beta2"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	"github.com/warden-protocol/warden-exporter/pkg/grpc"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	spacesMetricName              = "warden_spaces"
	keysEcdsaMetricName           = "warden_keys_ecdsa"
	keysEddsaMetricName           = "warden_keys_eddsa"
	keysPendingMetricName         = "warden_keys_pending"
	keychainsMetricName           = "warden_keychains"
	keychainRequestsName          = "warden_keychain_requests"
	keychainName                  = "warden_keychain"
	keychainSignatureRequestsName = "warden_keychain_signature_requests"
	successStatus                 = "success"
	errorStatus                   = "error"
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var spaces = prometheus.NewDesc(
	spacesMetricName,
	"Returns the number of Spaces existing in chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var ecdsaKeys = prometheus.NewDesc(
	keysEcdsaMetricName,
	"Returns the number of ECDSA keys existing in chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var eddsaKeys = prometheus.NewDesc(
	keysEddsaMetricName,
	"Returns the number of EDDSA keys existing in chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var pendingKeys = prometheus.NewDesc(
	keysPendingMetricName,
	"Returns the number of pending KeyRequests existing in chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var keychains = prometheus.NewDesc(
	keychainsMetricName,
	"Returns the number of Keychains existing in chain",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var keychainRequests = prometheus.NewDesc(
	keychainRequestsName,
	"Returns the number of Keychain requests per Keychain",
	[]string{
		"chain_id",
		"keychain_id",
		"keychain_name",
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var keychain = prometheus.NewDesc(
	keychainName,
	"Returns keychain information",
	[]string{
		"chain_id",
		"keychain_id",
		"description",
		"admins",
		"parties",
		"admin_intent_id",
		"fees",
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var keychainSignatureRequests = prometheus.NewDesc(
	keychainSignatureRequestsName,
	"Returns Keychain Signature Requests",
	[]string{
		"chain_id",
		"keychain_id",
		"keychain_name",
		"status",
	},
	nil,
)

type WardenCollector struct {
	Cfg config.Config
}

func (w WardenCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- spaces
	ch <- ecdsaKeys
	ch <- eddsaKeys
	ch <- pendingKeys
	ch <- keychains
	ch <- keychainRequests
	ch <- keychain
	ch <- keychainSignatureRequests
}

func (w WardenCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(w.Cfg.Timeout)*time.Second,
	)

	defer cancel()

	status := successStatus

	client, err := grpc.NewClient(w.Cfg)
	if err != nil {
		log.Error(fmt.Sprintf("error getting spaces metrics: %s", err))
	}

	spacesAmount, err := client.Spaces(ctx)
	if err != nil {
		status = errorStatus

		log.Error(err.Error())
	}

	ch <- prometheus.MustNewConstMetric(
		spaces,
		prometheus.GaugeValue,
		float64(spacesAmount),
		[]string{
			w.Cfg.ChainID,
			status,
		}...,
	)

	ecdsa, eddsa, pending, err := client.Keys(ctx)
	if err != nil {
		status = errorStatus

		log.Error(err.Error())
	}
	ch <- prometheus.MustNewConstMetric(
		ecdsaKeys,
		prometheus.GaugeValue,
		float64(ecdsa),
		[]string{
			w.Cfg.ChainID,
			status,
		}...,
	)
	ch <- prometheus.MustNewConstMetric(
		eddsaKeys,
		prometheus.GaugeValue,
		float64(eddsa),
		[]string{
			w.Cfg.ChainID,
			status,
		}...,
	)
	ch <- prometheus.MustNewConstMetric(
		pendingKeys,
		prometheus.GaugeValue,
		float64(pending),
		[]string{
			w.Cfg.ChainID,
			status,
		}...,
	)

	keyChainsAmount, err := client.Keychains(ctx)
	if err != nil {
		status = errorStatus

		log.Error(err.Error())
	}

	ch <- prometheus.MustNewConstMetric(
		keychains,
		prometheus.GaugeValue,
		float64(keyChainsAmount),
		[]string{
			w.Cfg.ChainID,
			status,
		}...,
	)

	var keychainRequestsAmount uint64
	for x := 1; x <= int(keyChainsAmount); x++ {
		var keychainResponse v1beta2.Keychain
		status = successStatus
		keychainRequestsAmount, err = client.KeychainRequests(ctx, uint64(x))
		if err != nil {
			log.Error(err.Error())
			status = errorStatus
		}
		keychainResponse, err = client.KeyChain(ctx, uint64(x))
		if err != nil {
			log.Error(err.Error())
			status = errorStatus
		}
		ch <- prometheus.MustNewConstMetric(
			keychainRequests,
			prometheus.GaugeValue,
			float64(keychainRequestsAmount),
			[]string{
				w.Cfg.ChainID,
				fmt.Sprintf("%d", x),
				keychainResponse.Description,
				status,
			}...,
		)

		var boolStatus float64
		if keychainResponse.IsActive {
			boolStatus = 1
		} else {
			boolStatus = 0
		}
		ch <- prometheus.MustNewConstMetric(
			keychain,
			prometheus.GaugeValue,
			boolStatus,
			[]string{
				w.Cfg.ChainID,
				fmt.Sprintf("%d", keychainResponse.Id),
				keychainResponse.Description,
				fmt.Sprintf("%v", keychainResponse.Admins),
				fmt.Sprintf("%v", keychainResponse.Parties),
				fmt.Sprintf("%v", keychainResponse.AdminIntentId),
				keychainResponse.Fees.String(),
				status,
			}...,
		)

		// Signature Requests
		var keychainSignaturesResponse uint64
		keychainSignaturesResponse, err = client.KeychainSignatureRequests(ctx, uint64(x))
		if err != nil {
			log.Error(err.Error())
			status = errorStatus
		}
		ch <- prometheus.MustNewConstMetric(
			keychainSignatureRequests,
			prometheus.GaugeValue,
			float64(keychainSignaturesResponse),
			[]string{
				w.Cfg.ChainID,
				fmt.Sprintf("%d", keychainResponse.Id),
				keychainResponse.Description,
				status,
			}...,
		)
	}
}
