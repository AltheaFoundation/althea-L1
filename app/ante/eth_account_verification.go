package ante

import (
	"github.com/ethereum/go-ethereum/common"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	ethtypes "github.com/evmos/ethermint/types"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/statedb"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

// This file contains contents largely copied from the upstream ethermint repository,
// the only modification is to explicitly set the account type to EthAccount rather than using the default BaseAccount

// EthAccountVerificationDecorator validates an account balance checks
type EthAccountVerificationDecorator struct {
	ak        AccountKeeper
	evmKeeper *evmkeeper.Keeper
}

// NewEthAccountVerificationDecorator creates a new EthAccountVerificationDecorator
func NewEthAccountVerificationDecorator(ak AccountKeeper, ek *evmkeeper.Keeper) EthAccountVerificationDecorator {
	if ek == nil {
		panic("evm keeper is required for EthAccountVerificationDecorator")
	}

	return EthAccountVerificationDecorator{
		ak:        ak,
		evmKeeper: ek,
	}
}

// AnteHandle validates checks that the sender balance is greater than the total transaction cost.
// The account will be set to store if it doesn't exis, i.e cannot be found on store.
// This AnteHandler decorator will fail if:
// - any of the msgs is not a MsgEthereumTx
// - from address is empty
// - account balance is lower than the transaction cost
func (avd EthAccountVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !ctx.IsCheckTx() {
		return next(ctx, tx, simulate)
	}

	for i, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return ctx, errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}

		txData, err := evmtypes.UnpackTxData(msgEthTx.Data)
		if err != nil {
			return ctx, errorsmod.Wrapf(err, "failed to unpack tx data any for tx %d", i)
		}

		// sender address should be in the tx cache from the previous AnteHandle call
		from := msgEthTx.GetFrom()
		if from.Empty() {
			return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "from address cannot be empty")
		}

		// check whether the sender address is EOA
		fromAddr := common.BytesToAddress(from)
		acct := avd.evmKeeper.GetAccount(ctx, fromAddr)

		if acct == nil {
			acc := ethtypes.ProtoAccount()
			err := acc.SetAddress(from)
			if err != nil {
				return ctx, errorsmod.Wrap(err, "failed to set address")
			}
			avd.ak.NewAccount(ctx, acc)
			acct = statedb.NewEmptyAccount()
		} else if acct.IsContract() {
			return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidType,
				"the sender is not EOA: address %s, codeHash <%s>", fromAddr, acct.CodeHash)
		}

		if err := evmkeeper.CheckSenderBalance(sdk.NewIntFromBigInt(acct.Balance), txData); err != nil {
			return ctx, errorsmod.Wrap(err, "failed to check sender balance")
		}

	}
	return next(ctx, tx, simulate)
}
