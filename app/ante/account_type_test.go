package ante_test

import (
	"math/big"

	altheaconfig "github.com/AltheaFoundation/althea-L1/config"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/tests"
	ethtypes "github.com/evmos/ethermint/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

func isEthAccount(acc authtypes.AccountI) bool {
	_, ok := acc.(*ethtypes.EthAccount)
	return ok
}

func isCosmosAccount(acc authtypes.AccountI) bool {
	_, ok := acc.(*authtypes.BaseAccount)
	return ok
}
func isCosmosPubKey(pubKey cryptotypes.PubKey) bool {
	_, ok := pubKey.(*secp256k1.PubKey)
	return ok
}
func isEthPubKey(pubKey cryptotypes.PubKey) bool {
	_, ok := pubKey.(*ethsecp256k1.PubKey)
	return ok
}

func (suite *AnteTestSuite) testCaseSetup(ctx sdk.Context, addr sdk.AccAddress, isBaseAccount bool) {
	var acc authtypes.AccountI
	suite.enableFeemarket = false

	if isBaseAccount {
		acc = suite.app.AccountKeeper.NewAccountWithAddress(ctx, addr)
		suite.Require().True(isCosmosAccount(acc), "account type should be BaseAccount")
	} else {
		acc = ethtypes.ProtoAccount()
		suite.Require().NoError(acc.SetAddress(addr))
		suite.app.AccountKeeper.SetAccount(ctx, acc)
		acc = suite.app.AccountKeeper.GetAccount(ctx, addr)
		suite.Require().True(isEthAccount(acc), "account type should be EthAccount")
	}
	evmAddr := common.BytesToAddress(addr.Bytes())
	suite.Require().NoError(suite.app.EvmKeeper.SetBalance(ctx, evmAddr, big.NewInt(10000000000)))

	suite.app.FeemarketKeeper.SetBaseFee(ctx, big.NewInt(100))
}

// Checks that the account types are updated correctly
func (suite *AnteTestSuite) TestAccountTypeAnteHandler() {
	to := tests.GenerateAddress()
	suite.SetupTest()

	// Setup will create the account as either a BaseAccount or an EthAccount (based on isBaseAccount) and set the balance

	testCases := []struct {
		name           string                                         // A name which is printed when the test fails
		txPrivKeyFn    func() (client.TxBuilder, cryptotypes.PrivKey) // generates a tx and privkey for the test
		signCosmosTx   bool
		expPass        bool                          // If the test should pass or fail
		startBaseAcc   bool                          // If the account should be a BaseAccount or an EthAccount to start
		correctPubKey  func(cryptotypes.PubKey) bool // The expected pubkey type on the account after tx submission
		correctAccount func(authtypes.AccountI) bool // The expected account type after the tx is processed
	}{
		{
			/*
				This one starts as a Cosmos BaseAccount and sends a signed EVM tx, we want to see the account type change to EthAccount
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
			true,
			isEthPubKey,
			isEthAccount, // AnteHandler must change account type
		},
		{
			/*
				This one starts as a EthAccount and sends a signed EVM tx, we want to see the account type stay the same EthAccount
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
			isEthPubKey,
			isEthAccount, // AnteHandler must not change account type
		},
		{
			/*
				This one starts as a Cosmos BaseAccount and sends a Cosmos Tx, we want to see the account type stay the same BaseAccount
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
			isCosmosPubKey,
			isCosmosAccount, // AnteHandler must not change account type
		},
		{
			/*
				This one starts as an EthAccount and sends a Cosmos Tx, we want to see the account type stay the same BaseAccount
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
			false,
			isCosmosPubKey,
			isEthAccount, // The AnteHandler does not fix EthAccount with Cosmos pubkey
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
				suite.Require().True(tc.correctAccount(acc), "account type incorrect after AnteHandler")
			} else {
				suite.Require().Error(err)
			}
			suite.Require().True(tc.correctPubKey(pubKey), "PubKey type incorrect after AnteHandler")
		})
	}
}
