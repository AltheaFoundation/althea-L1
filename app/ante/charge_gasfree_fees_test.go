package ante_test

import (
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"

	altheaconfig "github.com/althea-net/althea-L1/config"
	microtxtypes "github.com/althea-net/althea-L1/x/microtx/types"
)

// --------------
// In this file we test that the ChargeGasfreeFeesDecorator only charges fees where applicable, so we have 4 checked scenarios:
//	1. There are no gasfree messages (No gasfree fees should be charged)
//  2. MsgMicrotx is the only gasfree message (expected, charge a % fee of the microtx transfer amount)
//  3. MsgSend is the only gasfree message (No gasfree fees should be charged because there is no logic for that)
//  4. MsgSend AND MsgMicrotx are the only gasfree messages (Only charge a % fee of the microtx transfer amount)
// Note that there is only one special gasfree fee deduction for MsgMicrotx so the tests do not make full assertions about MsgSend deductions

func runGasfreeTests(suite *AnteTestSuite, gasfreeMicrotxCtx, gasfreeSendCtx, noGasfreeCtx, bothGasfreeCtx sdk.Context, msgMicrotxTx, msgSendTx, bothTx sdk.Tx, addr sdk.AccAddress, testDenom string) error {
	baseDenom := altheaconfig.BaseDenom

	// --- Microtx is the only Gasfree Msg type ---
	// Only charge baseDenom fees for microtx
	if err := runGasfreeTest(suite, true, gasfreeMicrotxCtx, msgMicrotxTx, addr, baseDenom, testDenom); err != nil {
		return err
	}
	// Charge testDenom fees for the others
	if err := runGasfreeTest(suite, false, gasfreeMicrotxCtx, msgSendTx, addr, testDenom, "fake"); err != nil {
		return err
	}
	if err := runGasfreeTest(suite, false, gasfreeMicrotxCtx, bothTx, addr, testDenom, "fake"); err != nil {
		return err
	}

	// --- Send is the only Gasfree Msg type ---
	if err := runGasfreeTest(suite, false, gasfreeSendCtx, msgMicrotxTx, addr, testDenom, "fake"); err != nil {
		return err
	}
	// Here we only want to see that there is no error, no balances should change
	if err := runNoChangeGasfreeTest(suite, false, gasfreeSendCtx, msgMicrotxTx, addr, testDenom); err != nil {
		return err
	}

	if err := runGasfreeTest(suite, false, gasfreeSendCtx, bothTx, addr, testDenom, "fake"); err != nil {
		return err
	}

	// --- No Gasfree Msg types ---
	if err := runGasfreeTest(suite, false, noGasfreeCtx, msgMicrotxTx, addr, testDenom, "fake"); err != nil {
		return err
	}
	if err := runGasfreeTest(suite, false, noGasfreeCtx, msgSendTx, addr, testDenom, "fake"); err != nil {
		return err
	}
	if err := runGasfreeTest(suite, false, noGasfreeCtx, bothTx, addr, testDenom, "fake"); err != nil {
		return err
	}

	// --- Both Gasfree Msg types ---
	if err := runGasfreeTest(suite, true, bothGasfreeCtx, msgMicrotxTx, addr, baseDenom, testDenom); err != nil {
		return err
	}
	// Here we only want to see that there is no error, no balances should change
	if err := runNoChangeGasfreeTest(suite, true, bothGasfreeCtx, msgSendTx, addr, testDenom); err != nil {
		return err
	}
	if err := runGasfreeTest(suite, true, bothGasfreeCtx, bothTx, addr, baseDenom, testDenom); err != nil {
		return err
	}
	return nil
}

func runGasfreeTest(suite *AnteTestSuite, expPass bool, ctx sdk.Context, tx sdk.Tx, addr sdk.AccAddress, shouldChangeDenom, shouldNotChangeDenom string) error {
	shouldChangeBalance := suite.app.BankKeeper.GetBalance(ctx, addr, shouldChangeDenom)
	shouldNotChangeBalance := suite.app.BankKeeper.GetBalance(ctx, addr, shouldNotChangeDenom)

	cached, _ := ctx.CacheContext()
	if _, err := suite.anteHandler(cached, tx, false); err != nil && expPass {
		return err
	}

	if !expPass {
		return nil
	}

	shouldChangeBalanceAfter := suite.app.BankKeeper.GetBalance(cached, addr, shouldChangeDenom)
	shouldNotChangeBalanceAfter := suite.app.BankKeeper.GetBalance(cached, addr, shouldNotChangeDenom)

	var shouldChangeDiff, shouldNotChangeDiff sdk.Int
	if shouldChangeBalance.Amount.GT(shouldChangeBalanceAfter.Amount) {
		shouldChangeDiff = shouldChangeBalance.Amount.Sub(shouldChangeBalanceAfter.Amount)
	} else {
		shouldChangeDiff = shouldChangeBalanceAfter.Amount.Sub(shouldChangeBalance.Amount)
	}

	if shouldNotChangeBalance.Amount.GT(shouldNotChangeBalanceAfter.Amount) {
		shouldNotChangeDiff = shouldNotChangeBalance.Amount.Sub(shouldNotChangeBalanceAfter.Amount)
	} else {
		shouldNotChangeDiff = shouldNotChangeBalanceAfter.Amount.Sub(shouldNotChangeBalance.Amount)
	}

	if !shouldNotChangeDiff.IsZero() {
		return fmt.Errorf("expected no change in %s balance, got a difference of +/- %s", shouldNotChangeDenom, shouldNotChangeDiff)
	}

	if shouldChangeDiff.IsZero() {
		return fmt.Errorf("expected a change in %s balance, got no change", shouldChangeDenom)
	}
	return nil
}

func runNoChangeGasfreeTest(suite *AnteTestSuite, expPass bool, ctx sdk.Context, tx sdk.Tx, addr sdk.AccAddress, shouldNotChangeDenom string) error {
	shouldNotChangeBalance := suite.app.BankKeeper.GetBalance(ctx, addr, shouldNotChangeDenom)

	cached, _ := ctx.CacheContext()
	if _, err := suite.anteHandler(cached, tx, false); err != nil && expPass {
		return err
	}

	if !expPass {
		return nil
	}

	shouldNotChangeBalanceAfter := suite.app.BankKeeper.GetBalance(cached, addr, shouldNotChangeDenom)

	var shouldNotChangeDiff sdk.Int

	if shouldNotChangeBalance.Amount.GT(shouldNotChangeBalanceAfter.Amount) {
		shouldNotChangeDiff = shouldNotChangeBalance.Amount.Sub(shouldNotChangeBalanceAfter.Amount)
	} else {
		shouldNotChangeDiff = shouldNotChangeBalanceAfter.Amount.Sub(shouldNotChangeBalance.Amount)
	}

	if !shouldNotChangeDiff.IsZero() {
		return fmt.Errorf("expected no change in %s balance, got a difference of +/- %s", shouldNotChangeDenom, shouldNotChangeDiff)
	}

	return nil
}

// Checks that the gasfree fees decorator and only the gasfree fees decorator charges a fee where applicable
func (suite *AnteTestSuite) TestChargeGasfreeFeesDecorator() {
	suite.SetupTest()

	testDenom := "test"
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, evmtypes.ModuleName, sdk.NewCoins(sdk.NewCoin(testDenom, sdk.NewInt(10000000000)))))
	metadata := banktypes.Metadata{
		Base:        testDenom,
		Display:     testDenom,
		Name:        testDenom,
		Description: "Test coin",
		DenomUnits:  []*banktypes.DenomUnit{{Denom: testDenom, Exponent: 0, Aliases: []string{}}},
		Symbol:      strings.ToUpper(testDenom),
	}
	suite.app.BankKeeper.SetDenomMetaData(suite.ctx, metadata)

	suite.ctx = suite.ctx.WithIsCheckTx(true) // use checkTx true to trigger mempool fee decorator
	suite.ctx = suite.ctx.WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(altheaconfig.BaseDenom, sdk.NewInt(100)), sdk.NewDecCoin(testDenom, sdk.NewInt(100))))
	privKey := suite.NewCosmosPrivkey()
	addr := sdk.AccAddress(privKey.PubKey().Address().Bytes())
	holding := int64(10000000000)
	suite.FundAccount(suite.ctx, addr, big.NewInt(holding))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, evmtypes.ModuleName, addr, sdk.NewCoins(sdk.NewCoin(testDenom, sdk.NewInt(holding)))))
	denom := altheaconfig.BaseDenom
	amount := sdk.NewInt(200000)
	var feeAmount int64 = 100000000

	// Create messages whose fees are the testDenom, we only want to see the testDenom balance change on non-gasfree txs
	msgSendMsg := suite.CreateTestCosmosMsgSend(sdk.NewInt(0), denom, amount, addr, addr)
	msgSendMsg.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(testDenom, sdk.NewInt(feeAmount))))
	msgSendTx := suite.CreateSignedCosmosTx(suite.ctx, msgSendMsg, privKey)
	msgMicrotxMsg := suite.CreateTestCosmosMsgMicrotx(sdk.NewInt(0), denom, amount, addr, addr)
	msgMicrotxMsg.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(testDenom, sdk.NewInt(feeAmount))))
	msgMicrotxTx := suite.CreateSignedCosmosTx(suite.ctx, msgMicrotxMsg, privKey)
	bothMsgs := suite.CreateTestCosmosMsgMicrotxMsgSend(sdk.NewInt(0), denom, amount, addr, addr)
	bothMsgs.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(testDenom, sdk.NewInt(feeAmount))))
	bothTx := suite.CreateSignedCosmosTx(suite.ctx, bothMsgs, privKey)

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
	suite.Require().NoError(runGasfreeTests(suite, gasfreeMicrotxCtx, gasfreeSendCtx, noGasfreeCtx, bothGasfreeCtx, msgMicrotxTx, msgSendTx, bothTx, addr, testDenom))
}
