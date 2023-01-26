package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/althea-net/althea-chain/x/lockup/types"
)

// nolint: exhaustruct
// Enforce via type assertion that the Keeper functions as a query server
var _ types.QueryServer = Keeper{}

// Params queries the params of the lockup module
func (k Keeper) Params(c context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	p, err := k.GetParamsIfSet(sdk.UnwrapSDKContext(c))
	if err != nil {
		return nil, err // Force an empty response on error
	}
	return &types.QueryParamsResponse{Params: p}, nil
}
