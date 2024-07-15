package grpc

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/query"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"

	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

// Accounts.
func (c Client) Accounts(ctx context.Context) (uint64, error) {
	log.Info("grpc: fetching accounts")
	var key []byte

	client := auth.NewQueryClient(c.conn)

	req := auth.QueryAccountsRequest{Pagination: &query.PageRequest{Key: key}}

	accounts, err := client.Accounts(ctx, &req)
	if err != nil {
		return 0, endpointError(err.Error())
	}

	log.Info("grpc: fetching accounts complete")
	return accounts.Pagination.Total, nil
}
