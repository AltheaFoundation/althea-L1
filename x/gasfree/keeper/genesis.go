package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/althea-net/althea-L1/x/gasfree/types"
)

// InitGenesis starts a chain from a genesis state
func InitGenesis(ctx sdk.Context, k Keeper, data types.GenesisState) {
	params := data.Params
	k.SetGasFreeMessageTypes(ctx, params.GetGasFreeMessageTypes())
}

// ExportGenesis exports all the state needed to restart the chain
// from the current state of the chain
func ExportGenesis(ctx sdk.Context, k Keeper) types.GenesisState {
	return types.GenesisState{
		Params: &types.Params{
			GasFreeMessageTypes: k.GetGasFreeMessageTypes(ctx),
		},
	}
}
