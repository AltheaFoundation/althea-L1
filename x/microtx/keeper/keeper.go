package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	erc20keeper "github.com/Canto-Network/Canto/v6/x/erc20/keeper"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"

	gasfreekeeper "github.com/AltheaFoundation/althea-L1/x/gasfree/keeper"
	"github.com/AltheaFoundation/althea-L1/x/microtx/types"
)

// Keeper maintains the link to storage and exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	// NOTE: If you add anything to this struct, add a nil check to ValidateMembers below!
	storeKey   storetypes.StoreKey // Unexposed key to access store from sdk.Context
	paramSpace paramtypes.Subspace

	// NOTE: If you add anything to this struct, add a nil check to ValidateMembers below!
	cdc           codec.BinaryCodec // The wire codec for binary encoding/decoding.
	bankKeeper    *bankkeeper.BaseKeeper
	accountKeeper *authkeeper.AccountKeeper
	evmKeeper     *evmkeeper.Keeper
	erc20Keeper   *erc20keeper.Keeper
	gasfreeKeeper *gasfreekeeper.Keeper
}

// Check for nil members
func (k Keeper) ValidateMembers() {
	if k.bankKeeper == nil {
		panic("Nil bankKeeper!")
	}
	if k.accountKeeper == nil {
		panic("Nil accountKeeper!")
	}
	if k.evmKeeper == nil {
		panic("Nil evmKeeper!")
	}
	if k.erc20Keeper == nil {
		panic("Nil erc20Keeper!")
	}
	if k.gasfreeKeeper == nil {
		panic("Nil gasfreeKeeper!")
	}
}

// NewKeeper returns a new instance of the microtx keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	paramSpace paramtypes.Subspace,
	cdc codec.BinaryCodec,
	bankKeeper *bankkeeper.BaseKeeper,
	accKeeper *authkeeper.AccountKeeper,
	evmKeeper *evmkeeper.Keeper,
	erc20Keeper *erc20keeper.Keeper,
	gasfreeKeeper *gasfreekeeper.Keeper,
) Keeper {
	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	k := Keeper{
		storeKey:   storeKey,
		paramSpace: paramSpace,

		cdc:           cdc,
		bankKeeper:    bankKeeper,
		accountKeeper: accKeeper,
		evmKeeper:     evmKeeper,
		erc20Keeper:   erc20Keeper,
		gasfreeKeeper: gasfreeKeeper,
	}

	k.ValidateMembers()

	return k
}

// GetParams will return the current Params
// Note that if this function is called before the chain has been initalized, a
// panic will occur. Use GetParamsIfSet instead e.g. in an AnteHandler which
// may run for creating genesis transactions
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return
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

// SetParams will store the given params after validating them
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "unable to store params with failing ValidateBasic()")
	}
	k.paramSpace.SetParamSet(ctx, &params)
	return nil
}

// GetMicrotxFeeBasisPoints will get the MicrotxFeeBasisPoints, if the params have been set
func (k Keeper) GetMicrotxFeeBasisPoints(ctx sdk.Context) (uint64, error) {
	params, err := k.GetParamsIfSet(ctx)
	if err != nil {
		// The params have been set, get the min send to eth fee
		return 0, err
	}
	return params.MicrotxFeeBasisPoints, nil
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
