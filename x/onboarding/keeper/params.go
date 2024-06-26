package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/AltheaFoundation/althea-L1/x/onboarding/types"
)

// GetParams returns the total set of onboarding parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.ParamStore.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the onboarding parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.ParamStore.SetParamSet(ctx, &params)
}
