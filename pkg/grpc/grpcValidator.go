package grpc

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	base "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/query"
	slashing "github.com/cosmos/cosmos-sdk/x/slashing/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
	types "github.com/warden-protocol/warden-exporter/pkg/types"
)

const (
	valConsStr       = "valcons"
	bondStatusPrefix = "BOND_STATUS_"
	prefix           = "warden"
)

func (c Client) SignigInfos(ctx context.Context) ([]slashing.ValidatorSigningInfo, error) {
	infos := []slashing.ValidatorSigningInfo{}
	key := []byte{}
	client := slashing.NewQueryClient(c.conn)

	for {
		request := &slashing.QuerySigningInfosRequest{Pagination: &query.PageRequest{Key: key}}

		slashRes, err := client.SigningInfos(ctx, request)
		if err != nil {
			return nil, endpointError(err.Error())
		}

		if slashRes == nil {
			return nil, endpointError("got empty response from signing infos endpoint")
		}

		infos = append(infos, slashRes.GetInfo()...)

		page := slashRes.GetPagination()
		if page == nil {
			break
		}

		key = page.GetNextKey()
		if len(key) == 0 {
			break
		}
	}

	log.Debug(fmt.Sprintf("SigningInfos: %d", len(infos)))

	return infos, nil
}

func (c Client) Validators(ctx context.Context) ([]staking.Validator, error) {
	vals := []staking.Validator{}
	key := []byte{}

	// https://github.com/cosmos/cosmos-sdk/issues/8045#issuecomment-829142440
	encCfg := testutil.MakeTestEncodingConfig()
	interfaceRegistry := encCfg.InterfaceRegistry

	client := staking.NewQueryClient(c.conn)

	for {
		request := &staking.QueryValidatorsRequest{Pagination: &query.PageRequest{Key: key}}

		stakingRes, err := client.Validators(ctx, request)
		if err != nil {
			return nil, endpointError(err.Error())
		}

		if stakingRes == nil {
			return nil, endpointError("got empty response from validators endpoint")
		}

		for _, val := range stakingRes.GetValidators() {
			err = val.UnpackInterfaces(interfaceRegistry)
			if err != nil {
				return nil, endpointError(err.Error())
			}

			vals = append(vals, val)
		}

		page := stakingRes.GetPagination()
		if page == nil {
			break
		}

		key = page.GetNextKey()
		if len(key) == 0 {
			break
		}
	}

	log.Debug(fmt.Sprintf("Validators: %d", len(vals)))

	return vals, nil
}

func (c Client) valConsMap(vals []staking.Validator) (map[string]staking.Validator, error) {
	vMap := map[string]staking.Validator{}

	for _, val := range vals {
		addr, err := val.GetConsAddr()
		if err != nil {
			return nil, endpointError(err.Error())
		}

		consAddr, err := bech32.ConvertAndEncode(prefix+valConsStr, sdk.ConsAddress(addr))
		if err != nil {
			return nil, endpointError(err.Error())
		}

		vMap[consAddr] = val
	}

	return vMap, nil
}

func SigningValidators(ctx context.Context, cfg config.Config) ([]types.Validator, error) {
	sVals := []types.Validator{}

	client, err := NewClient(cfg)
	if err != nil {
		log.Error(err.Error())

		return []types.Validator{}, endpointError(err.Error())
	}

	sInfos, err := client.SignigInfos(ctx)

	defer func() {
		if tempErr := client.conn.Close(); tempErr != nil {
			log.Error(tempErr.Error())
		}
	}()

	if err != nil {
		log.Error(err.Error())

		return []types.Validator{}, endpointError(err.Error())
	}

	vals, err := client.Validators(ctx)
	if err != nil {
		log.Error(err.Error())

		return []types.Validator{}, endpointError(err.Error())
	}

	valsMap, err := client.valConsMap(vals)
	if err != nil {
		log.Error(err.Error())

		return []types.Validator{}, endpointError(err.Error())
	}

	for _, info := range sInfos {
		if _, ok := valsMap[info.Address]; !ok {
			log.Debug(fmt.Sprintf("Not in validators: %s", info.Address))
		}

		val := valsMap[info.Address]

		// Convert math.Int to float64 via big.Int
		tokensBigInt := val.Tokens.BigInt()
		tokens, _ := new(big.Float).SetInt(tokensBigInt).Float64()

		// Convert math.LegacyDec to float64
		delegatorShares, _ := val.DelegatorShares.Float64()

		sVals = append(sVals, types.Validator{
			ConsAddress:     info.Address,
			OperatorAddress: val.OperatorAddress,
			Moniker:         val.Description.Moniker,
			MissedBlocks:    info.MissedBlocksCounter,
			Jailed:          val.IsJailed(),
			Tombstoned:      info.Tombstoned,
			BondStatus:      bondStatus(val.GetStatus()),
			Tokens:          tokens,
			DelegatorShares: delegatorShares,
		})
	}

	return sVals, nil
}

func LatestBlockHeight(ctx context.Context, cfg config.Config) (int64, error) {
	client, err := NewClient(cfg)
	if err != nil {
		log.Error(err.Error())

		return 0, endpointError(err.Error())
	}

	request := &base.GetLatestBlockRequest{}
	baseClient := base.NewServiceClient(client.conn)

	blockResp, err := baseClient.GetLatestBlock(ctx, request)

	defer func() {
		if tempErr := client.conn.Close(); tempErr != nil {
			log.Error(tempErr.Error())
		}
	}()

	if err != nil {
		log.Error(err.Error())

		return 0, endpointError(err.Error())
	}

	height := blockResp.GetBlock().Header.Height
	log.Debug(fmt.Sprintf("Latest height: %d", height))

	return height, nil
}

func BlockProposers(ctx context.Context, cfg config.Config, blockCount int64) (map[string]int64, error) {
	client, err := NewClient(cfg)
	if err != nil {
		log.Error(err.Error())

		return nil, endpointError(err.Error())
	}

	baseClient := base.NewServiceClient(client.conn)

	// Get latest block height first
	latestReq := &base.GetLatestBlockRequest{}
	latestResp, err := baseClient.GetLatestBlock(ctx, latestReq)

	defer func() {
		if tempErr := client.conn.Close(); tempErr != nil {
			log.Error(tempErr.Error())
		}
	}()

	if err != nil {
		log.Error(err.Error())

		return nil, endpointError(err.Error())
	}

	latestHeight := latestResp.GetBlock().Header.Height
	startHeight := latestHeight - blockCount
	if startHeight < 1 {
		startHeight = 1
	}

	// Count proposers
	proposerCounts := make(map[string]int64)

	for height := startHeight; height <= latestHeight; height++ {
		blockReq := &base.GetBlockByHeightRequest{Height: height}
		blockResp, err := baseClient.GetBlockByHeight(ctx, blockReq)

		if err != nil {
			log.Debug(fmt.Sprintf("Error fetching block %d: %s", height, err.Error()))
			continue
		}

		if blockResp.GetBlock() != nil && blockResp.GetBlock().Header != nil {
			proposerAddr := blockResp.GetBlock().Header.ProposerAddress

			// Convert proposer address bytes to valcons address
			consAddr, err := bech32.ConvertAndEncode(prefix+valConsStr, proposerAddr)
			if err != nil {
				log.Debug(fmt.Sprintf("Error converting proposer address at height %d: %s", height, err.Error()))
				continue
			}

			proposerCounts[consAddr]++
		}
	}

	log.Debug(fmt.Sprintf("Scanned blocks %d to %d, found %d unique proposers", startHeight, latestHeight, len(proposerCounts)))

	return proposerCounts, nil
}

func AverageBlockTime(ctx context.Context, cfg config.Config, sampleSize int64) (float64, error) {
	client, err := NewClient(cfg)
	if err != nil {
		log.Error(err.Error())

		return 0, endpointError(err.Error())
	}

	baseClient := base.NewServiceClient(client.conn)

	// Get latest block height first
	latestReq := &base.GetLatestBlockRequest{}
	latestResp, err := baseClient.GetLatestBlock(ctx, latestReq)

	defer func() {
		if tempErr := client.conn.Close(); tempErr != nil {
			log.Error(tempErr.Error())
		}
	}()

	if err != nil {
		log.Error(err.Error())

		return 0, endpointError(err.Error())
	}

	latestHeight := latestResp.GetBlock().Header.Height
	startHeight := latestHeight - sampleSize
	if startHeight < 1 {
		startHeight = 1
	}

	// Get start block
	startBlockReq := &base.GetBlockByHeightRequest{Height: startHeight}
	startBlockResp, err := baseClient.GetBlockByHeight(ctx, startBlockReq)
	if err != nil {
		log.Error(err.Error())

		return 0, endpointError(err.Error())
	}

	// Get end block (latest)
	endBlockReq := &base.GetBlockByHeightRequest{Height: latestHeight}
	endBlockResp, err := baseClient.GetBlockByHeight(ctx, endBlockReq)
	if err != nil {
		log.Error(err.Error())

		return 0, endpointError(err.Error())
	}

	startTime := startBlockResp.GetBlock().Header.Time.AsTime()
	endTime := endBlockResp.GetBlock().Header.Time.AsTime()

	timeDiff := endTime.Sub(startTime).Seconds()
	blockCount := float64(latestHeight - startHeight)

	if blockCount == 0 {
		return 0, endpointError("invalid block count")
	}

	avgBlockTime := timeDiff / blockCount

	log.Debug(fmt.Sprintf("Average block time: %.2f seconds (sampled %d blocks)", avgBlockTime, int64(blockCount)))

	return avgBlockTime, nil
}

func bondStatus(status staking.BondStatus) string {
	statusWithoutPrefix, _ := strings.CutPrefix(status.String(), bondStatusPrefix)

	return strings.ToLower(statusWithoutPrefix)
}
