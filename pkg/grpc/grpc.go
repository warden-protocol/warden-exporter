package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/query"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	intent "github.com/warden-protocol/wardenprotocol/warden/x/intent/types"
	warden "github.com/warden-protocol/wardenprotocol/warden/x/warden/types/v1beta2"
	"google.golang.org/grpc"

	"github.com/warden-protocol/warden-exporter/pkg/config"
)

var errEndpoint = errors.New("grpc error")

func endpointError(msg string) error {
	return fmt.Errorf("%w: %s", errEndpoint, msg)
}

type Client struct {
	cfg  config.Config
	conn *grpc.ClientConn
}

func NewClient(cfg config.Config) (Client, error) {
	client := Client{
		cfg: cfg,
	}

	conn, err := cfg.GRPCConn()
	if err != nil {
		return Client{}, endpointError(err.Error())
	}

	client.conn = conn

	return client, nil
}

// spaces metric
func (c Client) Spaces(ctx context.Context) (uint64, error) {
	client := warden.NewQueryClient(c.conn)
	req := warden.QuerySpacesRequest{Pagination: &query.PageRequest{
		Limit: 1,
		CountTotal: true,
	}}

	spacesRes, err := client.Spaces(ctx, &req)
	if err != nil {
		return 0, endpointError(err.Error())
	}

	return spacesRes.Pagination.Total, nil
}

// keys metric
func (c Client) Keys(ctx context.Context) (uint64, uint64, uint64, error) {
	var (
		addressTypes []warden.AddressType
		pendingKeys  uint64
		ecdsaKeys    uint64
		eddsaKeys    uint64
		key          []byte
	)

	client := warden.NewQueryClient(c.conn)

	for _, k := range warden.AddressType_value {
		addressTypes = append(addressTypes, warden.AddressType(k))
	}

	for {
		req := warden.QueryAllKeysRequest{Pagination: &query.PageRequest{Key: key}, DeriveAddresses: addressTypes}

		allKeys, err := client.AllKeys(ctx, &req)
		if err != nil {
			return 0, 0, 0, endpointError(err.Error())
		}

		for _, key := range allKeys.Keys {
			if key.Key.Type == warden.KeyType_KEY_TYPE_ECDSA_SECP256K1 {
				ecdsaKeys++
				continue
			}

			if key.Key.Type == warden.KeyType_KEY_TYPE_EDDSA_ED25519 {
				eddsaKeys++
				continue
			}

			pendingKeys++
		}

		if allKeys.GetPagination() == nil {
			break
		}

		key = allKeys.Pagination.GetNextKey()
		if len(key) == 0 {
			break
		}
	}

	return ecdsaKeys, eddsaKeys, pendingKeys, nil
}

// Keychains metric
func (c Client) Keychains(ctx context.Context) (uint64, error) {
	var key []byte

	client := warden.NewQueryClient(c.conn)

	req := warden.QueryKeychainsRequest{Pagination: &query.PageRequest{Key: key}}

	keyChains, err := client.Keychains(ctx, &req)
	if err != nil {
		return 0, endpointError(err.Error())
	}

	return keyChains.Pagination.Total, nil
}

// Intents
func (c Client) Intents(ctx context.Context) (uint64, error) {
	var key []byte

	client := intent.NewQueryClient(c.conn)

	req := intent.QueryIntentsRequest{Pagination: &query.PageRequest{Key: key}}

	intents, err := client.Intents(ctx, &req)
	if err != nil {
		return 0, endpointError(err.Error())
	}

	return intents.Pagination.Total, nil
}

// Actions
func (c Client) Actions(ctx context.Context) (uint64, error) {
	var key []byte

	client := intent.NewQueryClient(c.conn)

	req := intent.QueryActionsRequest{Pagination: &query.PageRequest{Key: key}}

	actions, err := client.Actions(ctx, &req)
	if err != nil {
		return 0, endpointError(err.Error())
	}

	return actions.Pagination.Total, nil
}

// Accounts
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
