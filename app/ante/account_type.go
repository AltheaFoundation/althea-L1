package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	ethsecp "github.com/evmos/ethermint/crypto/ethsecp256k1"
)

type SignedTx interface {
	sdk.Tx
	GetSigners() []sdk.AccAddress
}

// SetAccountTypeDecorator sets the account type to EthAccount for Ethermint pubkeys
// PubKeys must be set in context before this decorator runs
// ethAccountProto initializes an Ethermint account
//
// CONTRACT: Tx must implement SignedTx interface (sdk.Tx interface + GetSigners() method)
type SetAccountTypeDecorator struct {
	ak              AccountKeeper
	ethAccountProto func(sdk.AccAddress) authtypes.AccountI
}

func NewSetAccountTypeDecorator(ak AccountKeeper, ethAccountProto func(sdk.AccAddress) authtypes.AccountI) SetAccountTypeDecorator {
	return SetAccountTypeDecorator{
		ak:              ak,
		ethAccountProto: ethAccountProto,
	}
}

func (satd SetAccountTypeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	sigTx, ok := tx.(SignedTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "invalid tx type")
	}

	signers := sigTx.GetSigners()

	for _, signer := range signers {
		acc, err := authante.GetSignerAcc(ctx, satd.ak, signer)
		if err != nil {
			return ctx, err
		}
		storedPubKey := acc.GetPubKey()
		if storedPubKey == nil {
			return next(ctx, tx, simulate)
		}

		// If the account is a BaseAccount and the pubkey is an Ethermint pubkey, replace the account with an EthAccount
		baseAccount, ok := acc.(*authtypes.BaseAccount)
		if storedPubKey.Type() == ethsecp.KeyType && ok {
			replacement := satd.ethAccountProto(signer)

			// Copy over the BaseAccount values
			if err = replacement.SetAccountNumber(baseAccount.GetAccountNumber()); err != nil {
				return ctx, sdkerrors.Wrap(err, "unable to set account number on replacement EthAccount")
			}
			if err = replacement.SetSequence(baseAccount.GetSequence()); err != nil {
				return ctx, sdkerrors.Wrap(err, "unable to set sequence on replacement EthAccount")
			}
			if err = replacement.SetPubKey(storedPubKey); err != nil {
				return ctx, sdkerrors.Wrap(err, "unable to set pubkey on replacement EthAccount")
			}

			satd.ak.SetAccount(ctx, replacement)
		}
	}
	return next(ctx, tx, simulate)
}
