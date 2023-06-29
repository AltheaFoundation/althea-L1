package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/ethereum/go-ethereum/common"

	"github.com/althea-net/althea-chain/config"
	"github.com/althea-net/althea-chain/x/microtx/types"
)

// BasisPointDivisor used in calculating the MsgXfer fee amount to deduct
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
// 												XFER
// ========================================================================================================

// Xfer delegates the msg server's call to the keeper
func (m msgServer) Xfer(c context.Context, msg *types.MsgXfer) (*types.MsgXferResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	// The following validation logic has been copied from x/bank in the sdk
	if err := m.bankKeeper.IsSendEnabledCoins(ctx, msg.Amounts...); err != nil {
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
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", msg.Receiver)
	}

	// Call the actual transfer implementation
	if err := m.Keeper.Xfer(ctx, sender, receiver, msg.Amounts); err != nil {
		return nil, sdkerrors.Wrap(err, "unable to complete the transfer")
	}

	return &types.MsgXferResponse{}, err
}

// Xfer implements the transfer of funds from sender to receiver
// Due to the function of Tokenized Accounts, any Xfer must contain solely EVM compatible bank coins
func (k Keeper) Xfer(ctx sdk.Context, sender sdk.AccAddress, receiver sdk.AccAddress, amounts sdk.Coins) error {
	var erc20Amounts []*common.Address // Denoms of `amounts` converted to ERC20 addresses
	for _, amount := range amounts {
		// The native token is automatically usable within the EVM
		if amount.Denom == config.BaseDenom {
			continue
		}

		// Ensure the input tokens are actively registered as an ERC20-convertible token
		pair, found := k.erc20Keeper.GetTokenPair(ctx, k.erc20Keeper.GetTokenPairID(ctx, amount.Denom))
		if !found {
			return sdkerrors.Wrapf(types.ErrInvalidMicrotx, "token %v is not registered as an erc20, only evm-compatible tokens may be used", amount.Denom)
		}
		if !pair.Enabled {
			return sdkerrors.Wrapf(types.ErrInvalidMicrotx, "token %v is registered as an erc20 (%v), but the pair is not enabled", amount.Denom, pair.Erc20Address)
		}
		// Collect the ERC20s for later use in funneling
		erc20Address := common.HexToAddress(pair.Erc20Address)
		erc20Amounts = append(erc20Amounts, &erc20Address)
	}

	feesCollected, err := k.DeductXferFee(ctx, sender, amounts)

	if err != nil {
		return sdkerrors.Wrap(err, "unable to collect fees")
	}

	err = k.bankKeeper.SendCoins(ctx, sender, receiver, amounts)

	if err != nil {
		return sdkerrors.Wrap(err, "unable to send tokens via the bank module")
	}

	// Emit an event for the block's event log
	ctx.EventManager().EmitEvent(
		types.NewEventXfer(sender.String(), receiver.String(), amounts, feesCollected),
	)

	// Migrate balances to the NFT if the amounts are in excess of any configured threshold
	k.Logger(ctx).Info("Detecting and funneling excess balances for tokenized accounts")
	if err := k.RedirectTokenizedAccountExcessBalances(ctx, receiver, erc20Amounts); err != nil {
		return sdkerrors.Wrapf(err, "failed to redirect excess balances")
	}

	return nil
}

// checkAndDeductSendToEthFees asserts that the minimum chainFee has been met for the given sendAmount
func (k Keeper) DeductXferFee(ctx sdk.Context, sender sdk.AccAddress, sendAmounts sdk.Coins) (feeCollected sdk.Coins, err error) {
	// Compute the minimum fees which must be paid
	xferFeeBasisPoints, err := k.GetXferFeeBasisPoints(ctx)
	if err != nil {
		xferFeeBasisPoints = 0
	}
	var xferFees sdk.Coins
	for _, sendAmount := range sendAmounts {
		xferFee := k.getXferFeeForAmount(sendAmount.Amount, xferFeeBasisPoints)
		xferFeeCoin := sdk.NewCoin(sendAmount.Denom, xferFee)
		xferFees = xferFees.Add(xferFeeCoin)
	}

	// Require that the minimum has been met
	if !xferFees.IsZero() { // Ignore fees too low to collect
		balances := k.bankKeeper.GetAllBalances(ctx, sender)
		if xferFees.IsAnyGT(balances) {
			err := sdkerrors.Wrapf(
				sdkerrors.ErrInsufficientFee,
				"balances are insufficient, one of the needed fees are larger (%v > %v)",
				xferFees,
				balances,
			)
			return nil, err
		}

		// Finally, collect the necessary fee
		senderAcc := k.accountKeeper.GetAccount(ctx, sender)

		err = sdkante.DeductFees(k.bankKeeper, ctx, senderAcc, xferFees)
		if err != nil {
			ctx.Logger().Error("Could not deduct MsgXfer fee!", "error", err, "account", senderAcc, "fees", xferFees)
			return nil, err
		}
	}

	return xferFees, nil
}

// getXferFeeForAmount Computes the fee a user must pay for any input `amount`, given the current `basisPoints`
func (k Keeper) getXferFeeForAmount(amount sdk.Int, basisPoints uint64) sdk.Int {
	return sdk.NewDecFromInt(amount).
		QuoInt64(int64(BasisPointDivisor)).
		MulInt64(int64(basisPoints)).
		TruncateInt()
}

// ========================================================================================================
// 												TOKENIZE ACCOUNT
// ========================================================================================================

// TokenizeAccount delegates the tokenize request to the keeper, with stateful validation
func (m *msgServer) TokenizeAccount(c context.Context, msg *types.MsgTokenizeAccount) (*types.MsgTokenizeAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	// The following validation logic has been copied from x/bank in the sdk
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	senderAcc := m.accountKeeper.GetAccount(ctx, sender)
	if !IsEthermintAccount(senderAcc) {
		return nil, sdkerrors.Wrap(sdkerrors.ErrorInvalidSigner, "tokenized accounts must use ethermint keys, perhaps this is the first message the sender has sent?")
	}

	// Call the actual tokenize implementation
	nft, err := m.Keeper.DoTokenizeAccount(ctx, sender)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to tokenize account")
	}

	return &types.MsgTokenizeAccountResponse{
		Account: &types.TokenizedAccount{
			Owner:            sender.String(),
			TokenizedAccount: sender.String(),
			NftAddress:       nft.Hex(),
		},
	}, err
}
