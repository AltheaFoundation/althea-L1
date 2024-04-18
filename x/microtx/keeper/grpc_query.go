package keeper

import (
	"context"
	"fmt"
	"strings"

	"github.com/AltheaFoundation/althea-L1/x/microtx/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerror "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
)

// nolint: exhaustruct
// Enforce via type assertion that the Keeper functions as a query server
var _ types.QueryServer = Keeper{}

// Params queries the params of the microtx module
func (k Keeper) Params(c context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	p, err := k.GetParamsIfSet(sdk.UnwrapSDKContext(c))
	if err != nil {
		return nil, err // Force an empty response on error
	}
	return &types.QueryParamsResponse{Params: p}, nil
}

// MicrotxFee computes the amount which will be charged in fees for a given Microtx amount
func (k Keeper) MicrotxFee(c context.Context, req *types.QueryMicrotxFeeRequest) (*types.QueryMicrotxFeeResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	microtxFeeBasisPoints, err := k.GetMicrotxFeeBasisPoints(ctx)
	if err != nil {
		return nil, err
	}
	fee := k.getMicrotxFeeForAmount(sdk.NewIntFromUint64(req.Amount), microtxFeeBasisPoints)
	return &types.QueryMicrotxFeeResponse{FeeAmount: fee.Uint64()}, nil
}

// LiquidAccount implements types.QueryServer.
func (k Keeper) LiquidAccount(c context.Context, req *types.QueryLiquidAccountRequest) (*types.QueryLiquidAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	byOwner := len(req.Owner) > 0
	byLiquidAccount := len(req.Account) > 0
	byNFTAddress := len(req.Nft) > 0

	if byOwner && byLiquidAccount && byNFTAddress {
		return nil, fmt.Errorf("all details already provided, not searching for account")
	}

	// Detect if the owner is EIP-55 style (0xD34DB33F...) or bech32 style (althea1...)
	if byOwner {
		ethOwner := strings.HasPrefix(req.Owner, "0x")
		cosmosPrefix := sdk.GetConfig().GetBech32AccountAddrPrefix()
		cosmosOwner := strings.HasPrefix(req.Owner, cosmosPrefix)
		if ethOwner && !cosmosOwner {
			ethOwnerAddr := common.HexToAddress(req.Owner)
			accs, err := k.GetLiquidAccountsByEVMOwner(ctx, ethOwnerAddr)
			if err != nil {
				return nil, err
			}

			return &types.QueryLiquidAccountResponse{Accounts: accs}, nil
		} else if cosmosOwner && !ethOwner {
			reqAcc, err := sdk.AccAddressFromBech32(req.Owner)
			if err != nil {
				return nil, err
			}

			accs, err := k.GetLiquidAccountsByCosmosOwner(ctx, reqAcc)
			if err != nil {
				return nil, err
			}

			return &types.QueryLiquidAccountResponse{Accounts: accs}, nil
		} else {
			return nil, sdkerror.Wrapf(sdkerror.ErrInvalidAddress, "owner must start with 0x (eip-55) or %v (bech32)", cosmosPrefix)
		}
	}

	if byLiquidAccount {
		reqAcc, err := sdk.AccAddressFromBech32(req.Account)
		if err != nil {
			return nil, err
		}
		acc, err := k.GetLiquidAccount(ctx, reqAcc)
		if err != nil {
			return nil, err
		}
		return &types.QueryLiquidAccountResponse{Accounts: []*types.LiquidInfrastructureAccount{acc}}, nil
	}

	if byNFTAddress {
		nftEVMAddr := common.HexToAddress(req.Nft)
		acc, err := k.GetLiquidAccountByNFTAddress(ctx, nftEVMAddr)
		if err != nil {
			return nil, err
		}
		return &types.QueryLiquidAccountResponse{Accounts: []*types.LiquidInfrastructureAccount{acc}}, nil
	}

	return nil, sdkerror.ErrInvalidRequest
}

// LiquidAccounts fetches all of the known liquid infrastructure accounts
// TODO: Implement pagination
func (k Keeper) LiquidAccounts(c context.Context, req *types.QueryLiquidAccountsRequest) (*types.QueryLiquidAccountsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	accounts, err := k.CollectLiquidAccounts(ctx)

	return &types.QueryLiquidAccountsResponse{Accounts: accounts}, err
}
