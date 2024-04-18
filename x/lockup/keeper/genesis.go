package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/AltheaFoundation/althea-L1/x/lockup/types"
)

// InitGenesis starts a chain from a genesis state
func InitGenesis(ctx sdk.Context, k Keeper, data types.GenesisState) {
	params := data.Params
	k.SetChainLocked(ctx, params.GetLocked())
	k.SetLockExemptAddresses(ctx, params.GetLockExempt())
	k.SetLockedMessageTypes(ctx, params.GetLockedMessageTypes())
	k.SetLockedTokenDenoms(ctx, params.LockedTokenDenoms)
}

// ExportGenesis exports all the state needed to restart the chain
// from the current state of the chain
func ExportGenesis(ctx sdk.Context, k Keeper) types.GenesisState {
	return types.GenesisState{
		Params: &types.Params{
			Locked:             k.GetChainLocked(ctx),
			LockExempt:         k.GetLockExemptAddresses(ctx),
			LockedMessageTypes: k.GetLockedMessageTypes(ctx),
			LockedTokenDenoms:  k.GetLockedTokenDenoms(ctx),
		},
	}
}
