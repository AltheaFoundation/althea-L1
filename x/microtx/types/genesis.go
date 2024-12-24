package types

import (
	"bytes"
	"fmt"

	"sigs.k8s.io/yaml"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
	ParamStoreKeyBaseProposerReward     = "BaseProposerReward"
	ParamStoreKeyBonusProposerReward    = "BonusProposerReward"
)

// ValidateBasic validates genesis state by looping through the params and
// calling their validation functions
func (s GenesisState) ValidateBasic() error {
	if err := s.Params.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "params")
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
		BaseProposerReward:    sdk.NewDecWithPrec(1, 2), // 1%
		BonusProposerReward:   sdk.NewDecWithPrec(4, 2), // 4%
	}
}

// ValidateBasic checks that the parameters have valid values.
func (p Params) ValidateBasic() error {
	if err := validateMicrotxFeeBasisPoints(p.MicrotxFeeBasisPoints); err != nil {
		return sdkerrors.Wrap(err, "MicrotxFeeBasisPoints")
	}
	if p.BaseProposerReward.IsNegative() {
		return fmt.Errorf(
			"base proposer reward should be positive: %s", p.BaseProposerReward,
		)
	}
	if p.BonusProposerReward.IsNegative() {
		return fmt.Errorf(
			"bonus proposer reward should be positive: %s", p.BonusProposerReward,
		)
	}
	if v := p.BaseProposerReward.Add(p.BonusProposerReward); v.GT(sdk.OneDec()) {
		return fmt.Errorf(
			"sum of base and bonus proposer rewards cannot be greater than one: %s", v,
		)
	}

	return nil
}

func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(DefaultParams())
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of auth module's parameters.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair([]byte(ParamsStoreKeyMicrotxFeeBasisPoints), &p.MicrotxFeeBasisPoints, validateMicrotxFeeBasisPoints),
		paramtypes.NewParamSetPair([]byte(ParamStoreKeyBaseProposerReward), &p.BaseProposerReward, validateBaseProposerReward),
		paramtypes.NewParamSetPair([]byte(ParamStoreKeyBonusProposerReward), &p.BonusProposerReward, validateBonusProposerReward),
	}
}

// Equal returns a boolean determining if two Params types are identical.
func (p Params) Equal(p2 Params) bool {
	bz1 := ModuleCdc.MustMarshalLengthPrefixed(&p)
	bz2 := ModuleCdc.MustMarshalLengthPrefixed(&p2)
	return bytes.Equal(bz1, bz2)
}

func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
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

func validateBaseProposerReward(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("base proposer reward must be not nil")
	}
	if v.IsNegative() {
		return fmt.Errorf("base proposer reward must be positive: %s", v)
	}
	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("base proposer reward too large: %s", v)
	}

	return nil
}

func validateBonusProposerReward(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("bonus proposer reward must be not nil")
	}
	if v.IsNegative() {
		return fmt.Errorf("bonus proposer reward must be positive: %s", v)
	}
	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("bonus proposer reward too large: %s", v)
	}

	return nil
}
