package grpc

import (
	context "context"

	"github.com/cosmos/cosmos-sdk/types/query"
	act "github.com/warden-protocol/wardenprotocol/warden/x/act/types/v1beta1"
	warden "github.com/warden-protocol/wardenprotocol/warden/x/warden/types/v1beta3"
)

const (
	keyPageLimit = 200000
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
		req := warden.QueryAllKeysRequest{Pagination: &query.PageRequest{Key: key, Limit: keyPageLimit}}

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

func (c Client) KeyChain(ctx context.Context, id uint64) (warden.Keychain, error) {
	client := warden.NewQueryClient(c.conn)
	req := warden.QueryKeychainByIdRequest{Id: id}
	keychain, err := client.KeychainById(ctx, &req)
	if err != nil {
		return warden.Keychain{}, endpointError(err.Error())
	}
	return *keychain.Keychain, nil
}

func (c Client) KeychainRequests(ctx context.Context, id uint64) (uint64, error) {
	var key []byte
	client := warden.NewQueryClient(c.conn)
	req := warden.QueryKeyRequestsRequest{KeychainId: id, Pagination: &query.PageRequest{Key: key}}

	keychainRequests, err := client.KeyRequests(ctx, &req)
	if err != nil {
		return 0, endpointError(err.Error())
	}

	return keychainRequests.Pagination.Total, nil
}

func (c Client) KeychainSignatureRequests(ctx context.Context, id uint64) (uint64, error) {
	var key []byte
	client := warden.NewQueryClient(c.conn)
	req := warden.QuerySignRequestsRequest{KeychainId: id, Pagination: &query.PageRequest{Key: key}}

	keychainRequests, err := client.SignRequests(ctx, &req)
	if err != nil {
		return 0, endpointError(err.Error())
	}

	return keychainRequests.Pagination.Total, nil
}

// Actions.
func (c Client) Actions(ctx context.Context) (uint64, error) {
	var key []byte

	client := act.NewQueryClient(c.conn)

	req := act.QueryActionsRequest{Pagination: &query.PageRequest{Key: key}}

	actions, err := client.Actions(ctx, &req)
	if err != nil {
		return 0, endpointError(err.Error())
	}

	return actions.Pagination.Total, nil
}

// Rules.
func (c Client) Rules(ctx context.Context) (uint64, error) {
	var key []byte

	client := act.NewQueryClient(c.conn)

	req := act.QueryRulesRequest{Pagination: &query.PageRequest{Key: key}}

	actions, err := client.Rules(ctx, &req)
	if err != nil {
		return 0, endpointError(err.Error())
	}

	return actions.Pagination.Total, nil
}
