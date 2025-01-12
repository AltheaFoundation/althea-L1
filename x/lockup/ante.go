// The lockup AnteHandler defines a configurable Tx validation system to control which Txs submitted to the chain will
// be allowed into the Mempool and eventually included in a block. In particular, this file defines a LockupAnteHandler,
// which performs the funds transfer locking feature and a WrappedAnteHanlder which makes it simpler to chain multiple
// AnteHandlers from disparate sources in the chain.
package lockup

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"

	evmtypes "github.com/evmos/ethermint/x/evm/types"

	microtxtypes "github.com/AltheaFoundation/althea-L1/x/microtx/types"

	"github.com/AltheaFoundation/althea-L1/x/lockup/keeper"
	"github.com/AltheaFoundation/althea-L1/x/lockup/types"
)

// WrappedAnteHandler An AnteDecorator used to wrap any AnteHandler for decorator chaining
// This is necessary to use the Cosmos SDK's NewAnteHandler() output with a LockupAnteHandler, as the sdk does not
// expose a stable list of AnteDecorators before chaining, and every upgrade poses a risk of missing updates
type WrappedAnteHandler struct {
	anteHandler sdk.AnteHandler
}

// AnteHandle calls wad.anteHandler and then the next one in the chain
func (wad WrappedAnteHandler) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx, simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	modCtx, err := wad.anteHandler(ctx, tx, simulate)
	if err != nil {
		return modCtx, err
	}
	return next(modCtx, tx, simulate)
}

// NewLockupAnteHandler returns an AnteHandler that ensures any transaction under a locked chain
// originates from a LockExempt address
func NewLockupAnteHandler(lockupKeeper keeper.Keeper, cdc codec.Codec) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(NewLockupAnteDecorator(lockupKeeper, cdc))
}

// NewLockupAnteDecorator initializes a LockupAnteDecorator for locking messages
// based on the settings stored in lockupKeeper
func NewLockupAnteDecorator(lockupKeeper keeper.Keeper, cdc codec.Codec) LockAnteDecorator {
	return LockAnteDecorator{lockupKeeper, cdc}
}

// WrappedLockupAnteHandler wraps a LockupAnteHandler around the input AnteHandler
func NewWrappedLockupAnteHandler(
	anteHandler sdk.AnteHandler,
	lockupKeeper keeper.Keeper,
	cdc codec.Codec,
) sdk.AnteHandler {
	wrapped := WrappedAnteHandler{anteHandler} // Must wrap to use in ChainAnteDecorators
	lad := NewLockupAnteDecorator(lockupKeeper, cdc)

	// Produces an AnteHandler which runs wrapped, then lad
	// Note: this is important as the default SetUpContextDecorator must be the
	// outermost one (see cosmos-sdk/x/auth/ante.NewAnteHandler())
	return sdk.ChainAnteDecorators(wrapped, lad)
}

// LockAnteDecorator Ensures that any transaction under a locked chain originates from a LockExempt address
type LockAnteDecorator struct {
	lockupKeeper keeper.Keeper
	cdc          codec.Codec
}

// AnteHandle Ensures that any transaction transferring a locked token via a locked message type from a nonexempt address
// is blocked when the chain is locked
func (lad LockAnteDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	if lad.lockupKeeper.GetChainLocked(ctx) {
		for _, msg := range tx.GetMsgs() {
			switch msg := msg.(type) {
			// Since authz MsgExec holds Msgs inside of it, all those inner Msgs must be checked
			case *authz.MsgExec:
				for _, m := range msg.Msgs {
					var inner sdk.Msg
					err := lad.cdc.UnpackAny(m, &inner)
					if err != nil {
						return ctx,
							errorsmod.Wrapf(sdkerrors.ErrInvalidType, "unable to unpack authz msgexec message: %v", err)
					}
					// Check if the inner Msg is acceptable or not, returning an error kicks this whole Tx out of the mempool
					if err := lad.isAcceptable(ctx, inner); err != nil {
						return ctx, err
					}
				}
			// TODO: If ICA is used, those Msgs must be checked as well or they could cause a loophole for us
			default:
				// Check if this Msg is acceptable or not, returning an error kicks this whole Tx out of the mempool
				if err := lad.isAcceptable(ctx, msg); err != nil {
					return ctx, err
				}
			}
		}
	}

	return next(ctx, tx, simulate)
}

// isAcceptable checks if the given msg is permissible under a locked chain, returns an error if not
// A Msg is not permissibile if it involves the transfer of a locked token via a locked Msg type from a nonexempt address
// All of these conditions are determined by what is stored in the lockup module params
func (lad LockAnteDecorator) isAcceptable(ctx sdk.Context, msg sdk.Msg) error {
	lockedTokenDenomsSet := lad.lockupKeeper.GetLockedTokenDenomsSet(ctx)
	lockedMsgTypesSet := lad.lockupKeeper.GetLockedMessageTypesSet(ctx)
	exemptSet := lad.lockupKeeper.GetLockExemptAddressesSet(ctx)

	msgType := sdk.MsgTypeURL(msg)
	if _, typePresent := lockedMsgTypesSet[msgType]; typePresent {
		// Check that any locked msg is permissible on a type-case basis
		if allow, err := allowMessage(msg, exemptSet, lockedTokenDenomsSet); !allow {
			return errorsmod.Wrap(err, "Transaction blocked because of a message")
		} else {
			// The user is exempt, allow it to pass
			return nil
		}
	}
	if msgType == "/cosmos.distribution.v1beta1.MsgSetWithdrawAddress" {
		return errorsmod.Wrap(types.ErrLocked, "The chain is locked, only exempt addresses may submit this Msg type")
	}
	if msgType == "/cosmos.authz.v1beta1.MsgExec" {
		return errorsmod.Wrap(types.ErrLocked, "The chain is locked, recursively MsgExec-wrapped Msgs are not allowed")
	}
	if msgType == "/ethermint.evm.v1.MsgEthereumTx" {
		if allow, err := allowMessage(msg, exemptSet, lockedTokenDenomsSet); !allow {
			return errorsmod.Wrap(err, "The chain is locked, only exempt addresses may submit this Msg type")
		} else {
			return nil
		}
	}

	return nil
}

// allowMessage checks that an input `msg` transferring a token in `lockedTokenDenomsSet` was sent by only addresses
// in `exemptSet`
// Returns (true, nil) if the message should be allowed to execute, (false, non-nil) if there was an issue
// NOTE: THIS MUST ONLY BE CALLED **AFTER** DETERMINING BOTH THE CHAIN IS LOCKED AND THE `msg` TYPE IS LOCKED,
// otherwise `msg` will be unnecessarily blocked
func allowMessage(msg sdk.Msg, exemptSet map[string]struct{}, lockedTokenDenomsSet map[string]struct{}) (bool, error) {
	switch sdk.MsgTypeURL(msg) {
	// ^v^v^v^v^v^v^v^v^v^v^v^v BANK MODULE MESSAGES ^v^v^v^v^v^v^v^v^v^v^v^v
	// nolint: exhaustruct
	case sdk.MsgTypeURL(&banktypes.MsgSend{}):
		msgSend := msg.(*banktypes.MsgSend)

		if _, present := exemptSet[msgSend.FromAddress]; !present {
			// Message sent from a non-exempt address while the chain is locked up, is the transfer a locked coin?
			for _, coin := range msgSend.Amount {
				if _, present := lockedTokenDenomsSet[coin.Denom]; present {
					return false, errorsmod.Wrap(types.ErrLocked,
						"The chain is locked, only exempt addresses may send locked denoms")
				}
			}
		}
		return true, nil
	// nolint: exhaustruct
	case sdk.MsgTypeURL(&banktypes.MsgMultiSend{}):
		msgMultiSend := msg.(*banktypes.MsgMultiSend)
		for _, input := range msgMultiSend.Inputs {
			lockedToken, blockedAddress := false, false

			for _, coin := range input.Coins {
				if _, present := lockedTokenDenomsSet[coin.Denom]; present {
					lockedToken = true
				}
			}
			if _, present := exemptSet[input.Address]; !present {
				// Multi-send Message sent with a non-exempt input address while the chain is locked up, is this a locked token?
				blockedAddress = true
			}

			if lockedToken && blockedAddress {
				return false, errorsmod.Wrap(types.ErrLocked,
					"The chain is locked, only exempt addresses may be inputs in a MultiSend message containing a locked token denom")
			}
		}
		return true, nil

	// ^v^v^v^v^v^v^v^v^v^v^v^v IBC TRANSFER MODULE MESSAGES ^v^v^v^v^v^v^v^v^v^v^v^v
	// nolint: exhaustruct
	case sdk.MsgTypeURL(&ibctransfertypes.MsgTransfer{}):
		msgTransfer := msg.(*ibctransfertypes.MsgTransfer)
		if _, present := exemptSet[msgTransfer.Sender]; !present {
			// The sender is not exempt, but are they sending a locked token?
			if _, present := lockedTokenDenomsSet[msgTransfer.Token.Denom]; present {
				// The token is locked, return an error
				return false, errorsmod.Wrap(types.ErrLocked,
					"The chain is locked, only exempt addresses may Transfer a locked token denom over IBC")
			}
		}
		return true, nil

	// ^v^v^v^v^v^v^v^v^v^v^v^v MICROTX MODULE MESSAGES ^v^v^v^v^v^v^v^v^v^v^v^v
	// nolint: exhaustruct
	case sdk.MsgTypeURL(&microtxtypes.MsgMicrotx{}):
		msgMicrotx := msg.(*microtxtypes.MsgMicrotx)
		if _, present := exemptSet[msgMicrotx.GetSender()]; !present {
			// The sender is not exempt, but are they sending a locked token?
			if _, present := lockedTokenDenomsSet[msgMicrotx.Amount.Denom]; present {
				// The token is locked, return an error
				return false, errorsmod.Wrap(types.ErrLocked,
					"The chain is locked, only exempt addresses may Microtx a locked token denom")
			}
		}
		return true, nil

	// ^v^v^v^v^v^v^v^v^v^v^v^v EVM MODULE MESSAGES ^v^v^v^v^v^v^v^v^v^v^v^v
	// nolint: exhaustruct
	case sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}):
		msgEvmTx := msg.(*evmtypes.MsgEthereumTx)
		addressBytes := common.HexToAddress(msgEvmTx.From).Bytes()
		ethermintAddr := sdk.AccAddress(addressBytes)
		if _, present := exemptSet[ethermintAddr.String()]; !present {
			return false, errorsmod.Wrap(types.ErrLocked,
				"The chain is locked, only exempt addresses may send a MsgEthereumTx")
		}
		return true, nil

	default:
		return false, errorsmod.Wrap(types.ErrUnhandled,
			fmt.Sprintf("Message type %v does not have a case in allowMessage, unable to handle messages like this",
				sdk.MsgTypeURL(msg),
			),
		)
	}
}
