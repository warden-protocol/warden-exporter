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
)

const (
	errorStatus   = "error"
	successStatus = "success"
)

type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	ID      int    `json:"id"`
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

func getBalance(ctx context.Context, rpc, address string, timeout int) (float64, error) {
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_getBalance",
		Params:  []any{address, "latest"},
		ID:      1,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		rpc,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return 0, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
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

	// Convert Wei to None
	weiPerEth := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	balanceFloat := new(big.Float).SetInt(balanceWei)
	balanceEth := new(big.Float).Quo(balanceFloat, weiPerEth)

	// Convert to float64
	balance, _ := balanceEth.Float64()

	return balance, nil
}
