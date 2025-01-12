package microtx

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/AltheaFoundation/althea-L1/x/microtx/keeper"
	"github.com/AltheaFoundation/althea-L1/x/microtx/types"
)

// NewHandler returns a handler for "microtx" type messages.
func NewHandler(k keeper.Keeper) sdk.Handler {
	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		case *types.MsgMicrotx:
			res, err := msgServer.Microtx(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			return nil, errorsmod.Wrap(sdkerrors.ErrUnknownRequest, fmt.Sprintf("Unrecognized microtx Msg type: %v", sdk.MsgTypeURL(msg)))
		}
	}
}
