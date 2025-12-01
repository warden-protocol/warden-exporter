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

func (c Client) TotalSupply(ctx context.Context, denom string) (string, error) {
	client := bank.NewQueryClient(c.conn)

	req := bank.QuerySupplyOfRequest{Denom: denom}

	resp, err := client.SupplyOf(ctx, &req)
	if err != nil {
		return "", endpointError(err.Error())
	}

	// The Amount field in the SDK types is math.Int, we need to convert to string
	return resp.Amount.Amount.String(), nil
}
