package keeper

import (
	"github.com/AltheaFoundation/althea-L1/x/nativedex/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// nolint: exhaustruct
var _ types.MsgServer = msgServer{}
