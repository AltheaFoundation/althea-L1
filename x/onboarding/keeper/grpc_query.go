package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/AltheaFoundation/althea-L1/x/onboarding/types"
)

// nolint: exhaustruct
var _ types.QueryServer = Keeper{}

// Params returns the module parameters
func (k Keeper) Params(
	c context.Context,
	_ *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}
