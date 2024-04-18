package gasfree

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/AltheaFoundation/althea-L1/x/gasfree/keeper"
)

// NewSelectiveBypassDecorator returns an AnteDecorator which will not execute the
// bypassable decorator for any Txs which **only** contain messages
// of types in the GasFreeMessageTypes set
// This decorator is meant to avoid an exempt Tx from being kicked out of the mempool,
// instead allowing alternative fees collection in the message handler or in secondary AnteHandlers
func NewSelectiveBypassDecorator(gasfreeKeeper keeper.Keeper, bypassable sdk.AnteDecorator) SelectiveBypassDecorator {
	return SelectiveBypassDecorator{gasfreeKeeper, bypassable}
}

// SelectiveBypassDecorator enables AnteHandler bypassing for Txs containing only GasFreeMessageTypes
type SelectiveBypassDecorator struct {
	gasfreeKeeper keeper.Keeper
	bypassable    sdk.AnteDecorator
}

// AnteHandle first checks to see if the tx contains **only** messages in the GasFreeMessageTypes set, if so it will
// skip calling the bypassable AnteDecorator. Otherwise, the bypassable AnteDecorator will be called as normal.
func (sbd SelectiveBypassDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	gasFree, err := sbd.gasfreeKeeper.IsGasFreeTx(ctx, sbd.gasfreeKeeper, tx)
	if err != nil {
		return ctx, sdkerrors.Wrap(err, "failed to check AnteDecorator can be bypassed")
	}

	if !gasFree {
		return sbd.bypassable.AnteHandle(ctx, tx, simulate, next)
	} else {
		return next(ctx, tx, simulate)
	}
}

/* TODO: Handle ICA messages (not mission critical, they will just not be supported by the gasfree module if not considered)

var data icatypes.InterchainAccountPacketData

if err := icatypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
	// UnmarshalJSON errors are indeterminate and therefore are not wrapped and included in failed acks
	return nil, sdkerrors.Wrapf(icatypes.ErrUnknownDataType, "cannot unmarshal ICS-27 interchain account packet data")
}

switch data.Type {
case icatypes.EXECUTE_TX:
	msgs, err := icatypes.DeserializeCosmosTx(k.cdc, data.Data)
	if err != nil {
		return nil, err
	}

	txResponse, err := k.executeTx(ctx, packet.SourcePort, packet.DestinationPort, packet.DestinationChannel, msgs)
	if err != nil {
		return nil, err
	}

	return txResponse, nil
default:
	return nil, icatypes.ErrUnknownDataType
}
*/
