package ante

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	grouptypes "github.com/cosmos/cosmos-sdk/x/group"
)

// maxNestedMsgs defines a cap for the number of nested messages on a MsgExec message
const maxNestedMsgs = 6

// GroupLimiterDecorator blocks certain msg types from being granted or executed
// within the authorization module.
type GroupLimiterDecorator struct {
	// disabledMsgs is a set that contains type urls of unauthorized msgs.
	disabledMsgs map[string]struct{}
}

// NewGroupLimiterDecorator creates a decorator to block certain msg types
// from being granted or executed within authz.
func NewGroupLimiterDecorator(disabledMsgTypes []string) GroupLimiterDecorator {
	disabledMsgs := make(map[string]struct{})
	for _, url := range disabledMsgTypes {
		disabledMsgs[url] = struct{}{}
	}

	return GroupLimiterDecorator{
		disabledMsgs: disabledMsgs,
	}
}

func (gld GroupLimiterDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if err := gld.checkDisabledMsgs(tx.GetMsgs(), false, 0); err != nil {
		return ctx, errorsmod.Wrapf(errortypes.ErrUnauthorized, err.Error())
	}
	return next(ctx, tx, simulate)
}

// checkDisabledMsgs iterates through the msgs and returns an error if it finds any unauthorized msgs.
//
// This method is recursive as MsgExec's can wrap other MsgExecs. nestedMsgs sets a reasonable limit on
// the total messages, regardless of how they are nested.
func (gld GroupLimiterDecorator) checkDisabledMsgs(msgs []sdk.Msg, isGroupInnerMsg bool, nestedMsgs int) error {
	if nestedMsgs >= maxNestedMsgs {
		return fmt.Errorf("found more nested msgs than permitted. Limit is : %d", maxNestedMsgs)
	}
	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *grouptypes.MsgSubmitProposal:
			if isGroupInnerMsg {
				return fmt.Errorf("found MsgSubmitProposal inside another group msg")
			}
			innerMsgs, err := msg.GetMsgs()
			if err != nil {
				return err
			}
			nestedMsgs++
			if err := gld.checkDisabledMsgs(innerMsgs, true, nestedMsgs); err != nil {
				return err
			}
		case *authz.MsgExec:
			if isGroupInnerMsg {
				return fmt.Errorf("found MsgExec inside a group msg")
			}
		default:
			url := sdk.MsgTypeURL(msg)
			if isGroupInnerMsg && gld.isDisabledMsg(url) {
				return fmt.Errorf("found disabled msg type: %s", url)
			}
		}
	}
	return nil
}

// isDisabledMsg returns true if the given message is in the set of restricted
// messages from the AnteHandler.
func (gld GroupLimiterDecorator) isDisabledMsg(msgTypeURL string) bool {
	_, ok := gld.disabledMsgs[msgTypeURL]
	return ok
}
