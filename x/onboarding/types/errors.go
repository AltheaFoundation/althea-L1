package types

import (
	sdkerrors "cosmossdk.io/errors"
)

// errors
var (
	ErrBlockedAddress = sdkerrors.Register(ModuleName, 2, "blocked address")
	ErrInvalidType    = sdkerrors.Register(ModuleName, 3, "invalid type")
)
