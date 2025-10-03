package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/AltheaFoundation/althea-L1/x/erc20/types"
)

// Keeper of this module maintains collections of erc20.
type Keeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	paramstore paramtypes.Subspace

	accountKeeper     types.AccountKeeper
	bankKeeper        types.BankKeeper
	evmKeeper         types.EVMKeeper
	gasfreeKeeper     types.GasfreeKeeper
	ibcTransferKeeper types.IBCTransferKeeper
}

// NewKeeper creates new instances of the erc20 Keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	ps paramtypes.Subspace,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	evmKeeper types.EVMKeeper,
	gasfreeKeeper types.GasfreeKeeper,
) Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	if ak == nil {
		panic("account keeper is nil")
	}
	if bk == nil {
		panic("bank keeper is nil")
	}
	if evmKeeper == nil {
		panic("evm keeper is nil")
	}
	if gasfreeKeeper == nil {
		panic("gasfree keeper is nil")
	}
	return Keeper{
		storeKey:      storeKey,
		cdc:           cdc,
		paramstore:    ps,
		accountKeeper: ak,
		bankKeeper:    bk,
		evmKeeper:     evmKeeper,
		gasfreeKeeper: gasfreeKeeper,
	}
}

// SetIBCTransferKeeper injects the IBC transfer keeper after both erc20 and the dependent modules are constructed.
// It panics if called more than once or with a nil argument.
func (k *Keeper) SetIBCTransferKeeper(ibc types.IBCTransferKeeper) {
	if ibc == nil {
		panic("attempted to set a nil ibcTransferKeeper on erc20 keeper")
	}
	if k.ibcTransferKeeper != nil {
		panic("ibcTransferKeeper already set on erc20 keeper")
	}
	k.ibcTransferKeeper = ibc
}

// ValidateDependencies ensures all late-bound dependencies have been set; call at end of app constructor.
func (k Keeper) ValidateDependencies() {
	if k.ibcTransferKeeper == nil {
		panic("erc20 keeper dependency not set: ibcTransferKeeper")
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
