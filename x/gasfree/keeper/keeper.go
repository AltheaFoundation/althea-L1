package keeper

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/AltheaFoundation/althea-L1/x/gasfree/types"
)

type Keeper struct {
	storeKey   storetypes.StoreKey
	paramSpace paramstypes.Subspace
	Cdc        codec.Codec
}

func NewKeeper(cdc codec.Codec, storeKey storetypes.StoreKey, paramSpace paramstypes.Subspace) Keeper {
	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	k := Keeper{
		paramSpace: paramSpace,
		storeKey:   storeKey,
		Cdc:        cdc,
	}

	return k
}

// GetParamsIfSet will return the current params, but will return an error if the
// chain is still initializing. By error checking this function is safe to use in
// handling genesis transactions.
func (k Keeper) GetParamsIfSet(ctx sdk.Context) (params types.Params, err error) {
	for _, pair := range params.ParamSetPairs() {
		if !k.paramSpace.Has(ctx, pair.Key) {
			return types.Params{}, errorsmod.Wrapf(sdkerrors.ErrNotFound, "the param key %s has not been set", string(pair.Key))
		}
		k.paramSpace.Get(ctx, pair.Key, pair.Value)
	}

	return
}

func (k Keeper) GetGasFreeMessageTypes(ctx sdk.Context) []string {
	gasFreeMessageTypes := types.DefaultParams().GasFreeMessageTypes
	k.paramSpace.GetIfExists(ctx, types.GasFreeMessageTypesKey, &gasFreeMessageTypes)
	return gasFreeMessageTypes
}

func (k Keeper) SetGasFreeMessageTypes(ctx sdk.Context, gasFreeMessageTypes []string) {
	k.paramSpace.Set(ctx, types.GasFreeMessageTypesKey, &gasFreeMessageTypes)
}

func (k Keeper) GetGasFreeMessageTypesSet(ctx sdk.Context) map[string]struct{} {
	return createSet(k.GetGasFreeMessageTypes(ctx))
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

// Checks if the given Tx contains only messages in the GasFreeMessageTypes set
func (k Keeper) IsGasFreeTx(ctx sdk.Context, keeper Keeper, tx sdk.Tx) (bool, error) {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return false, nil
	}

	gasFreeMessageSet := k.GetGasFreeMessageTypesSet(ctx)
	for _, msg := range msgs {
		switch msg := msg.(type) {
		// Since authz MsgExec holds Msgs inside of it, all those inner Msgs must be checked
		case *authz.MsgExec:
			for _, m := range msg.Msgs {
				var inner sdk.Msg
				err := k.Cdc.UnpackAny(m, &inner)
				if err != nil {
					return false,
						errorsmod.Wrapf(sdkerrors.ErrInvalidType, "unable to unpack authz msgexec message: %v", err)
				}
				// Check if the inner Msg is acceptable or not, returning an error kicks this whole Tx out of the mempool
				if !k.IsGasFreeMsg(gasFreeMessageSet, inner) {
					return false, nil
				}
			}
		// TODO: If ICA is used, those Msgs must be checked as well or they could be missed
		default:
			// Check if this Msg is acceptable or not, returning an error kicks this whole Tx out of the mempool
			if !k.IsGasFreeMsg(gasFreeMessageSet, msg) {
				return false, nil
			}
		}
	}

	// Discovered only gas-free messages
	return true, nil
}

// IsGasFreeMsg checks if the given msg is in the given gasFreeMessageTypes set, and if it shoudl be
func (k Keeper) IsGasFreeMsg(gasFreeMessageTypes map[string]struct{}, msg sdk.Msg) bool {
	return inSet(gasFreeMessageTypes, sdk.MsgTypeURL(msg))
}

// IsGasFreeMsgType checks if the given msgType is one of the GasFreeMessageTypes
func (k Keeper) IsGasFreeMsgType(ctx sdk.Context, msgType string) bool {
	msgTypes := k.GetGasFreeMessageTypes(ctx)
	for _, t := range msgTypes {
		if t == msgType {
			return true
		}
	}
	return false
}

func inSet(set map[string]struct{}, key string) bool {
	_, present := set[key]
	return present
}
