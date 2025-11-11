package keeper

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/AltheaFoundation/althea-L1/x/nativedex/types"
)

type (
	Keeper struct {
		storeKey   storetypes.StoreKey
		cdc        codec.BinaryCodec
		paramSpace paramtypes.Subspace

		EVMKeeper types.EVMKeeper
	}
)

func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	ps paramtypes.Subspace,

	ek types.EVMKeeper,

) Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		paramSpace: ps,
		EVMKeeper:  ek,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetParams will return the current Params
// Note that if this function is called before the chain has been initalized, a
// panic will occur. Use GetParamsIfSet instead e.g. in an AnteHandler which
// may run for creating genesis transactions
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

// GetParamsIfSet will return the current params, but will return an error if the
// chain is still initializing. By error checking this function is safe to use in
// handling genesis transactions and upgrade handlers.
func (k Keeper) GetParamsIfSet(ctx sdk.Context) (params types.Params, err error) {
	// Use a temporary params instance to get the param set pairs
	// We can't call methods on the zero-valued return params yet
	tempParams := types.Params{
		VerifiedNativeDexAddress:     "",
		VerifiedCrocPolicyAddress:    "",
		WhitelistedContractAddresses: []string{},
	}
	for _, pair := range tempParams.ParamSetPairs() {
		if !k.paramSpace.Has(ctx, pair.Key) {
			return types.Params{}, fmt.Errorf("the param key %s has not been set", string(pair.Key))
		}
	}

	// If all params are set, populate and return them
	k.paramSpace.GetParamSet(ctx, &params)
	return params, nil
}

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}

func (k Keeper) GetNativeDexAddress(ctx sdk.Context) common.Address {
	return common.HexToAddress(k.GetParams(ctx).VerifiedNativeDexAddress)
}

func (k Keeper) GetVerifiedCrocPolicyAddress(ctx sdk.Context) common.Address {
	return common.HexToAddress(k.GetParams(ctx).VerifiedCrocPolicyAddress)
}

func (k Keeper) GetWhitelistedContractAddresses(ctx sdk.Context) []string {
	return k.GetParams(ctx).WhitelistedContractAddresses
}
