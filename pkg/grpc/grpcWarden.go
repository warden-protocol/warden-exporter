package grpc

import (
	context "context"

	"github.com/cosmos/cosmos-sdk/types/query"
	intent "github.com/warden-protocol/wardenprotocol/warden/x/intent/types"
	warden "github.com/warden-protocol/wardenprotocol/warden/x/warden/types/v1beta2"
)

// spaces metric.
func (c Client) Spaces(ctx context.Context) (uint64, error) {
	client := warden.NewQueryClient(c.conn)
	req := warden.QuerySpacesRequest{Pagination: &query.PageRequest{
		Limit:      1,
		CountTotal: true,
	}}

	spacesRes, err := client.Spaces(ctx, &req)
	if err != nil {
		return 0, endpointError(err.Error())
	}

	return spacesRes.Pagination.Total, nil
}

// keys metric.
func (c Client) Keys(ctx context.Context) (uint64, uint64, uint64, error) {
	var (
		// 	addressTypes []warden.AddressType
		pendingKeys uint64
		ecdsaKeys   uint64
		eddsaKeys   uint64
		key         []byte
	)

	client := warden.NewQueryClient(c.conn)

	for {
		req := warden.QueryAllKeysRequest{Pagination: &query.PageRequest{Key: key}}

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

// Keychains metric.
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

// Intents.
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

// Actions.
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
