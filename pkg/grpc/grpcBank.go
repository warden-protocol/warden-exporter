package grpc

import (
	"context"

	bank "github.com/cosmos/cosmos-sdk/x/bank/types"

	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

func (c Client) Balance(ctx context.Context, address, denom string) (uint64, error) {
	log.Info("grpc: fetching balance")
	client := bank.NewQueryClient(c.conn)

	req := bank.QueryBalanceRequest{Address: address, Denom: denom}

	balance, err := client.Balance(ctx, &req)
	if err != nil {
		return 0, endpointError(err.Error())
	}

	log.Info("grpc: fetching balance complete")
	return balance.Balance.Amount.Uint64(), nil
}
