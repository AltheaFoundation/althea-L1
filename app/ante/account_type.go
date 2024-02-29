package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	ethsecp "github.com/evmos/ethermint/crypto/ethsecp256k1"
	ethtypes "github.com/evmos/ethermint/types"
)

type SignedTx interface {
	sdk.Tx
	GetSigners() []sdk.AccAddress
}

// SetAccountTypeDecorator sets the account type to EthAccount for Ethermint pubkeys
// PubKeys must be set in context before this decorator runs
// CONTRACT: Tx must implement SignedTx interface (sdk.Tx interface + GetSigners() method)
//
// Note that this is largely a copy of the SetPubKeyDecorator from the auth module, with the only difference being
// that it sets the account type to EthAccount for Ethermint pubkeys if the original account was a BaseAccount
type SetAccountTypeDecorator struct {
	ak AccountKeeper
}

func NewSetAccountTypeDecorator(ak AccountKeeper) SetAccountTypeDecorator {
	return SetAccountTypeDecorator{
		ak: ak,
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
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrInvalidPubKey,
				"pubKey has not been set for signer: %s", signer.String())
		}

		// If the account is a BaseAccount and the pubkey is an Ethermint pubkey, replace the account with an EthAccount
		baseAccount, ok := acc.(*authtypes.BaseAccount)
		if storedPubKey.Type() == ethsecp.KeyType && ok {
			replacement := ethtypes.ProtoAccount() // Sets the codehash for us, but does not set any BaseAccount values

			// Copy over the BaseAccount values, which may include everything EXCEPT the PubKey
			if err = replacement.SetAddress(baseAccount.GetAddress()); err != nil {
				return ctx, sdkerrors.Wrap(err, "unable to set address on replacement EthAccount")
			}
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
