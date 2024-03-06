package ante

import (
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

// EthSetPubkeyDecorator sets the pubkey on the account for EVM Txs
// CONTRACT: Tx must contain a single ExtensionOptionsEthereumTx
//
// This decorator should come AFTER assigning From on the Tx, AFTER signature verification, and AFTER the account is stored
type EthSetPubkeyDecorator struct {
	ak        AccountKeeper
	evmKeeper *evmkeeper.Keeper
}

func NewEthSetPubkeyDecorator(ak AccountKeeper, evmKeeper *evmkeeper.Keeper) EthSetPubkeyDecorator {
	if evmKeeper == nil {
		panic("evm keeper is required for EthSetPubkeyDecorator")
	}

	return EthSetPubkeyDecorator{
		ak:        ak,
		evmKeeper: evmKeeper,
	}
}

// AnteHandle sets the pubkey for the account
func (espd EthSetPubkeyDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	chainID := espd.evmKeeper.ChainID()

	params := espd.evmKeeper.GetParams(ctx)

	ethCfg := params.ChainConfig.EthereumConfig(chainID)
	blockNum := big.NewInt(ctx.BlockHeight())

	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}

		// sender address should be in the tx cache from the previous AnteHandle call
		from := msgEthTx.GetFrom()
		if from == nil || from.Empty() {
			return ctx, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "from address cannot be empty")
		}

		fromAddr := common.BytesToAddress(from)
		acct := espd.evmKeeper.GetAccount(ctx, fromAddr)
		if acct.IsContract() {
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "tx submitted by a contract account: %s", fromAddr.Hex())
		}

		fromCosmosAcc := espd.ak.GetAccount(ctx, from)
		if fromCosmosAcc == nil {
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "account %s does not exist", from)
		}

		if fromCosmosAcc.GetPubKey() != nil {
			// Pubkey set, do not recover
			continue
		}

		// Recover the pubkey from the transaction's signature
		ethTx := msgEthTx.AsTransaction()

		pubkey, err := recoverPubKey(ethCfg, blockNum, ethTx)
		if err != nil {
			return ctx, sdkerrors.Wrapf(
				sdkerrors.ErrorInvalidSigner,
				"couldn't retrieve sender pubkey from the ethereum transaction: %s",
				err.Error(),
			)
		}

		// Set the pubkey on the account
		if err := fromCosmosAcc.SetPubKey(pubkey); err != nil {
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrInvalidPubKey, "failed to set pubkey on account: %s", err.Error())
		}
		espd.ak.SetAccount(ctx, fromCosmosAcc)
	}

	return next(ctx, tx, simulate)
}

// For this AnteHandler we need to basically copy some private geth functions to recover the pubkey
// which is only necessary because geth returns the address instead of the pubkey, which is a one way operation
// so here instead of calling ethtypes.MakeSigner() and then signer.Sender() we duplicate all the Sender()
// logic for each of the different transaction types and return the pubkey
// These senders are necessary because different EVM configs lead to different recovery params

// Transaction types.
const (
	LegacyTxType = iota
	AccessListTxType
	DynamicFeeTxType
)

var (
	ErrInvalidChainId     = errors.New("invalid chain id for signer")
	ErrTxTypeNotSupported = errors.New("transaction type not supported")
	ErrInvalidSig         = errors.New("invalid transaction v, r, s values")
)

// recoverPubKey converts the signature in ethTx to a public key
// this is highly dependent on the chain configuration and the transaction type
func recoverPubKey(config *params.ChainConfig, blockNumber *big.Int, ethTx *ethtypes.Transaction) (cryptotypes.PubKey, error) {
	var pubkeyUncompressed *ecdsa.PublicKey
	var err error
	switch {
	case config.IsLondon(blockNumber):
		signer := ethtypes.NewLondonSigner(config.ChainID)
		pubkeyUncompressed, err = londonPubKey(config, ethTx, signer)
	case config.IsBerlin(blockNumber):
		signer := ethtypes.NewEIP2930Signer(config.ChainID)
		pubkeyUncompressed, err = eip2930PubKey(config, ethTx, signer)
	case config.IsEIP155(blockNumber):
		signer := ethtypes.NewEIP155Signer(config.ChainID)
		pubkeyUncompressed, err = eip155PubKey(config, ethTx, signer)
	case config.IsHomestead(blockNumber):
		signer := ethtypes.HomesteadSigner{}
		pubkeyUncompressed, err = homesteadPubKey(ethTx, signer)
	default:
		signer := ethtypes.FrontierSigner{}
		pubkeyUncompressed, err = frontierPubKey(ethTx, signer)
	}

	if err != nil {
		return nil, err
	}

	pubkey := crypto.CompressPubkey(pubkeyUncompressed)
	return &ethsecp256k1.PubKey{Key: pubkey}, nil
}

// londonPubKey handles the recovery of the public key for London transactions, which can be EIP1559, EIP2930 and legacy txs
// this is functionally a copy of go-ethereum's londonSigner.Sender() function but calls recoverPubKeyBase() instead of recoverPlain()
func londonPubKey(config *params.ChainConfig, tx *ethtypes.Transaction, signer ethtypes.Signer) (*ecdsa.PublicKey, error) {
	if tx.Type() != DynamicFeeTxType {
		return eip2930PubKey(config, tx, ethtypes.NewEIP2930Signer(config.ChainID))
	}
	V, R, S := tx.RawSignatureValues()
	// DynamicFee txs are defined to use 0 and 1 as their recovery
	// id, add 27 to become equivalent to unprotected Homestead signatures.
	V = new(big.Int).Add(V, big.NewInt(27))
	if tx.ChainId().Cmp(config.ChainID) != 0 {
		return nil, ErrInvalidChainId
	}
	return recoverPubKeyBase(signer.Hash(tx), R, S, V, true)
}

// eip2930PubKey handles the recovery of the public key for legacy txs and access list (EIP2930) txs
// this is functionally a copy of go-ethereum's eip2930Signer.Sender() function but calls recoverPubKeyBase() instead of recoverPlain()
func eip2930PubKey(config *params.ChainConfig, tx *ethtypes.Transaction, signer ethtypes.Signer) (*ecdsa.PublicKey, error) {
	V, R, S := tx.RawSignatureValues()
	switch tx.Type() {
	case LegacyTxType:
		if !tx.Protected() {
			return homesteadPubKey(tx, ethtypes.HomesteadSigner{})
		}
		V = new(big.Int).Sub(V, new(big.Int).Mul(config.ChainID, big.NewInt(2)))
		V.Sub(V, big.NewInt(8))
	case AccessListTxType:
		// AL txs are defined to use 0 and 1 as their recovery
		// id, add 27 to become equivalent to unprotected Homestead signatures.
		V = new(big.Int).Add(V, big.NewInt(27))
	default:
		return nil, ErrTxTypeNotSupported
	}
	if tx.ChainId().Cmp(config.ChainID) != 0 {
		return nil, ErrInvalidChainId
	}
	return recoverPubKeyBase(signer.Hash(tx), R, S, V, true)

}

// eip155PubKey handles the recovery of the public key for EIP155 txs and unprotected homestead txs
// this is functionally a copy of go-ethereum's EIP155Signer.Sender() function but calls recoverPubKeyBase() instead of recoverPlain()
func eip155PubKey(config *params.ChainConfig, tx *ethtypes.Transaction, signer ethtypes.Signer) (*ecdsa.PublicKey, error) {
	if tx.Type() != LegacyTxType {
		return nil, ErrTxTypeNotSupported
	}
	if !tx.Protected() {
		return homesteadPubKey(tx, ethtypes.HomesteadSigner{})
	}
	if tx.ChainId().Cmp(config.ChainID) != 0 {
		return nil, ErrInvalidChainId
	}
	V, R, S := tx.RawSignatureValues()
	V = new(big.Int).Sub(V, new(big.Int).Mul(config.ChainID, big.NewInt(2)))
	V.Sub(V, big.NewInt(8))
	return recoverPubKeyBase(signer.Hash(tx), R, S, V, true)
}

// homesteadPubKey handles the recovery of the public key for non-EIP155 homestead txs
// this is functionally a copy of go-ethereum's HomesteadSigner.Sender() function but calls recoverPubKeyBase() instead of recoverPlain()
func homesteadPubKey(tx *ethtypes.Transaction, signer ethtypes.Signer) (*ecdsa.PublicKey, error) {
	if tx.Type() != LegacyTxType {
		return nil, ErrTxTypeNotSupported
	}
	v, r, s := tx.RawSignatureValues()
	return recoverPubKeyBase(signer.Hash(tx), r, s, v, true)

}

// frontierPubKey handles the recovery of the public key for non-EIP155 frontier txs
// this is functionally a copy of go-ethereum's FrontierSigner.Sender() function but calls recoverPubKeyBase() instead of recoverPlain()
func frontierPubKey(tx *ethtypes.Transaction, signer ethtypes.Signer) (*ecdsa.PublicKey, error) {
	if tx.Type() != LegacyTxType {
		return nil, ErrTxTypeNotSupported
	}
	v, r, s := tx.RawSignatureValues()
	return recoverPubKeyBase(signer.Hash(tx), r, s, v, false)

}

// recovers the public key from the signature's hash and components
// this is functionally a copy of go-ethereum's recoverPlain() function but returns the public key instead of the address
func recoverPubKeyBase(sighash common.Hash, R, S, Vb *big.Int, homestead bool) (*ecdsa.PublicKey, error) {
	if Vb.BitLen() > 8 {
		return nil, ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
		return nil, ErrInvalidSig
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the signature
	return crypto.SigToPub(sighash[:], sig)
}
