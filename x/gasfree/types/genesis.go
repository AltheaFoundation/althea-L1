package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	microtxtypes "github.com/althea-net/althea-L1/x/microtx/types"
)

// DefaultGenesisState creates a simple GenesisState suitible for testing
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

func DefaultParams() *Params {
	return &Params{
		GasFreeMessageTypes: []string{
			// nolint: exhaustruct
			sdk.MsgTypeURL(&microtxtypes.MsgMicrotx{}),
		},
	}
}

func (s GenesisState) ValidateBasic() error {
	if s.Params == nil {
		return ErrInvalidParams
	}
	if err := ValidateGasFreeMessageTypes(s.Params.GasFreeMessageTypes); err != nil {
		return sdkerrors.Wrap(err, "Invalid GasFreeMessageTypes GenesisState")
	}
	return nil
}

func ValidateGasFreeMessageTypes(i interface{}) error {
	_, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid gas free message types type: %T", i)
	}

	return nil
}

// ParamKeyTable for auth module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{
		GasFreeMessageTypes: []string{},
	})
}

func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(GasFreeMessageTypesKey, &p.GasFreeMessageTypes, ValidateGasFreeMessageTypes),
	}
}
