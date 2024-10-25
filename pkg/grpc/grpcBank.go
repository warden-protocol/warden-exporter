package grpc

import (
	"context"

	"cosmossdk.io/math"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (c Client) Balance(ctx context.Context, address, denom string) (math.Int, error) {
	client := bank.NewQueryClient(c.conn)

	req := bank.QueryBalanceRequest{Address: address, Denom: denom}

	balance, err := client.Balance(ctx, &req)
	if err != nil {
		return math.Int{}, endpointError(err.Error())
	}

	return balance.Balance.Amount, nil
}
