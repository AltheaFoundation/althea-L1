package types

import (
	sdkerrors "cosmossdk.io/errors"
)

const RootCodespace = "gasfree"

var (
	ErrInvalidParams = sdkerrors.Register(RootCodespace, 1, "invalid params")
)
