package types

import (
	"bytes"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/common"
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

	ParamsStoreKeyVerifiedNativeDexAddress  = "VerifiedNativeDexAddress"
	ParamsStoreKeyVerifiedCrocPolicyAddress = "VerifiedCrocPolicyAddress"
)

// ValidateBasic validates genesis state by looping through the params and
// calling their validation functions
func (s GenesisState) ValidateBasic() error {
	if err := s.Params.ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "params")
	}
	return nil
}

// DefaultGenesis returns empty genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: *DefaultParams(),
	}
}

// DefaultParams returns a copy of the default params
func DefaultParams() *Params {
	return &Params{
		VerifiedNativeDexAddress:  common.BytesToAddress([]byte{0x0}).String(),
		VerifiedCrocPolicyAddress: common.BytesToAddress([]byte{0x0}).String(),
	}
}

// ValidateBasic checks that the parameters have valid values.
func (p Params) ValidateBasic() error {
	if err := validateVerifiedNativeDexAddress(p.VerifiedNativeDexAddress); err != nil {
		return errorsmod.Wrap(err, "VerifiedNativeDexAddress")
	}
	if err := validateVerifiedCrocPolicyAddress(p.VerifiedCrocPolicyAddress); err != nil {
		return errorsmod.Wrap(err, "VerifiedCrocPolicyAddress")
	}
	return nil
}

// ParamKeyTable for auth module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(DefaultParams())
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of auth module's parameters.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair([]byte(ParamsStoreKeyVerifiedNativeDexAddress), &p.VerifiedNativeDexAddress, validateVerifiedNativeDexAddress),
		paramtypes.NewParamSetPair([]byte(ParamsStoreKeyVerifiedCrocPolicyAddress), &p.VerifiedCrocPolicyAddress, validateVerifiedCrocPolicyAddress),
	}
}

// Equal returns a boolean determining if two Params types are identical.
func (p Params) Equal(p2 Params) bool {
	bz1 := ModuleCdc.MustMarshalLengthPrefixed(&p)
	bz2 := ModuleCdc.MustMarshalLengthPrefixed(&p2)
	return bytes.Equal(bz1, bz2)
}

func validateVerifiedNativeDexAddress(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if !common.IsHexAddress(v) {
		return errorsmod.Wrap(ErrInvalidEvmAddress, "invalid address")
	}

	return nil
}

func validateVerifiedCrocPolicyAddress(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if !common.IsHexAddress(v) {
		return errorsmod.Wrap(ErrInvalidEvmAddress, "invalid address")
	}

	return nil
}
