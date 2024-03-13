package ante_test

import (
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	altheaconfig "github.com/althea-net/althea-L1/config"
	microtxtypes "github.com/althea-net/althea-L1/x/microtx/types"
)

// ----------------
// This file checks that the configured bypass decorators are being bypassed as expected, which gives 4 possible scenarios:
//	1. There are no gasfree messages (no bypass)
//  2. MsgMicrotx is the only gasfree message (expected, only bypass on txs containing only MsgMicrotx)
//  3. MsgSend is the only gasfree message (only bypass on txs containing only MsgSend)
//  4. MsgSend AND MsgMicrotx are the only gasfree messages (only bypass on txs containing MsgSend AND/OR MsgMicrotx)

func runBypassTest(suite *AnteTestSuite, expErrorStr string, gasfreeMicrotxCtx, gasfreeSendCtx, noGasfreeCtx, bothGasfreeCtx sdk.Context, msgMicrotxTx, msgSendTx, bothTx sdk.Tx) error {
	// --- Microtx is the only Gasfree Msg type ---
	// Expect bypass for the Microtx tx
	cached, _ := gasfreeMicrotxCtx.CacheContext()
	if _, err := suite.anteHandler(cached, msgMicrotxTx, false); err != nil {
		return sdkerrors.Wrap(err, "microtx gasfree expected no error")
	}
	// Expect failure for the Send tx
	cached, _ = gasfreeMicrotxCtx.CacheContext()
	if _, err := suite.anteHandler(cached, msgSendTx, false); !strings.Contains(err.Error(), expErrorStr) {
		return sdkerrors.Wrap(err, "microtx gasfree sent send - expected error")
	}
	// Expect failure for both msg tx
	cached, _ = gasfreeMicrotxCtx.CacheContext()
	if _, err := suite.anteHandler(cached, bothTx, false); !strings.Contains(err.Error(), expErrorStr) {
		return sdkerrors.Wrap(err, "microtx gasfree sent send and microtx - expected error")
	}

	// --- Send is the only Gasfree Msg type ---
	cached, _ = gasfreeSendCtx.CacheContext()
	if _, err := suite.anteHandler(cached, msgMicrotxTx, false); !strings.Contains(err.Error(), expErrorStr) {
		return sdkerrors.Wrap(err, "send gasfree sent microtx - expected error")
	}
	// Expect failure for the Send tx
	cached, _ = gasfreeSendCtx.CacheContext()
	if _, err := suite.anteHandler(cached, msgSendTx, false); err != nil {
		return sdkerrors.Wrap(err, "send gasfree expected no error")
	}
	// Expect failure for both msg tx
	cached, _ = gasfreeSendCtx.CacheContext()
	if _, err := suite.anteHandler(cached, bothTx, false); !strings.Contains(err.Error(), expErrorStr) {
		return sdkerrors.Wrap(err, "send gasfree sent send and microtx - expected error")
	}

	// --- No Gasfree Msg types ---
	// Expect failure for the Microtx tx
	cached, _ = noGasfreeCtx.CacheContext()
	if _, err := suite.anteHandler(cached, msgMicrotxTx, false); !strings.Contains(err.Error(), expErrorStr) {
		return sdkerrors.Wrap(err, "no gasfree msgs sent microtx - expected error")
	}
	// Expect failure for the Send tx
	cached, _ = noGasfreeCtx.CacheContext()
	if _, err := suite.anteHandler(cached, msgSendTx, false); !strings.Contains(err.Error(), expErrorStr) {
		return sdkerrors.Wrap(err, "no gasfree msgs sent send - expected error")
	}
	// Expect failure for both msg tx
	cached, _ = noGasfreeCtx.CacheContext()
	if _, err := suite.anteHandler(cached, bothTx, false); !strings.Contains(err.Error(), expErrorStr) {
		return sdkerrors.Wrap(err, "no gasfree msgs sent send and microtx - expected error")
	}

	// --- Send and Microtx are gasfree Msg types ---
	// Expect success on all txs
	cached, _ = bothGasfreeCtx.CacheContext()
	if _, err := suite.anteHandler(cached, msgMicrotxTx, false); err != nil {
		return sdkerrors.Wrap(err, "send + microtx gasfree expected no error")
	}
	cached, _ = bothGasfreeCtx.CacheContext()
	if _, err := suite.anteHandler(cached, msgSendTx, false); err != nil {
		return sdkerrors.Wrap(err, "send + microtx gasfree expected no error")
	}
	cached, _ = bothGasfreeCtx.CacheContext()
	if _, err := suite.anteHandler(cached, bothTx, false); err != nil {
		return sdkerrors.Wrap(err, "send + microtx gasfree expected no error")
	}
	return nil

}

// Checks that the MempoolFee antedecorator is bypassed for applicable txs
func (suite *AnteTestSuite) TestCosmosAnteHandlerMempoolFeeBypass() {
	suite.SetupTest()
	suite.ctx = suite.ctx.WithIsCheckTx(true) // use checkTx true to trigger mempool fee decorator
	suite.ctx = suite.ctx.WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(altheaconfig.BaseDenom, sdk.NewInt(100))))
	privKey := suite.NewCosmosPrivkey()
	addr := sdk.AccAddress(privKey.PubKey().Address().Bytes())
	suite.FundAccount(suite.ctx, addr, big.NewInt(10000000000))
	denom := altheaconfig.BaseDenom
	amount := sdk.NewInt(200000)

	msgSendMsg := suite.CreateTestCosmosMsgSend(sdk.NewInt(0), denom, amount, addr, addr)
	msgSendMsg.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(altheaconfig.BaseDenom, sdk.NewInt(1))))
	msgSendTx := suite.CreateSignedCosmosTx(suite.ctx, msgSendMsg, privKey)
	msgMicrotxTx := suite.CreateSignedCosmosTx(suite.ctx, suite.CreateTestCosmosMsgMicrotx(sdk.NewInt(0), denom, amount, addr, addr), privKey)
	bothTx := suite.CreateSignedCosmosTx(suite.ctx, suite.CreateTestCosmosMsgMicrotxMsgSend(sdk.NewInt(0), denom, amount, addr, addr), privKey)

	gasfreeMicrotxCtx, _ := suite.ctx.CacheContext()
	gasfreeSendCtx, _ := suite.ctx.CacheContext()
	// nolint: exhaustruct
	suite.app.GasfreeKeeper.SetGasFreeMessageTypes(gasfreeSendCtx, []string{sdk.MsgTypeURL(&banktypes.MsgSend{})})
	noGasfreeCtx, _ := suite.ctx.CacheContext()
	suite.app.GasfreeKeeper.SetGasFreeMessageTypes(noGasfreeCtx, []string{})
	bothGasfreeCtx, _ := suite.ctx.CacheContext()
	// nolint: exhaustruct
	suite.app.GasfreeKeeper.SetGasFreeMessageTypes(bothGasfreeCtx, []string{sdk.MsgTypeURL(&microtxtypes.MsgMicrotx{}), sdk.MsgTypeURL(&banktypes.MsgSend{})})

	// Expect the error from the mempool fee decorator to contain something like "insufficient fees; got: x required: provided fee < minimum global feey"
	suite.Require().NoError(runBypassTest(suite, "insufficient fees; got:", gasfreeMicrotxCtx, gasfreeSendCtx, noGasfreeCtx, bothGasfreeCtx, msgMicrotxTx, msgSendTx, bothTx))
}

// Checks that the MinGasPrices antedecorator is bypassed for applicable txs
// nolint: dupl
func (suite *AnteTestSuite) TestCosmosAnteHandlerMinGasPricesBypass() {
	suite.SetupTest()
	suite.ctx = suite.ctx.WithIsCheckTx(false) // use checkTx false to avoid triggering mempool fee decorator
	feemarketParams := suite.app.FeemarketKeeper.GetParams(suite.ctx)
	feemarketParams.MinGasPrice = sdk.NewDec(100)
	suite.app.FeemarketKeeper.SetParams(suite.ctx, feemarketParams) // Set the min gas price to trigger failure in the MinGasPricesDecorator
	privKey := suite.NewCosmosPrivkey()
	addr := sdk.AccAddress(privKey.PubKey().Address().Bytes())
	suite.FundAccount(suite.ctx, addr, big.NewInt(10000000000))
	denom := altheaconfig.BaseDenom
	amount := sdk.NewInt(200000)

	msgSendTx := suite.CreateSignedCosmosTx(suite.ctx, suite.CreateTestCosmosMsgSend(sdk.NewInt(10), denom, amount, addr, addr), privKey)
	msgMicrotxTx := suite.CreateSignedCosmosTx(suite.ctx, suite.CreateTestCosmosMsgMicrotx(sdk.NewInt(10), denom, amount, addr, addr), privKey)
	bothTx := suite.CreateSignedCosmosTx(suite.ctx, suite.CreateTestCosmosMsgMicrotxMsgSend(sdk.NewInt(10), denom, amount, addr, addr), privKey)

	gasfreeMicrotxCtx, _ := suite.ctx.CacheContext()
	gasfreeSendCtx, _ := suite.ctx.CacheContext()
	// nolint: exhaustruct
	suite.app.GasfreeKeeper.SetGasFreeMessageTypes(gasfreeSendCtx, []string{sdk.MsgTypeURL(&banktypes.MsgSend{})})
	noGasfreeCtx, _ := suite.ctx.CacheContext()
	suite.app.GasfreeKeeper.SetGasFreeMessageTypes(noGasfreeCtx, []string{})
	bothGasfreeCtx, _ := suite.ctx.CacheContext()
	// nolint: exhaustruct
	suite.app.GasfreeKeeper.SetGasFreeMessageTypes(bothGasfreeCtx, []string{sdk.MsgTypeURL(&microtxtypes.MsgMicrotx{}), sdk.MsgTypeURL(&banktypes.MsgSend{})})

	// Expect the error from the MinGasPricesDecorator to contain something like "provided fee < minimum global fee (x < y). Please increase the gas price."
	suite.Require().NoError(runBypassTest(suite, "provided fee < minimum global fee", gasfreeMicrotxCtx, gasfreeSendCtx, noGasfreeCtx, bothGasfreeCtx, msgMicrotxTx, msgSendTx, bothTx))
}

// Checks that the DeductFee antedecorator is bypassed for applicable txs
// nolint: dupl
func (suite *AnteTestSuite) TestCosmosAnteHandlerDeductFeeBypass() {
	suite.SetupTest()
	suite.ctx = suite.ctx.WithIsCheckTx(false) // use checkTx false to avoid triggering mempool fee decorator
	feemarketParams := suite.app.FeemarketKeeper.GetParams(suite.ctx)
	feemarketParams.MinGasPrice = sdk.NewDec(0)
	suite.app.FeemarketKeeper.SetParams(suite.ctx, feemarketParams) // Set the min gas price to avoid failure from MinGasPricesDecorator
	privKey := suite.NewCosmosPrivkey()
	addr := sdk.AccAddress(privKey.PubKey().Address().Bytes())
	suite.FundAccount(suite.ctx, addr, big.NewInt(1))
	denom := altheaconfig.BaseDenom
	amount := sdk.NewInt(2)

	msgSendTx := suite.CreateSignedCosmosTx(suite.ctx, suite.CreateTestCosmosMsgSend(sdk.NewInt(2), denom, amount, addr, addr), privKey)
	msgMicrotxTx := suite.CreateSignedCosmosTx(suite.ctx, suite.CreateTestCosmosMsgMicrotx(sdk.NewInt(2), denom, amount, addr, addr), privKey)
	bothTx := suite.CreateSignedCosmosTx(suite.ctx, suite.CreateTestCosmosMsgMicrotxMsgSend(sdk.NewInt(2), denom, amount, addr, addr), privKey)

	gasfreeMicrotxCtx, _ := suite.ctx.CacheContext()
	gasfreeSendCtx, _ := suite.ctx.CacheContext()
	// nolint: exhaustruct
	suite.app.GasfreeKeeper.SetGasFreeMessageTypes(gasfreeSendCtx, []string{sdk.MsgTypeURL(&banktypes.MsgSend{})})
	noGasfreeCtx, _ := suite.ctx.CacheContext()
	suite.app.GasfreeKeeper.SetGasFreeMessageTypes(noGasfreeCtx, []string{})
	bothGasfreeCtx, _ := suite.ctx.CacheContext()
	// nolint: exhaustruct
	suite.app.GasfreeKeeper.SetGasFreeMessageTypes(bothGasfreeCtx, []string{sdk.MsgTypeURL(&microtxtypes.MsgMicrotx{}), sdk.MsgTypeURL(&banktypes.MsgSend{})})

	// Expect the error to be a wrapped insufficient funds error
	suite.Require().NoError(runBypassTest(suite, "insufficient funds: insufficient funds", gasfreeMicrotxCtx, gasfreeSendCtx, noGasfreeCtx, bothGasfreeCtx, msgMicrotxTx, msgSendTx, bothTx))
}
