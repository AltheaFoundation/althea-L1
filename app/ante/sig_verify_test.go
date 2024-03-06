package ante_test

import (
	"errors"
	"fmt"
	"math"
	"strings"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	altheaconfig "github.com/althea-net/althea-L1/config"
)

// We duplicate some of the SDK-level tests in the event that changes to the local antehandler definition break
// signature verification, which would be a critical problem.
func (suite *AnteTestSuite) TestCosmosSignatureVerification() {
	var (
		txs     []sdk.Tx
		signers []sdk.AccAddress
	)

	suite.SetupTest()
	privkeys := []cryptotypes.PrivKey{suite.NewCosmosPrivkey(), suite.NewCosmosPrivkey(), suite.NewCosmosPrivkey(), suite.NewCosmosPrivkey(), suite.NewCosmosPrivkey()}
	var accounts []sdk.AccAddress
	for _, priv := range privkeys {
		acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, sdk.AccAddress(priv.PubKey().Address().Bytes()))
		suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
		accounts = append(accounts, acc.GetAddress())
	}

	testCases := []struct {
		name        string // testcase name
		setTestVars func() // set txs, , etc.
		expPass     bool   // expected pass
		expErr      error  // expected error
	}{
		{
			"Pass: good tx and signBytes",
			func() {
				builder := suite.CreateTestCosmosMsgSend(sdk.NewInt(150), altheaconfig.BaseDenom, sdk.NewInt(1*int64(math.Pow(10, 18))), accounts[0], accounts[1])
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, accounts[0])
				suite.SignTestCosmosTx(suite.ctx.ChainID(), builder, privkeys[0], acc.GetAccountNumber(), acc.GetSequence())
				txs = []sdk.Tx{builder.GetTx()}
				signers = []sdk.AccAddress{acc.GetAddress()}
			},
			true,
			nil,
		},
		{
			"Fail: no signatures",
			func() {
				builder := suite.CreateTestCosmosMsgSend(sdk.NewInt(150), altheaconfig.BaseDenom, sdk.NewInt(1*int64(math.Pow(10, 18))), accounts[0], accounts[1])
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, accounts[0])
				txs = []sdk.Tx{builder.GetTx()}
				signers = []sdk.AccAddress{acc.GetAddress()}
			},
			false,
			sdkerrors.ErrNoSignatures,
		},
		{
			"Fail: empty signature",
			func() {
				builder := suite.CreateTestCosmosMsgSend(sdk.NewInt(150), altheaconfig.BaseDenom, sdk.NewInt(1*int64(math.Pow(10, 18))), accounts[0], accounts[1])
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, accounts[0])
				txs = []sdk.Tx{builder.GetTx()}
				signers = []sdk.AccAddress{acc.GetAddress()}
				suite.SignTestCosmosTx(suite.ctx.ChainID(), builder, privkeys[0], acc.GetAccountNumber(), acc.GetSequence())
				sigs, err := builder.GetTx().GetSignaturesV2()
				suite.Require().NoError(err)

				// Overwrite the signature with empty bytes
				sigs[0].Data.(*signing.SingleSignatureData).Signature = []byte{}
				suite.Require().NoError(builder.SetSignatures(sigs...))
			},
			false,
			fmt.Errorf("signature verification failed;"), // Weirdly enough this is not in sdkerrors
		},

		{
			"Fail: wrong chainID",
			func() {
				builder := suite.CreateTestCosmosMsgSend(sdk.NewInt(150), altheaconfig.BaseDenom, sdk.NewInt(1*int64(math.Pow(10, 18))), accounts[0], accounts[1])
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, accounts[0])
				suite.SignTestCosmosTx("clearly-wrong-id_1234-1", builder, privkeys[0], acc.GetAccountNumber(), acc.GetSequence())
				txs = []sdk.Tx{builder.GetTx()}
				signers = []sdk.AccAddress{acc.GetAddress()}
			},
			false,
			sdkerrors.ErrUnauthorized,
		},
		{
			"Fail: wrong accSeq",
			func() {
				builder := suite.CreateTestCosmosMsgSend(sdk.NewInt(150), altheaconfig.BaseDenom, sdk.NewInt(1*int64(math.Pow(10, 18))), accounts[0], accounts[1])
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, accounts[0])
				suite.SignTestCosmosTx(suite.ctx.ChainID(), builder, privkeys[0], acc.GetAccountNumber(), acc.GetSequence()+1)
				txs = []sdk.Tx{builder.GetTx()}
				signers = []sdk.AccAddress{acc.GetAddress()}
			},
			false,
			sdkerrors.ErrWrongSequence,
		},
		{
			"Fail: wrong accNums",
			func() {
				builder := suite.CreateTestCosmosMsgSend(sdk.NewInt(150), altheaconfig.BaseDenom, sdk.NewInt(1*int64(math.Pow(10, 18))), accounts[0], accounts[1])
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, accounts[0])
				suite.SignTestCosmosTx(suite.ctx.ChainID(), builder, privkeys[0], acc.GetAccountNumber()+1, acc.GetSequence())
				txs = []sdk.Tx{builder.GetTx()}
				signers = []sdk.AccAddress{acc.GetAddress()}
			},
			false,
			sdkerrors.ErrUnauthorized,
		},
		{
			"Fail: wrong msg",
			func() {
				// Sign a valid tx
				builder := suite.CreateTestCosmosMsgSend(sdk.NewInt(150), altheaconfig.BaseDenom, sdk.NewInt(1*int64(math.Pow(10, 18))), accounts[0], accounts[1])
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, accounts[0])
				suite.SignTestCosmosTx(suite.ctx.ChainID(), builder, privkeys[0], acc.GetAccountNumber()+1, acc.GetSequence())
				sigs, err := builder.GetTx().GetSignaturesV2()
				suite.Require().NoError(err)
				// Generate a new tx to put the other's signature on
				builder = suite.CreateTestCosmosMsgSend(sdk.NewInt(150), altheaconfig.BaseDenom, sdk.NewInt(2*int64(math.Pow(10, 18))), accounts[0], accounts[2])
				suite.Require().NoError(builder.SetSignatures(sigs...))
				txs = []sdk.Tx{builder.GetTx()}
				signers = []sdk.AccAddress{acc.GetAddress()}

			},
			false,
			fmt.Errorf("signature verification failed;"), // Weirdly enough this is not in sdkerrors
		},
		{
			"Fail: sig byte manipulation",
			func() {
				// Sign a valid tx
				builder := suite.CreateTestCosmosMsgSend(sdk.NewInt(150), altheaconfig.BaseDenom, sdk.NewInt(1*int64(math.Pow(10, 18))), accounts[0], accounts[1])
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, accounts[0])
				suite.SignTestCosmosTx(suite.ctx.ChainID(), builder, privkeys[0], acc.GetAccountNumber()+1, acc.GetSequence())
				sigs, err := builder.GetTx().GetSignaturesV2()
				suite.Require().NoError(err)

				for _, sig := range sigs {
					single, isSingle := (sig.Data).(*signing.SingleSignatureData)
					if isSingle {
						single.Signature[0] = ^single.Signature[0]
					} else {
						panic("Unexpected multi signature in test")
					}

				}
				suite.Require().NoError(builder.SetSignatures(sigs...))

				txs = []sdk.Tx{builder.GetTx()}
				signers = []sdk.AccAddress{acc.GetAddress()}

			},
			false,
			fmt.Errorf("signature verification failed;"), // Weirdly enough this is not in sdkerrors
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			ctx, _ := suite.ctx.CacheContext()
			tc.setTestVars()
			for i, tx := range txs {
				signer := signers[i]
				suite.testCaseSetup(ctx, signer, true)
				ctx, anteErr := suite.anteHandler(ctx, tx, false)

				if tc.expPass {
					suite.Require().NoError(anteErr)
					suite.Require().NotNil(ctx)
				} else {
					switch {
					case anteErr != nil:
						suite.Require().Error(anteErr)
						suite.Require().True(errors.Is(anteErr, tc.expErr) || strings.Contains(anteErr.Error(), tc.expErr.Error()), "Expected error: %s, got: %s", tc.expErr, anteErr)

					default:
						suite.Fail("expected anteErr to be an error")
					}
				}
			}
		})

	}
}
