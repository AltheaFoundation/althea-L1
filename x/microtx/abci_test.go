package microtx_test

import (
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	ccrypto "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmodule "github.com/cosmos/cosmos-sdk/types/module"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	erc20types "github.com/AltheaFoundation/althea-L1/x/erc20/types"
	"github.com/AltheaFoundation/althea-L1/x/microtx/types"
)

var (
	// ConsPrivKeys generate ed25519 ConsPrivKeys to be used for validator operator keys
	ConsPrivKeys = []ccrypto.PrivKey{
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
	}

	// ConsPubKeys holds the consensus public keys to be used for validator operator keys
	ConsPubKeys = []ccrypto.PubKey{
		ConsPrivKeys[0].PubKey(),
		ConsPrivKeys[1].PubKey(),
		ConsPrivKeys[2].PubKey(),
		ConsPrivKeys[3].PubKey(),
		ConsPrivKeys[4].PubKey(),
		ConsPrivKeys[5].PubKey(),
	}

	ConsAddrs = []sdk.ValAddress{
		sdk.ValAddress(ConsPubKeys[0].Address()),
		sdk.ValAddress(ConsPubKeys[1].Address()),
		sdk.ValAddress(ConsPubKeys[2].Address()),
		sdk.ValAddress(ConsPubKeys[3].Address()),
		sdk.ValAddress(ConsPubKeys[4].Address()),
		sdk.ValAddress(ConsPubKeys[5].Address()),
	}

	// AccPrivKeys generate secp256k1 pubkeys to be used for account pub keys
	AccPrivKeys = []ccrypto.PrivKey{
		secp256k1.GenPrivKey(),
		secp256k1.GenPrivKey(),
		secp256k1.GenPrivKey(),
	}

	// AccPubKeys holds the pub keys for the account keys
	AccPubKeys = []ccrypto.PubKey{
		AccPrivKeys[0].PubKey(),
		AccPrivKeys[1].PubKey(),
		AccPrivKeys[2].PubKey(),
	}

	// AccAddrs holds the sdk.AccAddresses
	AccAddrs = []sdk.AccAddress{
		sdk.AccAddress(AccPubKeys[0].Address()),
		sdk.AccAddress(AccPubKeys[1].Address()),
		sdk.AccAddress(AccPubKeys[2].Address()),
	}
)

type validator struct {
	addr   sdk.ValAddress
	pubkey ccrypto.PubKey
	votes  []abci.VoteInfo
}

func (suite *MicrotxTestSuite) TestBeginBlocker() {
	var (
		ONE_ETH, _ = sdk.NewIntFromString("1000000000000000000")
	)

	var (
		valBalances              = []sdk.Coin{}
		txs                      = []types.MsgMicrotx{}
		totalValidators          = len(ConsPrivKeys)
		power                    = int64(100 / totalValidators)
		valTokens                = sdk.TokensFromConsensusPower(50, sdk.DefaultPowerReduction)
		valCoin                  = sdk.NewCoin("aalthea", valTokens)
		validatorCommissionRates = stakingtypes.NewCommissionRates(sdk.OneDec(), sdk.OneDec(), sdk.OneDec())
	)
	ctx := suite.ctx
	app := suite.app

	// Register a "balthea" token as an ERC20 token pair
	pair := erc20types.TokenPair{Erc20Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Denom: "balthea", Enabled: true, ContractOwner: erc20types.OWNER_MODULE}
	suite.app.Erc20Keeper.SetTokenPair(ctx, pair)
	suite.app.Erc20Keeper.SetDenomMap(ctx, pair.Denom, pair.GetID())
	suite.app.Erc20Keeper.SetERC20Map(ctx, common.HexToAddress(pair.Erc20Address), pair.GetID())
	// Set the previous proposer to avoid panics
	suite.app.MicrotxKeeper.SetPreviousProposerConsAddr(ctx, sdk.GetConsAddress(ConsPubKeys[0]))

	// Disable inflation to keep validator balances predictable
	mParams := suite.app.MintKeeper.GetParams(ctx)
	mParams.InflationMax = sdk.ZeroDec()
	mParams.InflationMin = sdk.ZeroDec()
	suite.app.MintKeeper.SetParams(ctx, mParams)

	for _, accAddr := range AccAddrs {
		// Give all acc addrs 1000 aalthea
		suite.InitAccountWithCoins(accAddr, sdk.NewCoins(sdk.NewCoin("balthea", ONE_ETH.Mul(sdk.NewInt(1000)))))
	}

	// create validators
	validators := make([]validator, totalValidators-1)
	for i := range totalValidators - 1 {
		suite.InitAccountWithCoins(sdk.AccAddress(ConsAddrs[i]), sdk.NewCoins(valCoin))
		validators[i].addr = ConsAddrs[i]
		validators[i].pubkey = ed25519.GenPrivKey().PubKey()
		validators[i].votes = make([]abci.VoteInfo, totalValidators)
		suite.CreateValidatorWithValPower(ctx, validators[i].addr, validators[i].pubkey, valCoin, validatorCommissionRates)
	}
	// Create the initial block
	app.EndBlock(abci.RequestEndBlock{})
	require.NotEmpty(suite.T(), app.Commit())
	suite.ctx = suite.ctx.WithBlockHeight(1)

	// verify validators lists
	require.Len(suite.T(), app.StakingKeeper.GetAllValidators(ctx), totalValidators)
	for i, val := range validators {
		// verify all validator exists
		require.NotNil(suite.T(), app.StakingKeeper.ValidatorByConsAddr(ctx, sdk.GetConsAddress(val.pubkey)))

		// populate last commit info
		voteInfos := []abci.VoteInfo{}
		for _, val2 := range validators {
			voteInfos = append(voteInfos, abci.VoteInfo{
				Validator: abci.Validator{
					Address: sdk.GetConsAddress(val2.pubkey),
					Power:   power,
				},
				SignedLastBlock: true,
			})
		}
		validators[i].votes = voteInfos
	}

	caseSetup := func(ctx sdk.Context, microtxs []types.MsgMicrotx, params types.Params) {
		suite.app.MicrotxKeeper.SetParams(ctx, params)
		for _, mtx := range microtxs {
			err := suite.app.MicrotxKeeper.Microtx(ctx, sdk.MustAccAddressFromBech32(mtx.Sender), sdk.MustAccAddressFromBech32(mtx.Receiver), mtx.Amount)
			suite.Require().NoError(err)
		}
	}

	defaultParams := types.DefaultParams()
	testCases := []struct {
		name        string
		params      types.Params
		malleate    func(ctx sdk.Context)
		proposerIdx int
		deferFunc   func(ctx sdk.Context)
		expPass     bool
	}{
		{
			name:   "no microtxs",
			params: *defaultParams,
			malleate: func(ctx sdk.Context) {
				txs = []types.MsgMicrotx{}
				valBalances = suite.snapshotValidatorBalances(ctx, validators)
			},
			proposerIdx: 1,
			deferFunc: func(ctx sdk.Context) {
				val0Balance := suite.app.BankKeeper.GetBalance(ctx, sdk.AccAddress(ConsAddrs[0]), "balthea")
				suite.Require().Equal(val0Balance, sdk.NewCoin("balthea", sdk.NewInt(0)))
			},
			expPass: true,
		},
		{
			name: "one microtx, no proposer rewards",
			params: types.Params{
				BaseProposerReward:    sdk.ZeroDec(),
				BonusProposerReward:   sdk.ZeroDec(),
				MicrotxFeeBasisPoints: defaultParams.MicrotxFeeBasisPoints,
			},
			malleate: func(ctx sdk.Context) {
				txs = []types.MsgMicrotx{{Sender: AccAddrs[0].String(), Receiver: AccAddrs[1].String(), Amount: sdk.NewCoin("balthea", ONE_ETH)}}
				valBalances = suite.snapshotValidatorBalances(ctx, validators)
			},
			proposerIdx: 1,
			deferFunc: func(ctx sdk.Context) {
				newBalances := suite.snapshotValidatorBalances(ctx, validators)
				// We expect fees to be 1/10th the amount transferred, and divided evenly among all validators less the community tax
				expectedIncrease := sdk.OneDec().Quo(sdk.NewDec(10)).Mul(sdk.OneDec().Sub(suite.app.DistrKeeper.GetCommunityTax(ctx))).Quo(sdk.NewDec(int64(totalValidators - 1))).MulInt(ONE_ETH).TruncateInt()
				val0Balances := newBalances[0]
				for i, balances := range newBalances {
					// Check that no validator was given higher rewards than any other
					suite.Require().Equal(val0Balances, balances)
					// Check that all validators have their balances improved or kept equal
					suite.Require().True(valBalances[i].IsLT(balances))
					suite.Require().Equal(valBalances[i].Add(sdk.NewCoin("balthea", expectedIncrease)), balances)
				}
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx := suite.ctx
			tc.malleate(ctx)
			caseSetup(ctx, txs, tc.params)
			app.MM.Modules[types.ModuleName].(sdkmodule.BeginBlockAppModule).BeginBlock(ctx, abci.RequestBeginBlock{
				Header:         tmproto.Header{ChainID: ctx.ChainID(), Height: app.LastBlockHeight() + 1, ProposerAddress: sdk.GetConsAddress(validators[tc.proposerIdx].pubkey)},
				LastCommitInfo: abci.LastCommitInfo{Votes: validators[tc.proposerIdx].votes},
			})

			suite.withdrawValidatorRewards(ctx, validators)
			tc.deferFunc(ctx)
		})
	}

}

func (suite *MicrotxTestSuite) snapshotValidatorBalances(ctx sdk.Context, validators []validator) []sdk.Coin {
	valBalances := make([]sdk.Coin, len(validators))
	for i, val := range validators {
		valBalances[i] = suite.app.BankKeeper.GetBalance(ctx, sdk.AccAddress(val.addr), "balthea")
	}
	return valBalances
}

func (suite *MicrotxTestSuite) withdrawValidatorRewards(ctx sdk.Context, validators []validator) {
	for _, val := range validators {
		suite.app.DistrKeeper.WithdrawValidatorCommission(ctx, val.addr)
	}
}
