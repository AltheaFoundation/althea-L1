package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	"github.com/tendermint/tendermint/version"

	evmtypes "github.com/evmos/ethermint/x/evm/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	althea "github.com/AltheaFoundation/althea-L1/app"
	altheaconfig "github.com/AltheaFoundation/althea-L1/config"
	"github.com/AltheaFoundation/althea-L1/x/onboarding/types"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx sdk.Context

	app         *althea.AltheaApp
	queryClient types.QueryClient
}

func (suite *KeeperTestSuite) SetupTest() {
	// consensus key

	suite.app = althea.NewSetup(false, func(app *althea.AltheaApp, genesis simapp.GenesisState) simapp.GenesisState {
		evmGenesis := evmtypes.DefaultGenesisState()
		evmGenesis.Params.EvmDenom = altheaconfig.BaseDenom

		genesis[evmtypes.ModuleName] = app.AppCodec().MustMarshalJSON(evmGenesis)

		authGenesis := authtypes.DefaultGenesisState()

		genesis[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)

		return genesis
	})
	consAddress := sdk.ConsAddress(althea.ValidatorPubKey.Address())
	// nolint: exhaustruct
	suite.ctx = suite.app.BaseApp.NewContext(false, tmproto.Header{
		Height:          1,
		ChainID:         altheaconfig.DefaultChainID(),
		Time:            time.Now().UTC(),
		ProposerAddress: consAddress.Bytes(),

		// nolint: exhaustruct
		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		AppHash:            tmhash.Sum([]byte("app")),
		DataHash:           tmhash.Sum([]byte("data")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
	})

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.app.OnboardingKeeper)
	suite.queryClient = types.NewQueryClient(queryHelper)

	vals := suite.app.StakingKeeper.GetValidatorSet()
	var val stakingtypes.ValidatorI
	vals.IterateValidators(suite.ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		val = validator
		return true
	})
	cAddr, err := val.GetConsAddr()
	require.NoError(suite.T(), err)
	cfg, err := suite.app.EvmKeeper.EVMConfig(suite.ctx, cAddr, suite.app.EvmKeeper.ChainID())
	require.NoError(suite.T(), err)
	cfg = cfg
}
func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
