package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/AltheaFoundation/althea-L1/x/gasfree/types"
)

// InitGenesis starts a chain from a genesis state
func InitGenesis(ctx sdk.Context, k Keeper, data types.GenesisState) {
	params := data.Params
	k.SetGasFreeMessageTypes(ctx, params.GetGasFreeMessageTypes())
	k.SetGasfreeErc20InteropTokens(ctx, params.GetGasFreeErc20InteropTokens())
	k.SetGasfreeErc20InteropFeeBasisPoints(ctx, params.GetGasFreeErc20InteropFeeBasisPoints())
}

// ExportGenesis exports all the state needed to restart the chain
// from the current state of the chain
func ExportGenesis(ctx sdk.Context, k Keeper) types.GenesisState {
	params, err := k.GetParamsIfSet(ctx)
	if err != nil {
		panic(err)
	}
	return types.GenesisState{
		Params: &params,
	}
}
