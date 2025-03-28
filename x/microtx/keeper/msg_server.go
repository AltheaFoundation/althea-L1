package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/ethereum/go-ethereum/common"

	"github.com/AltheaFoundation/althea-L1/config"
	"github.com/AltheaFoundation/althea-L1/x/microtx/types"
)

// BasisPointDivisor used in calculating the MsgMicrotx fee amount to deduct
const BasisPointDivisor uint64 = 10000

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the gov MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// ========================================================================================================
// 												MICROTX
// ========================================================================================================

// Microtx delegates the msg server's call to the keeper
func (m msgServer) Microtx(c context.Context, msg *types.MsgMicrotx) (*types.MsgMicrotxResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	// The following validation logic has been copied from x/bank in the sdk
	if err := m.bankKeeper.IsSendEnabledCoins(ctx, msg.Amount); err != nil {
		return nil, err
	}

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	receiver, err := sdk.AccAddressFromBech32(msg.Receiver)
	if err != nil {
		return nil, err
	}

	if m.bankKeeper.BlockedAddr(receiver) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", msg.Receiver)
	}

	// Call the actual transfer implementation
	if err := m.Keeper.Microtx(ctx, sender, receiver, msg.Amount); err != nil {
		return nil, errorsmod.Wrap(err, "unable to complete the transfer")
	}

	return &types.MsgMicrotxResponse{}, err
}

// Microtx implements the transfer of funds from sender to receiver
// Due to the function of Liquid Infrastructure Accounts, any Microtx must transfer only EVM compatible bank coins
func (k Keeper) Microtx(ctx sdk.Context, sender sdk.AccAddress, receiver sdk.AccAddress, amount sdk.Coin) error {
	erc20Address, err := k.ValidateAndGetERC20Address(ctx, amount)
	if err != nil {
		return err
	}

	// If MsgMicrotx is not a gas free msg, then the fees should be charged here since they were not charged in the antehandler
	// nolint: exhaustruct
	if !k.gasfreeKeeper.IsGasFreeMsgType(ctx, sdk.MsgTypeURL(&types.MsgMicrotx{})) {
		collected, err := k.DeductMicrotxFee(ctx, sender, amount)
		if err != nil {
			return errorsmod.Wrap(err, "unable to collect MsgMicrotx fees")
		}
		ctx.EventManager().EmitEvent(types.NewEventMicrotxFeeCollected(sender.String(), *collected))
	}

	// Perform the transfer now that fees have been collected
	err = k.bankKeeper.SendCoins(ctx, sender, receiver, sdk.NewCoins(amount))
	if err != nil {
		return errorsmod.Wrap(err, "unable to send tokens via the bank module")
	}

	// Emit an event for the block's event log
	ctx.EventManager().EmitEvent(
		types.NewEventMicrotx(sender.String(), receiver.String(), amount),
	)

	// Detect if this is a Liquid Infrastructure Account and if so
	// migrate balances to the NFT if the amounts are in excess of any configured threshold
	k.Logger(ctx).Debug("Detecting and funneling excess balances for liquid infrastructure accounts")
	if err := k.RedirectLiquidAccountExcessBalance(ctx, receiver, erc20Address); err != nil {
		return errorsmod.Wrapf(err, "failed to redirect excess balance")
	}

	return nil
}

func (k Keeper) ValidateAndGetERC20Address(ctx sdk.Context, amount sdk.Coin) (common.Address, error) {
	var erc20Address common.Address
	// The native token is automatically usable within the EVM
	if amount.Denom != config.BaseDenom {
		// Ensure the input tokens are actively registered as an ERC20-convertible token
		pair, found := k.erc20Keeper.GetTokenPair(ctx, k.erc20Keeper.GetTokenPairID(ctx, amount.Denom))
		if !found {
			return common.Address{}, errorsmod.Wrapf(types.ErrInvalidMicrotx, "token %v is not registered as an erc20, only evm-compatible tokens may be used", amount.Denom)
		}
		if !pair.Enabled {
			return common.Address{}, errorsmod.Wrapf(types.ErrInvalidMicrotx, "token %v is registered as an erc20 (%v), but the pair is not enabled", amount.Denom, pair.Erc20Address)
		}
		// Collect the ERC20 address for later use in funneling
		erc20Address = common.HexToAddress(pair.Erc20Address)
	}

	return erc20Address, nil
}

// DeductMsgMicrotxFee is expected to be called from the AnteHandler to deduct the fee for the Msg
// It is possible for MsgMicrotx to not be a gasfree message type, since governance controls the list,
// in that case the fee should be deducted in the Msg handler
//
// WARNING: Do **NOT** call this from the MsgMicrotx handler, as it will result in bad event logs, call DeductMicrotxFee instead
func (k Keeper) DeductMsgMicrotxFee(ctx sdk.Context, msg *types.MsgMicrotx) (feeCollected *sdk.Coin, err error) {
	_, err = k.ValidateAndGetERC20Address(ctx, msg.Amount)
	if err != nil {
		return nil, errorsmod.Wrap(err, "unable to deduct Microtx fees")
	}

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	feeCollected, err = k.DeductMicrotxFee(ctx, sender, msg.Amount)

	if err != nil {
		return nil, errorsmod.Wrap(err, "unable to collect fees")
	}

	return
}

// DeductMicrotxFee will check and deduct the MsgMicrotx fee for the given sendAmount, based on the MicrotxFeeBasisPoints param value
func (k Keeper) DeductMicrotxFee(ctx sdk.Context, sender sdk.AccAddress, sendAmount sdk.Coin) (feeCollected *sdk.Coin, err error) {
	// Compute the minimum fees which must be paid
	microtxFeeBasisPoints, err := k.GetMicrotxFeeBasisPoints(ctx)
	if err != nil {
		microtxFeeBasisPoints = 0
	}
	microtxFee := k.getMicrotxFeeForAmount(sendAmount.Amount, microtxFeeBasisPoints)
	microtxFeeCoin := sdk.NewCoin(sendAmount.Denom, microtxFee)

	// Require that the minimum has been met
	if !microtxFee.IsZero() { // Ignore fees too low to collect
		balance := k.bankKeeper.GetBalance(ctx, sender, microtxFeeCoin.Denom)
		if balance.IsLT(microtxFeeCoin) {
			err := errorsmod.Wrapf(
				sdkerrors.ErrInsufficientFee,
				"balance is insufficient to pay the fee (%v < %v)",
				balance.Amount,
				microtxFee,
			)
			return nil, err
		}

		// Finally, collect the necessary fee
		senderAcc := k.accountKeeper.GetAccount(ctx, sender)

		err = sdkante.DeductFees(k.bankKeeper, ctx, senderAcc, sdk.NewCoins(microtxFeeCoin))
		if err != nil {
			ctx.Logger().Error("Could not deduct MsgMicrotx fee!", "error", err, "account", senderAcc, "fee", microtxFee)
			return nil, err
		}
	}

	return &microtxFeeCoin, nil
}

// getMicrotxFeeForAmount Computes the fee a user must pay for any input `amount`, given the current `basisPoints`
func (k Keeper) getMicrotxFeeForAmount(amount sdkmath.Int, basisPoints uint64) sdkmath.Int {
	return sdk.NewDecFromInt(amount).
		MulInt64(int64(basisPoints)).
		QuoInt64(int64(BasisPointDivisor)).
		TruncateInt()
}

// ========================================================================================================
// 												LIQUIFY ACCOUNT
// ========================================================================================================

// LiquifyAccount delegates the liquify account request to the keeper, with stateful validation
func (m *msgServer) Liquify(c context.Context, msg *types.MsgLiquify) (*types.MsgLiquifyResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	// The following validation logic has been copied from x/bank in the sdk
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	senderAcc := m.accountKeeper.GetAccount(ctx, sender)
	if !IsEthermintAccount(senderAcc) {
		return nil, errorsmod.Wrap(sdkerrors.ErrorInvalidSigner, "liquid infrastructure accounts must use ethermint keys, perhaps this is the first message the sender has sent?")
	}

	if m.Keeper.IsLiquidAccount(ctx, sender) {
		return nil, types.ErrAccountAlreadyLiquid
	}

	// Call the actual liquify implementation
	nft, err := m.Keeper.DoLiquify(ctx, sender)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to liquify account")
	}

	return &types.MsgLiquifyResponse{
		Account: &types.LiquidInfrastructureAccount{
			Owner:      sender.String(),
			Account:    sender.String(),
			NftAddress: nft.Hex(),
		},
	}, err
}
