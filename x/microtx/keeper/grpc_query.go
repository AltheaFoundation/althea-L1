package keeper

import (
	"context"

	"github.com/althea-net/althea-chain/x/microtx/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

// XferFee computes the amount which will be charged in fees for a given Xfer amount
func (k Keeper) XferFee(c context.Context, req *types.QueryXferFeeRequest) (*types.QueryXferFeeResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	xferFeeBasisPoints, err := k.GetXferFeeBasisPoints(ctx)
	if err != nil {
		return nil, err
	}
	fee := k.getXferFeeForAmount(sdk.NewIntFromUint64(req.Amount), xferFeeBasisPoints)
	return &types.QueryXferFeeResponse{FeeAmount: fee.Uint64()}, nil
}
