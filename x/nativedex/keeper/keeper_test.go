package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	"github.com/tendermint/tendermint/version"

	althea "github.com/AltheaFoundation/althea-L1/app"
	"github.com/AltheaFoundation/althea-L1/x/nativedex/types"
)

type KeeperTestSuite struct {
	suite.Suite
	ctx            sdk.Context
	app            *althea.AltheaApp
	queryClientEvm evmtypes.QueryClient
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupTest() {
	// init app
	suite.app = althea.NewSetup(false, func(aa *althea.AltheaApp, gs simapp.GenesisState) simapp.GenesisState {
		// setup feemarketGenesis params
		feemarketGenesis := feemarkettypes.DefaultGenesisState()
		feemarketGenesis.Params.EnableHeight = 1
		feemarketGenesis.Params.NoBaseFee = false
		gs[feemarkettypes.ModuleName] = aa.AppCodec().MustMarshalJSON(feemarketGenesis)
		return gs
	})

	//nolint: exhaustruct
	suite.ctx = suite.app.BaseApp.NewContext(false, tmproto.Header{
		Height:          1,
		ChainID:         "althea_7357-1",
		ProposerAddress: althea.ValidatorPubKey.Address().Bytes(),

		//nolint: exhaustruct
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
	evmtypes.RegisterQueryServer(queryHelper, suite.app.EvmKeeper)
	suite.queryClientEvm = evmtypes.NewQueryClient(queryHelper)
}

// TestGetParamsIfSet_ParamsInitialized tests that GetParamsIfSet returns params when they are set
func (suite *KeeperTestSuite) TestGetParamsIfSet_ParamsInitialized() {
	// In this test, params are initialized by default genesis state
	params, err := suite.app.NativedexKeeper.GetParamsIfSet(suite.ctx)
	suite.Require().NoError(err, "GetParamsIfSet should succeed when params are initialized")

	// Verify we got actual params back
	suite.Require().NotNil(params)

	// The default params should have zero addresses
	zeroAddr := common.Address{}.String()
	suite.Require().Equal(zeroAddr, params.VerifiedNativeDexAddress, "Default should have zero address")
	suite.Require().Equal(zeroAddr, params.VerifiedCrocPolicyAddress, "Default should have zero address")
	suite.Require().Empty(params.WhitelistedContractAddresses, "Default should have no whitelisted contracts")
}

// TestGetParamsIfSet_ParamsSet tests that GetParamsIfSet returns the correct params after they're updated
func (suite *KeeperTestSuite) TestGetParamsIfSet_ParamsSet() {
	// Set specific params
	testDexAddr := "0xd263DC98dEc57828e26F69bA8687281BA5D052E0"
	testPolicyAddr := "0x14Ae279edb4D569BAFb98ff08299A0135Da6867a"
	testWhitelist := []string{"0x1111111111111111111111111111111111111111"}

	newParams := types.Params{
		VerifiedNativeDexAddress:     testDexAddr,
		VerifiedCrocPolicyAddress:    testPolicyAddr,
		WhitelistedContractAddresses: testWhitelist,
	}

	suite.app.NativedexKeeper.SetParams(suite.ctx, newParams)

	// Now retrieve them with GetParamsIfSet
	params, err := suite.app.NativedexKeeper.GetParamsIfSet(suite.ctx)
	suite.Require().NoError(err, "GetParamsIfSet should succeed after setting params")

	// Verify the returned params match what we set
	suite.Require().Equal(testDexAddr, params.VerifiedNativeDexAddress)
	suite.Require().Equal(testPolicyAddr, params.VerifiedCrocPolicyAddress)
	suite.Require().Equal(testWhitelist, params.WhitelistedContractAddresses)
}

// TestGetParams_ParamsInitialized tests the normal GetParams path
func (suite *KeeperTestSuite) TestGetParams_ParamsInitialized() {
	// GetParams should work fine when params are initialized
	params := suite.app.NativedexKeeper.GetParams(suite.ctx)

	// Verify we got actual params back
	zeroAddr := common.Address{}.String()
	suite.Require().Equal(zeroAddr, params.VerifiedNativeDexAddress)
}

// TestSetAndGetParams tests the full cycle of setting and getting params
func (suite *KeeperTestSuite) TestSetAndGetParams() {
	testDexAddr := "0xd263DC98dEc57828e26F69bA8687281BA5D052E0"
	testPolicyAddr := "0x14Ae279edb4D569BAFb98ff08299A0135Da6867a"
	testWhitelist := []string{
		"0x1111111111111111111111111111111111111111",
		"0x2222222222222222222222222222222222222222",
	}

	newParams := types.Params{
		VerifiedNativeDexAddress:     testDexAddr,
		VerifiedCrocPolicyAddress:    testPolicyAddr,
		WhitelistedContractAddresses: testWhitelist,
	}

	// Set params
	suite.app.NativedexKeeper.SetParams(suite.ctx, newParams)

	// Get with GetParams
	params1 := suite.app.NativedexKeeper.GetParams(suite.ctx)
	suite.Require().Equal(testDexAddr, params1.VerifiedNativeDexAddress)
	suite.Require().Equal(testPolicyAddr, params1.VerifiedCrocPolicyAddress)
	suite.Require().Equal(testWhitelist, params1.WhitelistedContractAddresses)

	// Get with GetParamsIfSet
	params2, err := suite.app.NativedexKeeper.GetParamsIfSet(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(testDexAddr, params2.VerifiedNativeDexAddress)
	suite.Require().Equal(testPolicyAddr, params2.VerifiedCrocPolicyAddress)
	suite.Require().Equal(testWhitelist, params2.WhitelistedContractAddresses)

	// Both methods should return identical params
	suite.Require().True(params1.Equal(params2), "GetParams and GetParamsIfSet should return identical params")
}
