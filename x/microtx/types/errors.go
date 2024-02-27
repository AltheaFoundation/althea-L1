package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	ErrContractDeployment   = sdkerrors.Register(ModuleName, 1, "contract deploy failed")
	ErrContractCall         = sdkerrors.Register(ModuleName, 2, "contract call failed")
	ErrNoLiquidAccount      = sdkerrors.Register(ModuleName, 3, "account is not a liquid infrastructure account")
	ErrInvalidThresholds    = sdkerrors.Register(ModuleName, 4, "invalid liquid infrastructure account thresholds")
	ErrInvalidMicrotx       = sdkerrors.Register(ModuleName, 5, "invalid microtx")
	ErrInvalidContract      = sdkerrors.Register(ModuleName, 6, "invalid contract")
	ErrAccountAlreadyLiquid = sdkerrors.Register(ModuleName, 7, "account is already a liquid infrastructure account")
)
