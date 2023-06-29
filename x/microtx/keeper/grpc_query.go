package keeper

import (
	"context"
	"fmt"
	"strings"

	"github.com/althea-net/althea-chain/x/microtx/types"
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

// TokenizedAccount implements types.QueryServer.
func (k Keeper) TokenizedAccount(c context.Context, req *types.QueryTokenizedAccountRequest) (*types.QueryTokenizedAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	byOwner := len(req.Owner) > 0
	byTokenizedAccount := len(req.TokenizedAccount) > 0
	byNFTAddress := len(req.NftAddress) > 0

	if byOwner && byTokenizedAccount && byNFTAddress {
		return nil, fmt.Errorf("all details already provided, not searching for account")
	}

	// Detect if the owner is EIP-55 style (0xD34DB33F...) or bech32 style (althea1...)
	if byOwner {
		ethOwner := strings.HasPrefix(req.Owner, "0x")
		cosmosPrefix := sdk.GetConfig().GetBech32AccountAddrPrefix()
		cosmosOwner := strings.HasPrefix(req.Owner, cosmosPrefix)
		if ethOwner && !cosmosOwner {
			ethOwnerAddr := common.HexToAddress(req.Owner)
			accs, err := k.GetTokenizedAccountsByEVMOwner(ctx, ethOwnerAddr)
			if err != nil {
				return nil, err
			}

			return &types.QueryTokenizedAccountResponse{Accounts: accs}, nil
		} else if cosmosOwner && !ethOwner {
			reqAcc, err := sdk.AccAddressFromBech32(req.Owner)
			if err != nil {
				return nil, err
			}

			accs, err := k.GetTokenizedAccountsByCosmosOwner(ctx, reqAcc)
			if err != nil {
				return nil, err
			}

			return &types.QueryTokenizedAccountResponse{Accounts: accs}, nil
		} else {
			return nil, sdkerror.Wrapf(sdkerror.ErrInvalidAddress, "owner must start with 0x (eip-55) or %v (bech32)", cosmosPrefix)
		}
	}

	if byTokenizedAccount {
		reqAcc, err := sdk.AccAddressFromBech32(req.TokenizedAccount)
		if err != nil {
			return nil, err
		}
		acc, err := k.GetTokenizedAccount(ctx, reqAcc)
		if err != nil {
			return nil, err
		}
		return &types.QueryTokenizedAccountResponse{Accounts: []*types.TokenizedAccount{acc}}, nil
	}

	if byNFTAddress {
		nftEVMAddr := common.HexToAddress(req.NftAddress)
		acc, err := k.GetTokenizedAccountByNFTAddress(ctx, nftEVMAddr)
		if err != nil {
			return nil, err
		}
		return &types.QueryTokenizedAccountResponse{Accounts: []*types.TokenizedAccount{acc}}, nil
	}

	return nil, sdkerror.ErrInvalidRequest
}

// TokenizedAccounts fetches all of the known tokenized accounts
// TODO: Implement pagination
func (k Keeper) TokenizedAccounts(c context.Context, req *types.QueryTokenizedAccountsRequest) (*types.QueryTokenizedAccountsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	accounts, err := k.CollectTokenizedAccounts(ctx)

	return &types.QueryTokenizedAccountsResponse{Accounts: accounts}, err
}
