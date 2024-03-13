package gasfree_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/althea-net/althea-L1/x/gasfree"
	"github.com/althea-net/althea-L1/x/gasfree/types"
	microtxtypes "github.com/althea-net/althea-L1/x/microtx/types"
)

// NewSelectiveBypassDecorator makes a test antehandler which indicates how many times it has been run
func NewBypassIndicatorDecorator() BypassIndicatorDecorator {
	var runs int = 0
	return BypassIndicatorDecorator{&runs}
}

// SelectiveBypassDecorator is a test antehandler which indicates how many times AnteHandle has been run
type BypassIndicatorDecorator struct {
	AnteHandlerRuns *int
}

// AnteHandle simply increments the anteHandlerRuns counter
func (bid BypassIndicatorDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	*bid.AnteHandlerRuns = *bid.AnteHandlerRuns + 1
	return next(ctx, tx, simulate)
}

func (suite *GasfreeTestSuite) TestSelectiveBypassAnteDecorator() {
	suite.SetupTest()

	var (
		bypassIndicator BypassIndicatorDecorator
		anteHandler     sdk.AnteHandler
		txs             []sdk.Tx
	)

	caseSetup := func() {
		bypassIndicator = NewBypassIndicatorDecorator()
		anteHandler = sdk.ChainAnteDecorators(gasfree.NewSelectiveBypassDecorator(*suite.app.GasfreeKeeper, bypassIndicator))
	}

	testCases := []struct {
		name      string
		malleate  func()
		deferFunc func()
		genState  *types.GenesisState
		expPass   bool
	}{
		{
			name: "empty no bypass",
			malleate: func() {
				tx := suite.app.EncodingConfig.TxConfig.NewTxBuilder().GetTx()
				txs = []sdk.Tx{tx}
			},
			deferFunc: func() {
				suite.Require().Equal(1, *bypassIndicator.AnteHandlerRuns)
			},
			genState: types.DefaultGenesisState(),
			expPass:  true,
		},
		{
			name: "single microtx bypass",
			malleate: func() {
				builder := suite.app.EncodingConfig.TxConfig.NewTxBuilder()
				builder.SetMsgs(&microtxtypes.MsgMicrotx{})
				tx := builder.GetTx()
				txs = []sdk.Tx{tx}
			},
			deferFunc: func() {
				suite.Require().Equal(0, *bypassIndicator.AnteHandlerRuns)
			},
			genState: types.DefaultGenesisState(),
			expPass:  true,
		},
		{
			name: "multi microtx bypass",
			malleate: func() {
				builder := suite.app.EncodingConfig.TxConfig.NewTxBuilder()
				builder.SetMsgs(&microtxtypes.MsgMicrotx{}, &microtxtypes.MsgMicrotx{}, &microtxtypes.MsgMicrotx{})
				tx := builder.GetTx()
				txs = []sdk.Tx{tx}
			},
			deferFunc: func() {
				suite.Require().Equal(0, *bypassIndicator.AnteHandlerRuns)
			},
			genState: types.DefaultGenesisState(),
			expPass:  true,
		},
		{
			name: "single microtx single send no bypass",
			malleate: func() {
				builder := suite.app.EncodingConfig.TxConfig.NewTxBuilder()
				builder.SetMsgs(&microtxtypes.MsgMicrotx{}, &banktypes.MsgSend{})
				tx := builder.GetTx()
				txs = []sdk.Tx{tx}
			},
			deferFunc: func() {
				suite.Require().Equal(1, *bypassIndicator.AnteHandlerRuns)
			},
			genState: types.DefaultGenesisState(),
			expPass:  true,
		},
		{
			name: "multi microtx multi send no bypass",
			malleate: func() {
				builder := suite.app.EncodingConfig.TxConfig.NewTxBuilder()
				builder.SetMsgs(&microtxtypes.MsgMicrotx{}, &banktypes.MsgSend{}, &microtxtypes.MsgMicrotx{}, &banktypes.MsgSend{}, &microtxtypes.MsgMicrotx{}, &banktypes.MsgSend{})
				tx := builder.GetTx()
				txs = []sdk.Tx{tx}
			},
			deferFunc: func() {
				suite.Require().Equal(1, *bypassIndicator.AnteHandlerRuns)
			},
			genState: types.DefaultGenesisState(),
			expPass:  true,
		},
		{
			name: "single microtx single send gov bypass",
			malleate: func() {
				suite.app.GasfreeKeeper.SetGasFreeMessageTypes(suite.ctx, []string{sdk.MsgTypeURL(&microtxtypes.MsgMicrotx{}), sdk.MsgTypeURL(&banktypes.MsgSend{})})
				builder := suite.app.EncodingConfig.TxConfig.NewTxBuilder()
				builder.SetMsgs(&microtxtypes.MsgMicrotx{}, &banktypes.MsgSend{})
				tx := builder.GetTx()
				txs = []sdk.Tx{tx}
			},
			deferFunc: func() {
				suite.Require().Equal(0, *bypassIndicator.AnteHandlerRuns)
			},
			genState: types.DefaultGenesisState(),
			expPass:  true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx, _ := suite.ctx.CacheContext()
			caseSetup()
			tc.malleate()

			for _, tx := range txs {
				_, err := anteHandler(ctx, tx, false)

				if tc.expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			}
			tc.deferFunc()
		})
	}

}
