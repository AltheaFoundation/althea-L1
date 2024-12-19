package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"

	evmtypes "github.com/evmos/ethermint/x/evm/types"

	"github.com/AltheaFoundation/althea-L1/config"
	microtxtypes "github.com/AltheaFoundation/althea-L1/x/microtx/types"
)

// DefaultGenesisState creates a simple GenesisState suitible for testing
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

func DefaultParams() *Params {
	return &Params{
		Locked:     false,
		LockExempt: []string{},
		LockedMessageTypes: []string{
			// nolint: exhaustruct
			sdk.MsgTypeURL(&banktypes.MsgSend{}),
			// nolint: exhaustruct
			sdk.MsgTypeURL(&banktypes.MsgMultiSend{}),
			// nolint: exhaustruct
			sdk.MsgTypeURL(&ibctransfertypes.MsgTransfer{}),
			// nolint: exhaustruct
			sdk.MsgTypeURL(&microtxtypes.MsgMicrotx{}),
			// nolint: exhaustruct
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
		},
		/* Note: The authoritative way to get the native token of the chain is by calling
		   mintKeeper.GetParams(ctx).MintDenom, but the context is not available yet
		   Therefore it's critical to know that this denom is correct in the
		   first BeginBlock. If you change any of these values, change app/config
		   where necessary */
		LockedTokenDenoms: []string{
			config.BaseDenom,
		},
	}
}

func (s GenesisState) ValidateBasic() error {
	if err := ValidateLockExempt(s.Params.LockExempt); err != nil {
		return sdkerrors.Wrap(err, "Invalid LockExempt GenesisState")
	}
	if err := ValidateLockedMessageTypes(s.Params.LockedMessageTypes); err != nil {
		return sdkerrors.Wrap(err, "Invalid LockedMessageTypes GenesisState")
	}
	if err := ValidateLockedTokenDenoms(s.Params.LockedTokenDenoms); err != nil {
		return sdkerrors.Wrap(err, "Invalid LockedTokenDenoms GenesisState")
	}
	return nil
}

func ValidateLocked(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid locked type: %T", i)
	}
	return nil
}

func ValidateLockExempt(i interface{}) error {
	v, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid lock exempt type: %T", i)
	}
	for i, address := range v {
		if _, err := sdk.AccAddressFromBech32(address); err != nil {
			return fmt.Errorf("invalid lock exempt address %d: %s", i, address)
		}
	}

	return nil
}

func ValidateLockedMessageTypes(i interface{}) error {
	v, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid locked message types type: %T", i)
	}
	if len(v) == 0 {
		return fmt.Errorf("no locked message types %v", v)
	}
	return nil
}

func ValidateLockedTokenDenoms(i interface{}) error {
	v, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid locked token denoms type: %T", i)
	}
	if len(v) == 0 {
		return fmt.Errorf("no locked token denoms %v", v)
	}
	return nil
}

// ParamKeyTable for auth module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{
		Locked:             false,
		LockExempt:         []string{},
		LockedMessageTypes: []string{},
		LockedTokenDenoms:  []string{},
	})
}

func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(LockedKey, &p.Locked, ValidateLocked),
		paramtypes.NewParamSetPair(LockExemptKey, &p.LockExempt, ValidateLockExempt),
		paramtypes.NewParamSetPair(LockedMessageTypesKey, &p.LockedMessageTypes, ValidateLockedMessageTypes),
		paramtypes.NewParamSetPair(LockedTokenDenomsKey, &p.LockedTokenDenoms, ValidateLockedTokenDenoms),
	}
}
