package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	gasfreekeeper "github.com/althea-net/althea-L1/x/gasfree/keeper"
	microtxkeeper "github.com/althea-net/althea-L1/x/microtx/keeper"
	microtxtypes "github.com/althea-net/althea-L1/x/microtx/types"
)

var microtxMsgType string = sdk.MsgTypeURL(&microtxtypes.MsgMicrotx{})

// ChargeGasfreeFeesDecorator enables custom fee charging for gas-free transactions on a per-message basis
type ChargeGasfreeFeesDecorator struct {
	ak            AccountKeeper
	gasfreeKeeper gasfreekeeper.Keeper
	microtxKeeper microtxkeeper.Keeper
}

func NewChargeGasfreeFeesDecorator(ak AccountKeeper, gasfreeKeeper gasfreekeeper.Keeper, microtxKeeper microtxkeeper.Keeper) ChargeGasfreeFeesDecorator {
	return ChargeGasfreeFeesDecorator{
		ak:            ak,
		gasfreeKeeper: gasfreeKeeper,
		microtxKeeper: microtxKeeper,
	}
}

// AnteHandle charges fees for gas-free transactions on a case-by-case basis
func (satd ChargeGasfreeFeesDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Handle any microtxs individually
	err := satd.DeductAnyMicrotxFees(ctx, tx)
	if err != nil {
		return ctx, sdkerrors.Wrap(err, "failed to deduct microtx fees")
	}

	return next(ctx, tx, simulate)
}

func (satd ChargeGasfreeFeesDecorator) DeductAnyMicrotxFees(ctx sdk.Context, tx sdk.Tx) error {
	// Only deduct Microtx fees in the AnteHandler if they are currently configured as gasfree messages
	if !satd.gasfreeKeeper.IsGasFreeMsgType(ctx, microtxMsgType) {
		return nil
	}

	for _, msg := range tx.GetMsgs() {
		msgMicrotx, isMicrotx := msg.(*microtxtypes.MsgMicrotx)
		if isMicrotx {
			feeCollected, err := satd.microtxKeeper.DeductMsgMicrotxFee(ctx, msgMicrotx)
			if err != nil {
				return sdkerrors.Wrap(err, "unable to collect microtx fee prior to msg execution")
			}
			ctx.EventManager().EmitEvent(microtxtypes.NewEventMicrotxFeeCollected(msgMicrotx.Sender, *feeCollected))
		}
	}

	return nil
}
