package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/nativedex module errors
var (
	ErrInvalidEvmAddress = sdkerrors.Register(ModuleName, 2, "Invalid EVM Address")
	ErrInvalidCallpath   = sdkerrors.Register(ModuleName, 3, "Invalid Callpath")
)
