package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	ErrContractDeployment = sdkerrors.Register(ModuleName, 1, "contract deploy failed")
	ErrContractCall       = sdkerrors.Register(ModuleName, 2, "contract call failed")
	ErrNoTokenizedAccount = sdkerrors.Register(ModuleName, 3, "account is not tokenized")
	ErrInvalidThresholds  = sdkerrors.Register(ModuleName, 4, "invalid tokenized account thresholds")
	ErrInvalidMicrotx     = sdkerrors.Register(ModuleName, 5, "invalid microtx")
)
