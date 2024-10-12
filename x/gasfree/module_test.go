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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/tests"
	"github.com/evmos/ethermint/x/evm"
	"github.com/evmos/ethermint/x/evm/statedb"
	evmtypes "github.com/evmos/ethermint/x/evm/types"

	althea "github.com/AltheaFoundation/althea-L1/app"
	altheaconfig "github.com/AltheaFoundation/althea-L1/config"
)

type GasfreeTestSuite struct {
	suite.Suite

	ctx     sdk.Context
	handler sdk.Handler
	app     *althea.AltheaApp

	signer    keyring.Signer
	ethSigner ethtypes.Signer
	from      common.Address
}

// / DoSetupTest setup test environment, it uses `require.TestingT` to support both `testing.T` and `testing.B`.
func (suite *GasfreeTestSuite) DoSetupTest(t require.TestingT) {
	checkTx := false

	// account key
	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	address := common.BytesToAddress(priv.PubKey().Address().Bytes())
	suite.signer = tests.NewSigner(priv)
	suite.from = address
	// consensus key
	priv, err = ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	consAddress := sdk.ConsAddress(priv.PubKey().Address())

	suite.app = Setup(checkTx, func(app *althea.AltheaApp, genesis simapp.GenesisState) simapp.GenesisState {
		evmGenesis := evmtypes.DefaultGenesisState()
		evmGenesis.Params.EvmDenom = altheaconfig.BaseDenom
		evmGenesis.Params.AllowUnprotectedTxs = false

		genesis[evmtypes.ModuleName] = app.AppCodec().MustMarshalJSON(evmGenesis)

		coins := sdk.NewCoins(sdk.NewCoin(altheaconfig.BaseDenom, sdk.NewInt(100000000000000)))
		genesisState := althea.ModuleBasics.DefaultGenesis(app.AppCodec())
		b32address := sdk.MustBech32ifyAddressBytes(sdk.GetConfig().GetBech32AccountAddrPrefix(), priv.PubKey().Address().Bytes())
		balances := []banktypes.Balance{
			{
				Address: b32address,
				Coins:   coins,
			},
			{
				Address: app.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName).String(),
				Coins:   coins,
			},
		}
		// Update total supply
		bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, sdk.NewCoins(sdk.NewCoin(altheaconfig.BaseDenom, sdk.NewInt(200000000000000))), []banktypes.Metadata{})
		genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)

		return genesis
	})

	// nolint: exhaustruct
	suite.ctx = suite.app.BaseApp.NewContext(checkTx, tmproto.Header{
		Height:          1,
		ChainID:         "althea_417834-1",
		Time:            time.Now().UTC(),
		ProposerAddress: consAddress.Bytes(),
		// nolint: exhaustruct
		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		// nolint: exhaustruct
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

	// queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	// evmtypes.RegisterQueryServer(queryHelper, suite.app.EvmKeeper)

	// acc := &ethermint.EthAccount{
	// 	BaseAccount: authtypes.NewBaseAccount(sdk.AccAddress(address.Bytes()), nil, 0, 0),
	// 	CodeHash:    common.BytesToHash(crypto.Keccak256(nil)).String(),
	// }

	// suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	// valAddr := sdk.ValAddress(address.Bytes())
	// // nolint: exhaustruct
	// validator, err := stakingtypes.NewValidator(valAddr, priv.PubKey(), stakingtypes.Description{})
	// require.NoError(t, err)

	// err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	// require.NoError(t, err)
	// err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	// require.NoError(t, err)
	// suite.app.StakingKeeper.SetValidator(suite.ctx, validator)

	suite.ethSigner = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())
	suite.handler = evm.NewHandler(suite.app.EvmKeeper)
}

// Setup initializes a new Althea app. A Nop logger is set in AltheaApp.
func Setup(isCheckTx bool, patchGenesis func(*althea.AltheaApp, simapp.GenesisState) simapp.GenesisState) *althea.AltheaApp {
	return althea.NewSetup(isCheckTx, patchGenesis)
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
