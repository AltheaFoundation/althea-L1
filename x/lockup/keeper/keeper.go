package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/althea-net/althea-chain/x/lockup/types"
)

type Keeper struct {
	storeKey   sdk.StoreKey
	paramSpace paramstypes.Subspace
	cdc        codec.BinaryCodec
}

func NewKeeper(cdc codec.BinaryCodec, storeKey sdk.StoreKey, paramSpace paramstypes.Subspace) Keeper {
	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	k := Keeper{
		cdc:        cdc,
		paramSpace: paramSpace,
		storeKey:   storeKey,
	}

	return k
}

// GetParamsIfSet will return the current params, but will return an error if the
// chain is still initializing. By error checking this function is safe to use in
// handling genesis transactions.
func (k Keeper) GetParamsIfSet(ctx sdk.Context) (params types.Params, err error) {
	for _, pair := range params.ParamSetPairs() {
		if !k.paramSpace.Has(ctx, pair.Key) {
			return types.Params{}, sdkerrors.Wrapf(sdkerrors.ErrNotFound, "the param key %s has not been set", string(pair.Key))
		}
		k.paramSpace.Get(ctx, pair.Key, pair.Value)
	}

	return
}

// TODO: Doc all these methods
func (k Keeper) GetChainLocked(ctx sdk.Context) bool {
	locked := types.DefaultParams().Locked
	k.paramSpace.GetIfExists(ctx, types.LockedKey, &locked)
	return locked
}

func (k Keeper) SetChainLocked(ctx sdk.Context, locked bool) {
	k.paramSpace.Set(ctx, types.LockedKey, &locked)
}

func (k Keeper) GetLockExemptAddresses(ctx sdk.Context) []string {
	lockExempt := types.DefaultParams().LockExempt
	k.paramSpace.GetIfExists(ctx, types.LockExemptKey, &lockExempt)
	return lockExempt
}

func (k Keeper) GetLockExemptAddressesSet(ctx sdk.Context) map[string]struct{} {
	return createSet(k.GetLockExemptAddresses(ctx))
}

func (k Keeper) GetLockedTokenDenoms(ctx sdk.Context) []string {
	lockedTokenDenoms := types.DefaultParams().LockedTokenDenoms
	k.paramSpace.GetIfExists(ctx, types.LockedTokenDenomsKey, &lockedTokenDenoms)
	return lockedTokenDenoms
}

func (k Keeper) GetLockedTokenDenomsSet(ctx sdk.Context) map[string]struct{} {
	return createSet(k.GetLockedTokenDenoms(ctx))
}

// TODO: It would be nice to just store the pseudo-set instead of the string array
// so that we get better efficiency on each read (happens each transaction in antehandler)
// however we would need to make a custom param change proposal handler to construct the
// set upon governance proposal before storage in keeper
func (k Keeper) SetLockExemptAddresses(ctx sdk.Context, lockExempt []string) {
	k.paramSpace.Set(ctx, types.LockExemptKey, &lockExempt)
}

func (k Keeper) GetLockedMessageTypes(ctx sdk.Context) []string {
	lockedMessageTypes := types.DefaultParams().LockedMessageTypes
	k.paramSpace.GetIfExists(ctx, types.LockedMessageTypesKey, &lockedMessageTypes)
	return lockedMessageTypes
}

func (k Keeper) GetLockedMessageTypesSet(ctx sdk.Context) map[string]struct{} {
	return createSet(k.GetLockedMessageTypes(ctx))
}

func (k Keeper) SetLockedMessageTypes(ctx sdk.Context, lockedMessageTypes []string) {
	k.paramSpace.Set(ctx, types.LockedMessageTypesKey, &lockedMessageTypes)
}

func (k Keeper) SetLockedTokenDenoms(ctx sdk.Context, lockedTokenDenoms []string) {
	k.paramSpace.Set(ctx, types.LockedTokenDenomsKey, &lockedTokenDenoms)
}

func createSet(strings []string) map[string]struct{} {
	type void struct{}
	var member void
	set := make(map[string]struct{})

	for _, str := range strings {
		if _, present := set[str]; present {
			continue
		}
		set[str] = member
	}

	return set
}
