package grpc

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/query"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// Accounts.
func (c Client) Accounts(ctx context.Context) (uint64, error) {
	var key []byte

	client := auth.NewQueryClient(c.conn)

	req := auth.QueryAccountsRequest{Pagination: &query.PageRequest{Key: key}}

	accounts, err := client.Accounts(ctx, &req)
	if err != nil {
		return 0, endpointError(err.Error())
	}

	return accounts.Pagination.Total, nil
}
