package gasfree_test

import (
	"testing"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"
	"github.com/tendermint/tendermint/version"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/tests"
	"github.com/evmos/ethermint/x/evm/statedb"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"

	althea "github.com/AltheaFoundation/althea-L1/app"
	altheacfg "github.com/AltheaFoundation/althea-L1/config"
	microtxtypes "github.com/AltheaFoundation/althea-L1/x/microtx/types"
)

type GasfreeTestSuite struct {
	suite.Suite

	ctx sdk.Context
	app *althea.AltheaApp

	signer           keyring.Signer
	ethSigner        ethtypes.Signer
	from             common.Address
	mintFeeCollector bool
}

// / DoSetupTest setup test environment, it uses `require.TestingT` to support both `testing.T` and `testing.B`.
func (suite *GasfreeTestSuite) DoSetupTest(t require.TestingT) {
	checkTx := false

	// account key
	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	suite.signer = tests.NewSigner(priv)

	// init app
	suite.app = althea.NewSetup(checkTx, func(aa *althea.AltheaApp, gs simapp.GenesisState) simapp.GenesisState {
		// setup feemarketGenesis params
		feemarketGenesis := feemarkettypes.DefaultGenesisState()
		feemarketGenesis.Params.EnableHeight = 1
		feemarketGenesis.Params.NoBaseFee = false
		feemarketGenesis.Params.BaseFee = sdk.NewInt(1)

		gs[feemarkettypes.ModuleName] = aa.AppCodec().MustMarshalJSON(feemarketGenesis)

		if suite.mintFeeCollector {
			// mint some coin to fee collector
			coins := sdk.NewCoins(sdk.NewCoin(altheacfg.BaseDenom, sdk.NewInt(int64(params.TxGas)-1)))
			balances := []banktypes.Balance{
				{
					Address: suite.app.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName).String(),
					Coins:   coins,
				},
			}
			// update total supply
			var bankGenesis banktypes.GenesisState
			aa.AppCodec().MustUnmarshalJSON(gs[banktypes.ModuleName], &bankGenesis)
			bankGenesis.Balances = append(bankGenesis.Balances, balances...)
			bankGenesis.Supply = bankGenesis.Supply.Add(coins...)
			gs[banktypes.ModuleName] = suite.app.AppCodec().MustMarshalJSON(&bankGenesis)

		}
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

	suite.app.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{
		ChainID:            "althea_7357-1",
		Height:             suite.app.LastBlockHeight() + 1,
		AppHash:            suite.app.LastCommitID().Hash,
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
	}})
}

// DefaultConsensusParams defines the default Tendermint consensus params used in
// EthermintApp testing.
// nolint: exhaustruct
var DefaultConsensusParams = &abci.ConsensusParams{
	Block: &abci.BlockParams{
		MaxBytes: 200000,
		MaxGas:   -1, // no limit
	},
	Evidence: &tmproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &tmproto.ValidatorParams{
		PubKeyTypes: []string{
			tmtypes.ABCIPubKeyTypeEd25519,
		},
	},
}

func (suite *GasfreeTestSuite) SetupTest() {
	suite.DoSetupTest(suite.T())
}

func (suite *GasfreeTestSuite) SignTx(tx *evmtypes.MsgEthereumTx) {
	tx.From = suite.from.String()
	err := tx.Sign(suite.ethSigner, suite.signer)
	suite.Require().NoError(err)
}

func (suite *GasfreeTestSuite) StateDB() *statedb.StateDB {
	return statedb.New(suite.ctx, suite.app.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(suite.ctx.HeaderHash().Bytes())))
}

func TestGasfreeTestSuite(t *testing.T) {
	// nolint: exhaustruct
	suite.Run(t, &GasfreeTestSuite{})
}
