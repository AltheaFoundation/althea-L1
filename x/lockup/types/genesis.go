package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
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
	_, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid lock exempt type: %T", i)
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

// ParamKeyTable for auth module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{
		Locked:             false,
		LockExempt:         []string{},
		LockedMessageTypes: []string{},
	})
}

func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(LockedKey, &p.Locked, ValidateLocked),
		paramtypes.NewParamSetPair(LockExemptKey, &p.LockExempt, ValidateLockExempt),
		paramtypes.NewParamSetPair(LockedMessageTypesKey, &p.LockedMessageTypes, ValidateLockedMessageTypes),
	}
}
