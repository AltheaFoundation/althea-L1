package ante_test

import (
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/evmos/ethermint/tests"
	evmtypes "github.com/evmos/ethermint/x/evm/types"

	altheaconfig "github.com/althea-net/althea-L1/config"
)

// These tests have been copied from ethermint but duplicated here to ensure that
// our changes to the AnteHandler do not cause conflicts
func (suite *AnteTestSuite) TestAnteHandler() {
	var acc authtypes.AccountI
	addr, privKey := tests.NewAddrKey()
	to := tests.GenerateAddress()

	setup := func() {
		suite.enableFeemarket = false
		suite.SetupTest() // reset

		acc = suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
		suite.Require().NoError(acc.SetSequence(1))
		suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

		suite.Require().NoError(suite.app.EvmKeeper.SetBalance(suite.ctx, addr, big.NewInt(10000000000)))

		suite.app.FeemarketKeeper.SetBaseFee(suite.ctx, big.NewInt(100))
	}

	testCases := []struct {
		name      string
		txFn      func() sdk.Tx
		checkTx   bool
		reCheckTx bool
		expPass   bool
	}{
		{
			"success - DeliverTx (contract)",
			func() sdk.Tx {
				signedContractTx := evmtypes.NewTxContract(
					suite.app.EvmKeeper.ChainID(),
					1,
					big.NewInt(10),
					100000,
					big.NewInt(150),
					big.NewInt(200),
					nil,
					nil,
					nil,
				)
				signedContractTx.From = addr.Hex()

				tx := suite.CreateTestEVMTx(signedContractTx, privKey, 1, false)
				return tx
			},
			false, false, true,
		},
		{
			"success - CheckTx (contract)",
			func() sdk.Tx {
				signedContractTx := evmtypes.NewTxContract(
					suite.app.EvmKeeper.ChainID(),
					1,
					big.NewInt(10),
					100000,
					big.NewInt(150),
					big.NewInt(200),
					nil,
					nil,
					nil,
				)
				signedContractTx.From = addr.Hex()

				tx := suite.CreateTestEVMTx(signedContractTx, privKey, 1, false)
				return tx
			},
			true, false, true,
		},
		{
			"success - ReCheckTx (contract)",
			func() sdk.Tx {
				signedContractTx := evmtypes.NewTxContract(
					suite.app.EvmKeeper.ChainID(),
					1,
					big.NewInt(10),
					100000,
					big.NewInt(150),
					big.NewInt(200),
					nil,
					nil,
					nil,
				)
				signedContractTx.From = addr.Hex()

				tx := suite.CreateTestEVMTx(signedContractTx, privKey, 1, false)
				return tx
			},
			false, true, true,
		},
		{
			"success - DeliverTx",
			func() sdk.Tx {
				signedTx := evmtypes.NewTx(
					suite.app.EvmKeeper.ChainID(),
					1,
					&to,
					big.NewInt(10),
					100000,
					big.NewInt(150),
					big.NewInt(200),
					nil,
					nil,
					nil,
				)
				signedTx.From = addr.Hex()

				tx := suite.CreateTestEVMTx(signedTx, privKey, 1, false)
				return tx
			},
			false, false, true,
		},
		{
			"success - CheckTx",
			func() sdk.Tx {
				signedTx := evmtypes.NewTx(
					suite.app.EvmKeeper.ChainID(),
					1,
					&to,
					big.NewInt(10),
					100000,
					big.NewInt(150),
					big.NewInt(200),
					nil,
					nil,
					nil,
				)
				signedTx.From = addr.Hex()

				tx := suite.CreateTestEVMTx(signedTx, privKey, 1, false)
				return tx
			},
			true, false, true,
		},
		{
			"success - ReCheckTx",
			func() sdk.Tx {
				signedTx := evmtypes.NewTx(
					suite.app.EvmKeeper.ChainID(),
					1,
					&to,
					big.NewInt(10),
					100000,
					big.NewInt(150),
					big.NewInt(200),
					nil,
					nil,
					nil,
				)
				signedTx.From = addr.Hex()

				tx := suite.CreateTestEVMTx(signedTx, privKey, 1, false)
				return tx
			}, false, true, true,
		},
		{
			"success - CheckTx (cosmos tx not signed)",
			func() sdk.Tx {
				signedTx := evmtypes.NewTx(
					suite.app.EvmKeeper.ChainID(),
					1,
					&to,
					big.NewInt(10),
					100000,
					big.NewInt(150),
					big.NewInt(200),
					nil,
					nil,
					nil,
				)
				signedTx.From = addr.Hex()

				tx := suite.CreateTestEVMTx(signedTx, privKey, 1, false)
				return tx
			}, false, true, true,
		},
		{
			"fail - CheckTx (cosmos tx is not valid)",
			func() sdk.Tx {
				signedTx := evmtypes.NewTx(suite.app.EvmKeeper.ChainID(), 1, &to, big.NewInt(10), 100000, big.NewInt(1), nil, nil, nil, nil)
				signedTx.From = addr.Hex()

				txBuilder := suite.CreateTestEVMTxBuilder(signedTx, privKey, 1, false)
				// bigger than MaxGasWanted
				txBuilder.SetGasLimit(uint64(1 << 63))
				return txBuilder.GetTx()
			}, true, false, false,
		},
		{
			"fail - CheckTx (memo too long)",
			func() sdk.Tx {
				signedTx := evmtypes.NewTx(suite.app.EvmKeeper.ChainID(), 1, &to, big.NewInt(10), 100000, big.NewInt(1), nil, nil, nil, nil)
				signedTx.From = addr.Hex()

				txBuilder := suite.CreateTestEVMTxBuilder(signedTx, privKey, 1, false)
				txBuilder.SetMemo(strings.Repeat("*", 257))
				return txBuilder.GetTx()
			}, true, false, false,
		},
		{
			"fail - CheckTx (ExtensionOptionsEthereumTx not set)",
			func() sdk.Tx {
				signedTx := evmtypes.NewTx(suite.app.EvmKeeper.ChainID(), 1, &to, big.NewInt(10), 100000, big.NewInt(1), nil, nil, nil, nil)
				signedTx.From = addr.Hex()

				txBuilder := suite.CreateTestEVMTxBuilder(signedTx, privKey, 1, false, true)
				return txBuilder.GetTx()
			}, true, false, false,
		},
		// Based on EVMBackend.SendTransaction, for cosmos tx, forcing null for some fields except ExtensionOptions, Fee, MsgEthereumTx
		// should be part of consensus
		{
			"fail - DeliverTx (cosmos tx signed)",
			func() sdk.Tx {
				nonce, err := suite.app.AccountKeeper.GetSequence(suite.ctx, acc.GetAddress())
				suite.Require().NoError(err)
				signedTx := evmtypes.NewTx(suite.app.EvmKeeper.ChainID(), nonce, &to, big.NewInt(10), 100000, big.NewInt(1), nil, nil, nil, nil)
				signedTx.From = addr.Hex()

				tx := suite.CreateTestEVMTx(signedTx, privKey, 1, true)
				return tx
			}, false, false, false,
		},
		{
			"fail - DeliverTx (cosmos tx with memo)",
			func() sdk.Tx {
				nonce, err := suite.app.AccountKeeper.GetSequence(suite.ctx, acc.GetAddress())
				suite.Require().NoError(err)
				signedTx := evmtypes.NewTx(suite.app.EvmKeeper.ChainID(), nonce, &to, big.NewInt(10), 100000, big.NewInt(1), nil, nil, nil, nil)
				signedTx.From = addr.Hex()

				txBuilder := suite.CreateTestEVMTxBuilder(signedTx, privKey, 1, false)
				txBuilder.SetMemo("memo for cosmos tx not allowed")
				return txBuilder.GetTx()
			}, false, false, false,
		},
		{
			"fail - DeliverTx (cosmos tx with timeoutheight)",
			func() sdk.Tx {
				nonce, err := suite.app.AccountKeeper.GetSequence(suite.ctx, acc.GetAddress())
				suite.Require().NoError(err)
				signedTx := evmtypes.NewTx(suite.app.EvmKeeper.ChainID(), nonce, &to, big.NewInt(10), 100000, big.NewInt(1), nil, nil, nil, nil)
				signedTx.From = addr.Hex()

				txBuilder := suite.CreateTestEVMTxBuilder(signedTx, privKey, 1, false)
				txBuilder.SetTimeoutHeight(10)
				return txBuilder.GetTx()
			}, false, false, false,
		},
		{
			"fail - DeliverTx (invalid fee amount)",
			func() sdk.Tx {
				nonce, err := suite.app.AccountKeeper.GetSequence(suite.ctx, acc.GetAddress())
				suite.Require().NoError(err)
				signedTx := evmtypes.NewTx(suite.app.EvmKeeper.ChainID(), nonce, &to, big.NewInt(10), 100000, big.NewInt(1), nil, nil, nil, nil)
				signedTx.From = addr.Hex()

				txBuilder := suite.CreateTestEVMTxBuilder(signedTx, privKey, 1, false)

				txData, err := evmtypes.UnpackTxData(signedTx.Data)
				suite.Require().NoError(err)

				expFee := txData.Fee()
				invalidFee := new(big.Int).Add(expFee, big.NewInt(1))
				invalidFeeAmount := sdk.Coins{sdk.NewCoin(altheaconfig.BaseDenom, sdk.NewIntFromBigInt(invalidFee))}
				txBuilder.SetFeeAmount(invalidFeeAmount)
				return txBuilder.GetTx()
			}, false, false, false,
		},
		{
			"fail - DeliverTx (invalid fee gaslimit)",
			func() sdk.Tx {
				nonce, err := suite.app.AccountKeeper.GetSequence(suite.ctx, acc.GetAddress())
				suite.Require().NoError(err)
				signedTx := evmtypes.NewTx(suite.app.EvmKeeper.ChainID(), nonce, &to, big.NewInt(10), 100000, big.NewInt(1), nil, nil, nil, nil)
				signedTx.From = addr.Hex()

				txBuilder := suite.CreateTestEVMTxBuilder(signedTx, privKey, 1, false)

				expGasLimit := signedTx.GetGas()
				invalidGasLimit := expGasLimit + 1
				txBuilder.SetGasLimit(invalidGasLimit)
				return txBuilder.GetTx()
			}, false, false, false,
		},
		{
			"success - DeliverTx EIP712 signed Cosmos Tx with MsgSend",
			func() sdk.Tx {
				from := acc.GetAddress()
				amount := sdk.NewCoins(sdk.NewCoin(altheaconfig.BaseDenom, sdk.NewInt(200000)))
				gas := uint64(200000)
				txBuilder := suite.CreateTestEIP712TxBuilderMsgSend(from, privKey, suite.ctx.ChainID(), gas, amount)
				return txBuilder.GetTx()
			}, false, false, true,
		},
		{
			"success - DeliverTx EIP712 signed Cosmos Tx with DelegateMsg",
			func() sdk.Tx {
				from := acc.GetAddress()
				coinAmount := sdk.NewCoin(altheaconfig.BaseDenom, sdk.NewInt(200000))
				amount := sdk.NewCoins(coinAmount)
				gas := uint64(200000)
				txBuilder := suite.CreateTestEIP712TxBuilderMsgDelegate(from, privKey, suite.ctx.ChainID(), gas, amount)
				return txBuilder.GetTx()
			}, false, false, true,
		},
		{
			"fails - DeliverTx EIP712 signed Cosmos Tx with wrong Chain ID",
			func() sdk.Tx {
				from := acc.GetAddress()
				amount := sdk.NewCoins(sdk.NewCoin(altheaconfig.BaseDenom, sdk.NewInt(20)))
				gas := uint64(200000)
				txBuilder := suite.CreateTestEIP712TxBuilderMsgSend(from, privKey, suite.ctx.ChainID(), gas, amount)
				return txBuilder.GetTx()
			}, false, false, false,
		},
		{
			"fails - DeliverTx EIP712 signed Cosmos Tx with different gas fees",
			func() sdk.Tx {
				from := acc.GetAddress()
				amount := sdk.NewCoins(sdk.NewCoin(altheaconfig.BaseDenom, sdk.NewInt(20)))
				gas := uint64(200000)
				txBuilder := suite.CreateTestEIP712TxBuilderMsgSend(from, privKey, suite.ctx.ChainID(), gas, amount)
				txBuilder.SetGasLimit(uint64(300000))
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(altheaconfig.BaseDenom, sdk.NewInt(30))))
				return txBuilder.GetTx()
			}, false, false, false,
		},
		{
			"fails - DeliverTx EIP712 signed Cosmos Tx with empty signature",
			func() sdk.Tx {
				from := acc.GetAddress()
				amount := sdk.NewCoins(sdk.NewCoin(altheaconfig.BaseDenom, sdk.NewInt(20)))
				gas := uint64(200000)
				txBuilder := suite.CreateTestEIP712TxBuilderMsgSend(from, privKey, suite.ctx.ChainID(), gas, amount)
				// nolint: exhaustruct
				sigsV2 := signing.SignatureV2{}
				// nolint: errcheck
				txBuilder.SetSignatures(sigsV2)
				return txBuilder.GetTx()
			}, false, false, false,
		},
		{
			"fails - DeliverTx EIP712 signed Cosmos Tx with invalid sequence",
			func() sdk.Tx {
				from := acc.GetAddress()
				amount := sdk.NewCoins(sdk.NewCoin(altheaconfig.BaseDenom, sdk.NewInt(20)))
				gas := uint64(200000)
				txBuilder := suite.CreateTestEIP712TxBuilderMsgSend(from, privKey, suite.ctx.ChainID(), gas, amount)
				nonce, err := suite.app.AccountKeeper.GetSequence(suite.ctx, acc.GetAddress())
				suite.Require().NoError(err)
				// nolint: exhaustruct
				sigsV2 := signing.SignatureV2{
					PubKey: privKey.PubKey(),
					Data: &signing.SingleSignatureData{
						SignMode: signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
					},
					Sequence: nonce - 1,
				}
				suite.Require().NoError(txBuilder.SetSignatures(sigsV2))
				return txBuilder.GetTx()
			}, false, false, false,
		},
		{
			"fails - DeliverTx EIP712 signed Cosmos Tx with invalid signMode",
			func() sdk.Tx {
				from := acc.GetAddress()
				amount := sdk.NewCoins(sdk.NewCoin(altheaconfig.BaseDenom, sdk.NewInt(20)))
				gas := uint64(200000)
				txBuilder := suite.CreateTestEIP712TxBuilderMsgSend(from, privKey, suite.ctx.ChainID(), gas, amount)
				nonce, err := suite.app.AccountKeeper.GetSequence(suite.ctx, acc.GetAddress())
				suite.Require().NoError(err)
				// nolint: exhaustruct
				sigsV2 := signing.SignatureV2{
					PubKey: privKey.PubKey(),
					Data: &signing.SingleSignatureData{
						SignMode: signing.SignMode_SIGN_MODE_UNSPECIFIED,
					},
					Sequence: nonce,
				}
				suite.Require().NoError(txBuilder.SetSignatures(sigsV2))
				return txBuilder.GetTx()
			}, false, false, false,
		},
		{
			"fails - invalid from",
			func() sdk.Tx {
				msg := evmtypes.NewTxContract(
					suite.app.EvmKeeper.ChainID(),
					1,
					big.NewInt(10),
					100000,
					big.NewInt(150),
					big.NewInt(200),
					nil,
					nil,
					nil,
				)
				msg.From = addr.Hex()
				tx := suite.CreateTestEVMTx(msg, privKey, 1, false)
				msg = tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)
				msg.From = addr.Hex()
				return tx
			}, true, false, false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			setup()

			suite.ctx = suite.ctx.WithIsCheckTx(tc.checkTx).WithIsReCheckTx(tc.reCheckTx)

			// expConsumed := params.TxGasContractCreation + params.TxGas
			_, err := suite.anteHandler(suite.ctx, tc.txFn(), false)

			// suite.Require().Equal(consumed, ctx.GasMeter().GasConsumed())

			if tc.expPass {
				suite.Require().NoError(err)
				// suite.Require().Equal(int(expConsumed), int(suite.ctx.GasMeter().GasConsumed()))
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
