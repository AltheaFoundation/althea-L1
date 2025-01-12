package types

import (
	"bytes"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// DefaultParamspace defines the default auth module parameter subspace
const (
	// todo: implement oracle constants as params
	DefaultParamspace = ModuleName
)

var (
	// Ensure that params implements the proper interface
	// nolint: exhaustruct
	_ paramtypes.ParamSet = &Params{}

	ParamsStoreKeyMicrotxFeeBasisPoints = "MicrotxFeeBasisPoints"
)

// ValidateBasic validates genesis state by looping through the params and
// calling their validation functions
func (s GenesisState) ValidateBasic() error {
	if err := s.Params.ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "params")
	}
	return nil
}

// DefaultGenesisState returns empty genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// DefaultParams returns a copy of the default params
func DefaultParams() *Params {
	return &Params{
		MicrotxFeeBasisPoints: 1000,
	}
}

// ValidateBasic checks that the parameters have valid values.
func (p Params) ValidateBasic() error {
	if err := validateMicrotxFeeBasisPoints(p.MicrotxFeeBasisPoints); err != nil {
		return errorsmod.Wrap(err, "MicrotxFeeBasisPoints")
	}
	return nil
}

// ParamKeyTable for auth module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{
		MicrotxFeeBasisPoints: 1000,
	})
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of auth module's parameters.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair([]byte(ParamsStoreKeyMicrotxFeeBasisPoints), &p.MicrotxFeeBasisPoints, validateMicrotxFeeBasisPoints),
	}
}

// Equal returns a boolean determining if two Params types are identical.
func (p Params) Equal(p2 Params) bool {
	bz1 := ModuleCdc.MustMarshalLengthPrefixed(&p)
	bz2 := ModuleCdc.MustMarshalLengthPrefixed(&p2)
	return bytes.Equal(bz1, bz2)
}

func validateMicrotxFeeBasisPoints(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v >= 10000 {
		return fmt.Errorf("excessive microtx fee of at least 100 percent")
	}
	return nil
}
