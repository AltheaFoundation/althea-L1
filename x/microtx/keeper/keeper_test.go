package keeper_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	"github.com/tendermint/tendermint/version"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/app"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/encoding"
	"github.com/evmos/ethermint/tests"
	evm "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"

	althea "github.com/AltheaFoundation/althea-L1/app"
	erc20types "github.com/AltheaFoundation/althea-L1/x/erc20/types"
	microtxtypes "github.com/AltheaFoundation/althea-L1/x/microtx/types"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx            sdk.Context
	app            *althea.AltheaApp
	queryClientEvm evm.QueryClient
	queryClient    erc20types.QueryClient
	address        common.Address
	clientCtx      client.Context
	ethSigner      ethtypes.Signer
	signer         keyring.Signer
}

var s *KeeperTestSuite

func TestKeeperTestSuite(t *testing.T) {
	s = new(KeeperTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Suite")
}

// Test helpers
func (suite *KeeperTestSuite) DoSetupTest(t require.TestingT) {
	checkTx := false

	// account key
	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	suite.address = common.BytesToAddress(priv.PubKey().Address().Bytes())
	suite.signer = tests.NewSigner(priv)

	// init app
	suite.app = althea.NewSetup(checkTx, func(aa *althea.AltheaApp, gs simapp.GenesisState) simapp.GenesisState {
		// setup feemarketGenesis params
		feemarketGenesis := feemarkettypes.DefaultGenesisState()
		feemarketGenesis.Params.EnableHeight = 1
		feemarketGenesis.Params.NoBaseFee = false
		feemarketGenesis.Params.BaseFee = sdk.NewInt(1)

		gs[feemarkettypes.ModuleName] = aa.AppCodec().MustMarshalJSON(feemarketGenesis)

		microtxGenesis := microtxtypes.DefaultGenesisState()
		microtxGenesis.PreviousProposer = sdk.AccAddress(althea.ValidatorPubKey.Address().Bytes()).String()

		gs[microtxtypes.ModuleName] = aa.AppCodec().MustMarshalJSON(microtxGenesis)
		return gs
	})

	suite.ctx = suite.app.BaseApp.NewContext(checkTx, tmproto.Header{
		Height:          1,
		ChainID:         "althea_7357-1",
		Time:            time.Now().UTC(),
		ProposerAddress: althea.ValidatorPubKey.Address().Bytes(),

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

	queryHelperEvm := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	evm.RegisterQueryServer(queryHelperEvm, suite.app.EvmKeeper)
	suite.queryClientEvm = evm.NewQueryClient(queryHelperEvm)

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	erc20types.RegisterQueryServer(queryHelper, suite.app.Erc20Keeper)
	suite.queryClient = erc20types.NewQueryClient(queryHelper)

	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	suite.clientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)
	suite.ethSigner = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())

	suite.app.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{
		ChainID:            "althea_7357-1",
		Height:             suite.app.LastBlockHeight() + 1,
		AppHash:            suite.app.LastCommitID().Hash,
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
	}})
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.DoSetupTest(suite.T())
}
