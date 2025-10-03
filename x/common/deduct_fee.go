package common

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const BasisPointDivisor uint64 = 10000

type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI
}

type BankKeeper interface {
	authtypes.BankKeeper
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// DeductBasisPointFee calculates and deducts a fee based on the provided basis points from the given account.
// It returns the calculated fee as a sdk.Coin. If the account does not have sufficient funds to cover the fee,
// an error is returned.
// If the calculated fee is zero, no deduction is made and a zero-value sdk.Coin is returned.
func DeductBasisPointFee(
	ctx sdk.Context,
	accountKeeper AccountKeeper,
	bankKeeper BankKeeper,
	basisPoints uint64,
	coin sdk.Coin,
	subject sdk.AccAddress,
) (sdk.Coin, error) {
	fee := CalculateBasisPointFee(coin.Amount, basisPoints)
	feeCoin := sdk.NewCoin(coin.Denom, fee)

	// Require that the minimum has been met
	if !fee.IsZero() { // Ignore fees too low to collect
		balance := bankKeeper.GetBalance(ctx, subject, feeCoin.Denom)
		if balance.IsLT(feeCoin) {
			err := errorsmod.Wrapf(
				sdkerrors.ErrInsufficientFee,
				"balance is insufficient to pay the fee (%v < %v)",
				balance.Amount,
				fee,
			)
			return sdk.Coin{}, err
		}

		// Finally, collect the necessary fee
		senderAcc := accountKeeper.GetAccount(ctx, subject)

		err := sdkante.DeductFees(bankKeeper, ctx, senderAcc, sdk.NewCoins(feeCoin))
		if err != nil {
			return sdk.Coin{}, err
		}
	}
	return feeCoin, nil
}

// CalculateBasisPointFee calculates the fee based on the given amount and basis points.
// One basis point is 0.0001 (1/100th of a percent), so a 1% fee would be 100 basis points.
func CalculateBasisPointFee(amount sdkmath.Int, basisPoints uint64) sdkmath.Int {
	return sdk.NewDecFromInt(amount).
		MulInt64(int64(basisPoints)).
		QuoInt64(int64(BasisPointDivisor)).
		TruncateInt()
}
