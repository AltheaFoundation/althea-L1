package microtx_test

import (
	"fmt"

	"github.com/AltheaFoundation/althea-L1/x/microtx/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	ccrypto "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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
	suite.SetupTest()

	var (
		txs                      = []types.MsgMicrotx{}
		totalValidators          = len(ConsPrivKeys)
		power                    = int64(100 / totalValidators)
		valTokens                = sdk.TokensFromConsensusPower(50, sdk.DefaultPowerReduction)
		valCoin                  = sdk.NewCoin("aalthea", valTokens)
		validatorCommissionRates = stakingtypes.NewCommissionRates(sdk.OneDec(), sdk.OneDec(), sdk.OneDec())
	)
	ctx := suite.ctx
	app := suite.app

	// create validators
	validators := make([]validator, totalValidators-1)
	for i := range validators {
		suite.InitAccountWithCoins(sdk.AccAddress(ConsAddrs[i]), sdk.NewCoins(valCoin))
		validators[i].addr = ConsAddrs[i]
		validators[i].pubkey = ed25519.GenPrivKey().PubKey()
		validators[i].votes = make([]abci.VoteInfo, totalValidators)
		suite.CreateValidatorWithValPower(ctx, validators[i].addr, validators[i].pubkey, valCoin, validatorCommissionRates)
	}
	// Create the initial block
	app.EndBlock(abci.RequestEndBlock{})
	require.NotEmpty(suite.T(), app.Commit())

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

	caseSetup := func(ctx sdk.Context, microtxs []types.MsgMicrotx) {
		for _, mtx := range microtxs {
			suite.app.MicrotxKeeper.Microtx(ctx, sdk.MustAccAddressFromBech32(mtx.Sender), sdk.MustAccAddressFromBech32(mtx.Receiver), mtx.Amount)
		}
	}

	testCases := []struct {
		name        string
		malleate    func()
		proposerIdx int
		deferFunc   func(ctx sdk.Context)
		expPass     bool
	}{
		{
			name: "no microtxs",
			malleate: func() {
				txs = []types.MsgMicrotx{}
			},
			proposerIdx: 1,
			deferFunc: func(ctx sdk.Context) {
				val0Balance := suite.app.BankKeeper.GetBalance(ctx, sdk.AccAddress(ConsAddrs[0]), "balthea")
				suite.Require().Equal(val0Balance, sdk.NewCoins())
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx, _ := suite.ctx.CacheContext()
			tc.malleate()
			caseSetup(ctx, txs)
			fmt.Println("Chain Id", ctx.ChainID())
			app.BeginBlock(abci.RequestBeginBlock{
				Header:         tmproto.Header{Height: app.LastBlockHeight() + 1, ProposerAddress: sdk.GetConsAddress(validators[tc.proposerIdx].pubkey)},
				LastCommitInfo: abci.LastCommitInfo{Votes: validators[tc.proposerIdx].votes},
			})
			require.NotEmpty(suite.T(), app.Commit())

			tc.deferFunc(ctx)
		})
	}

}
