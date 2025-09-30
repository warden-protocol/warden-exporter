package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	baseBalanceMetricName = "base_wallet_balance"
)

type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      int       `json:"id"`
	Result  string    `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var baseBalance = prometheus.NewDesc(
	baseBalanceMetricName,
	"Returns the wallet balance on Base blockchain",
	[]string{
		"account",
		"status",
	},
	nil,
)

type BaseCollector struct {
	Cfg config.Config
}

func (b BaseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- baseBalance
}

func (b BaseCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(b.Cfg.Timeout)*time.Second,
	)
	defer cancel()

	addresses := strings.Split(b.Cfg.BaseAddresses, ",")
	for _, addr := range addresses {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}

		status := successStatus
		balance, err := b.getBalance(ctx, addr)
		if err != nil {
			log.Error(fmt.Sprintf("error getting balance for address %s: %s", addr, err))
			status = errorStatus
			balance = 0
		}

		ch <- prometheus.MustNewConstMetric(
			baseBalance,
			prometheus.GaugeValue,
			balance,
			[]string{
				addr,
				status,
			}...,
		)
	}
}

func (b BaseCollector) getBalance(ctx context.Context, address string) (float64, error) {
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_getBalance",
		Params:  []interface{}{address, "latest"},
		ID:      1,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		b.Cfg.BaseRPCURL,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return 0, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Duration(b.Cfg.Timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error performing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body: %w", err)
	}

	var rpcResp JSONRPCResponse
	if err = json.Unmarshal(body, &rpcResp); err != nil {
		return 0, fmt.Errorf("error unmarshaling response: %w", err)
	}

	if rpcResp.Error != nil {
		return 0, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	// Convert hex balance to decimal
	balanceWei := new(big.Int)
	if _, success := balanceWei.SetString(strings.TrimPrefix(rpcResp.Result, "0x"), 16); !success {
		return 0, fmt.Errorf("error parsing balance: %s", rpcResp.Result)
	}

	// Convert Wei to ETH
	weiPerEth := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	balanceFloat := new(big.Float).SetInt(balanceWei)
	balanceEth := new(big.Float).Quo(balanceFloat, weiPerEth)

	// Convert to float64
	balance, _ := balanceEth.Float64()

	return balance, nil
}
