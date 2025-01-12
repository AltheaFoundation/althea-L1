package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrContractDeployment   = errorsmod.Register(ModuleName, 1, "contract deploy failed")
	ErrContractCall         = errorsmod.Register(ModuleName, 2, "contract call failed")
	ErrNoLiquidAccount      = errorsmod.Register(ModuleName, 3, "account is not a liquid infrastructure account")
	ErrInvalidThresholds    = errorsmod.Register(ModuleName, 4, "invalid liquid infrastructure account thresholds")
	ErrInvalidMicrotx       = errorsmod.Register(ModuleName, 5, "invalid microtx")
	ErrInvalidContract      = errorsmod.Register(ModuleName, 6, "invalid contract")
	ErrAccountAlreadyLiquid = errorsmod.Register(ModuleName, 7, "account is already a liquid infrastructure account")
)
