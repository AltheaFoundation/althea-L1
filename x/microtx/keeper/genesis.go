package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	microtxtypes "github.com/AltheaFoundation/althea-L1/x/microtx/types"
)

// InitGenesis starts a chain from a genesis state
func InitGenesis(ctx sdk.Context, k Keeper, data microtxtypes.GenesisState) {
	if err := k.SetParams(ctx, *data.Params); err != nil {
		panic(fmt.Sprintf("Unable to set params with error %v", err))
	}
}

// ExportGenesis exports all the state needed to restart the chain
// from the current state of the chain
func ExportGenesis(ctx sdk.Context, k Keeper) microtxtypes.GenesisState {
	p := k.GetParams(ctx)

	return microtxtypes.GenesisState{
		Params: &p,
	}
}
