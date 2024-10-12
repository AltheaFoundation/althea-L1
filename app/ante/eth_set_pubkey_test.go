package ante_test

import (
	"math/big"

	altheaconfig "github.com/AltheaFoundation/althea-L1/config"
	"github.com/cosmos/cosmos-sdk/client"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/ethermint/tests"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

// Checks that the PubKey types are updated correctly, and that the new ante handler succeeds where the old one failed
func (suite *AnteTestSuite) TestEthSetPubkeyHandler() {
	to := tests.GenerateAddress()
	suite.SetupTest()

	testCases := []struct {
		name          string                                         // A name which is printed when the test fails
		txPrivKeyFn   func() (client.TxBuilder, cryptotypes.PrivKey) // generates a tx and privkey for the test
		signCosmosTx  bool
		expPass       bool                          // If the test should pass or fail
		expOldPubKey  bool                          // If the old antehandler would set a pubkey
		startBaseAcc  bool                          // If the account should be a BaseAccount or an EthAccount to start
		correctPubKey func(cryptotypes.PubKey) bool // The expected pubkey type on the account after tx submission
	}{
		{
			/*
				This one starts as a Cosmos BaseAccount and sends a signed EVM tx, we want to see an Eth PubKey
			*/
			"EVM Tx Cosmos BaseAccount",
			func() (client.TxBuilder, cryptotypes.PrivKey) {
				privKey := suite.NewEthermintPrivkey()
				addr := privKey.PubKey().Address()
				unsignedTx := evmtypes.NewTx(
					suite.app.EvmKeeper.ChainID(),
					0,
					&to,
					big.NewInt(10),
					100000,
					big.NewInt(150),
					big.NewInt(200),
					nil,
					nil,
					nil,
				)
				unsignedTx.From = addr.String()

				builder := suite.CreateTestEVMTxBuilder(unsignedTx, privKey, 1, false)
				return builder, privKey
			},
			false,
			true,
			false,
			true,
			isEthPubKey,
		},
		{
			/*
				This one starts as a EthAccount and sends a signed EVM tx, we want to see an Eth pubkey
			*/
			"EVM Tx EthAccount",
			func() (client.TxBuilder, cryptotypes.PrivKey) {
				privKey := suite.NewEthermintPrivkey()
				addr := privKey.PubKey().Address()
				unsignedTx := evmtypes.NewTx(
					suite.app.EvmKeeper.ChainID(),
					0,
					&to,
					big.NewInt(10),
					100000,
					big.NewInt(150),
					big.NewInt(200),
					nil,
					nil,
					nil,
				)
				unsignedTx.From = addr.String()

				builder := suite.CreateTestEVMTxBuilder(unsignedTx, privKey, 1, false)
				return builder, privKey
			},
			false,
			true,
			false,
			false,
			isEthPubKey,
		},
		{
			/*
				This one starts as a Cosmos BaseAccount and sends a Cosmos Tx, we want to see a Cosmos PubKey
			*/
			"Cosmos Tx Cosmos BaseAccount",
			func() (client.TxBuilder, cryptotypes.PrivKey) {
				privKey := suite.NewCosmosPrivkey()
				addr := sdk.AccAddress(privKey.PubKey().Address().Bytes())
				denom := altheaconfig.BaseDenom
				amount := sdk.NewInt(200000)
				txBuilder := suite.CreateTestCosmosMsgSend(sdk.NewInt(150), denom, amount, addr, addr)
				return txBuilder, privKey
			},
			true,
			true,
			true,
			true,
			isCosmosPubKey,
		},
		{
			/*
				This one starts as an EthAccount and sends a Cosmos Tx, we want to see a Cosmos Pubkey
			*/
			"Cosmos Tx EthAccount",
			func() (client.TxBuilder, cryptotypes.PrivKey) {
				privKey := suite.NewCosmosPrivkey()
				addr := sdk.AccAddress(privKey.PubKey().Address().Bytes())
				denom := altheaconfig.BaseDenom
				amount := sdk.NewInt(200000)
				txBuilder := suite.CreateTestCosmosMsgSend(sdk.NewInt(150), denom, amount, addr, addr)
				return txBuilder, privKey
			},
			true,
			true,
			true,
			false,
			isCosmosPubKey,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx, _ := suite.ctx.CacheContext()
			builder, privKey := tc.txPrivKeyFn()
			addr := sdk.AccAddress(privKey.PubKey().Address())
			// Create the account for the test, which will be a BaseAccount or an EthAccount depending on startBaseAcc
			suite.testCaseSetup(ctx, addr, tc.startBaseAcc)
			acc := suite.app.AccountKeeper.GetAccount(ctx, addr)

			if tc.signCosmosTx {
				builder = suite.SignTestCosmosTx(ctx.ChainID(), builder, privKey, acc.GetAccountNumber(), acc.GetSequence())
			}

			tx := builder.GetTx()

			// Perform the test by executing the anteHandler on the tx
			_, err := suite.anteHandler(ctx, tx, false)

			acc = suite.app.AccountKeeper.GetAccount(ctx, addr)
			pubKey := acc.GetPubKey()
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
			suite.Require().True(tc.correctPubKey(pubKey), "PubKey type incorrect after AnteHandler")
		})
	}
}
