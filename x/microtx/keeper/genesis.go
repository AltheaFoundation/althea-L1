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

	// We do not care about the initial previous proposer, but it must be set for any other block (including upgrades)
	if ctx.BlockHeight() > 0 {
		if data.PreviousProposer == "" {
			fmt.Println("Previous proposer not set in InitGenesis, block: ", ctx.BlockHeight())
			panic("Previous proposer not set in InitGenesis")
		} else {
			// Convert the previous proposer from an AccAddress (cosmos1...) to a ConsAddress (cosmosvalcons1...)
			accAddr, err := sdk.AccAddressFromBech32(data.PreviousProposer)
			if err != nil {
				fmt.Println("Invalid previous proposer in InitGenesis, block: ", ctx.BlockHeight())
				panic(fmt.Sprintf("Unable to convert proposer address from bech32: %v", err))
			}
			consAddr := sdk.ConsAddress(accAddr)
			k.SetPreviousProposerConsAddr(ctx, consAddr)

		}
	}
}

// ExportGenesis exports all the state needed to restart the chain
// from the current state of the chain
func ExportGenesis(ctx sdk.Context, k Keeper) microtxtypes.GenesisState {
	p := k.GetParams(ctx)

	previousProposer := k.GetPreviousProposerConsAddr(ctx)

	return microtxtypes.GenesisState{
		PreviousProposer: sdk.AccAddress(previousProposer).String(),
		Params:           &p,
	}
}
