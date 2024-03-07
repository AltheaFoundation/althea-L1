package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const RootCodespace = "gasfree"

var (
	ErrInvalidParams = sdkerrors.Register(RootCodespace, 1, "invalid params")
)
