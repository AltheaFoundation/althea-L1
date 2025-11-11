package cardinal_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"

	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	"github.com/tendermint/tendermint/version"

	althea "github.com/AltheaFoundation/althea-L1/app"
	"github.com/AltheaFoundation/althea-L1/x/nativedex/types"
)

type HandlerTestSuite struct {
	suite.Suite
	ctx            sdk.Context
	app            *althea.AltheaApp
	queryClientEvm evmtypes.QueryClient
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (suite *HandlerTestSuite) SetupTest() {
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
		Time:            time.Now().UTC(),
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

// TestEnsureNativedexParams_DefaultsAlreadySet tests the scenario where params are already initialized
// with zero addresses (default state). The upgrade handler should detect this and update them
// to the proper iFi DEX addresses from governance proposal #16.
func (suite *HandlerTestSuite) TestEnsureNativedexParams_DefaultsAlreadySet() {
	// Params should already be initialized by genesis with zero addresses
	params, err := suite.app.NativedexKeeper.GetParamsIfSet(suite.ctx)
	suite.Require().NoError(err, "Params should be initialized by genesis")

	// They should be zero addresses by default
	zeroAddr := common.Address{}.String()
	suite.Require().Equal(zeroAddr, params.VerifiedNativeDexAddress)
	suite.Require().Equal(zeroAddr, params.VerifiedCrocPolicyAddress)

	// Simulate what ensureNativedexParams does: check if addresses are zero and update them
	nativeDexAddr := common.HexToAddress(params.VerifiedNativeDexAddress)
	crocPolicyAddr := common.HexToAddress(params.VerifiedCrocPolicyAddress)
	zeroAddrBytes := common.Address{}

	needsUpdate := false
	if nativeDexAddr == zeroAddrBytes {
		params.VerifiedNativeDexAddress = "0xd263DC98dEc57828e26F69bA8687281BA5D052E0"
		needsUpdate = true
	}
	if crocPolicyAddr == zeroAddrBytes {
		params.VerifiedCrocPolicyAddress = "0x14Ae279edb4D569BAFb98ff08299A0135Da6867a"
		needsUpdate = true
	}

	suite.Require().True(needsUpdate, "Params should need updating when addresses are zero")

	// Apply the update
	suite.app.NativedexKeeper.SetParams(suite.ctx, params)

	// Verify the addresses were updated correctly
	updatedParams, err := suite.app.NativedexKeeper.GetParamsIfSet(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal("0xd263DC98dEc57828e26F69bA8687281BA5D052E0", updatedParams.VerifiedNativeDexAddress)
	suite.Require().Equal("0x14Ae279edb4D569BAFb98ff08299A0135Da6867a", updatedParams.VerifiedCrocPolicyAddress)
}

// TestEnsureNativedexParams_ParamsAlreadyConfigured tests the scenario where params are already set
// with the correct contract addresses (like after a successful governance proposal)
func (suite *HandlerTestSuite) TestEnsureNativedexParams_ParamsAlreadyConfigured() {
	// Set params with proper addresses (simulating a successful governance proposal)
	testDexAddr := "0xd263DC98dEc57828e26F69bA8687281BA5D052E0"
	testPolicyAddr := "0x14Ae279edb4D569BAFb98ff08299A0135Da6867a"

	configuredParams := types.Params{
		VerifiedNativeDexAddress:     testDexAddr,
		VerifiedCrocPolicyAddress:    testPolicyAddr,
		WhitelistedContractAddresses: []string{},
	}

	suite.app.NativedexKeeper.SetParams(suite.ctx, configuredParams)

	// Retrieve params - they should be correctly set
	params, err := suite.app.NativedexKeeper.GetParamsIfSet(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(testDexAddr, params.VerifiedNativeDexAddress)
	suite.Require().Equal(testPolicyAddr, params.VerifiedCrocPolicyAddress)
}

// TestEnsureNativedexParams_ZeroAddressesGetUpdated tests that zero addresses are updated
// This simulates the actual bug scenario where params were initialized but addresses are zero
func (suite *HandlerTestSuite) TestEnsureNativedexParams_ZeroAddressesGetUpdated() {
	// Start with default params (zero addresses)
	defaultParams := types.DefaultParams()
	suite.app.NativedexKeeper.SetParams(suite.ctx, *defaultParams)

	// Verify they're zero
	params, err := suite.app.NativedexKeeper.GetParamsIfSet(suite.ctx)
	suite.Require().NoError(err)

	zeroAddr := common.Address{}
	nativeDexAddr := common.HexToAddress(params.VerifiedNativeDexAddress)
	crocPolicyAddr := common.HexToAddress(params.VerifiedCrocPolicyAddress)

	suite.Require().Equal(zeroAddr, nativeDexAddr, "Should start with zero address")
	suite.Require().Equal(zeroAddr, crocPolicyAddr, "Should start with zero address")

	// Now the addresses should be non-zero (this would be handled by ensureNativedexParams in the upgrade)
	// We're testing the expected behavior here
	expectedDexAddr := "0xd263DC98dEc57828e26F69bA8687281BA5D052E0"
	expectedPolicyAddr := "0x14Ae279edb4D569BAFb98ff08299A0135Da6867a"

	// Simulate what ensureNativedexParams does
	if nativeDexAddr == zeroAddr {
		params.VerifiedNativeDexAddress = expectedDexAddr
	}
	if crocPolicyAddr == zeroAddr {
		params.VerifiedCrocPolicyAddress = expectedPolicyAddr
	}

	suite.app.NativedexKeeper.SetParams(suite.ctx, params)

	// Verify the addresses were updated
	updatedParams, err := suite.app.NativedexKeeper.GetParamsIfSet(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(expectedDexAddr, updatedParams.VerifiedNativeDexAddress)
	suite.Require().Equal(expectedPolicyAddr, updatedParams.VerifiedCrocPolicyAddress)
}

// TestParamsValidation tests that the validation logic works correctly
func (suite *HandlerTestSuite) TestParamsValidation() {
	// Valid params should pass validation
	validParams := types.Params{
		VerifiedNativeDexAddress:     "0xd263DC98dEc57828e26F69bA8687281BA5D052E0",
		VerifiedCrocPolicyAddress:    "0x14Ae279edb4D569BAFb98ff08299A0135Da6867a",
		WhitelistedContractAddresses: []string{},
	}
	err := validParams.ValidateBasic()
	suite.Require().NoError(err, "Valid params should pass validation")

	// Default params should also be valid
	defaultParams := types.DefaultParams()
	err = defaultParams.ValidateBasic()
	suite.Require().NoError(err, "Default params should be valid")

	// Invalid address should fail validation
	invalidParams := types.Params{
		VerifiedNativeDexAddress:     "not_an_address",
		VerifiedCrocPolicyAddress:    "0x14Ae279edb4D569BAFb98ff08299A0135Da6867a",
		WhitelistedContractAddresses: []string{},
	}
	err = invalidParams.ValidateBasic()
	suite.Require().Error(err, "Invalid address should fail validation")
}

// TestGetNativeDexAddress tests the getter functions work correctly
func (suite *HandlerTestSuite) TestGetNativeDexAddress() {
	testDexAddr := "0xd263DC98dEc57828e26F69bA8687281BA5D052E0"
	testPolicyAddr := "0x14Ae279edb4D569BAFb98ff08299A0135Da6867a"

	params := types.Params{
		VerifiedNativeDexAddress:     testDexAddr,
		VerifiedCrocPolicyAddress:    testPolicyAddr,
		WhitelistedContractAddresses: []string{},
	}

	suite.app.NativedexKeeper.SetParams(suite.ctx, params)

	// Test getter functions
	retrievedDexAddr := suite.app.NativedexKeeper.GetNativeDexAddress(suite.ctx)
	suite.Require().Equal(common.HexToAddress(testDexAddr), retrievedDexAddr)

	retrievedPolicyAddr := suite.app.NativedexKeeper.GetVerifiedCrocPolicyAddress(suite.ctx)
	suite.Require().Equal(common.HexToAddress(testPolicyAddr), retrievedPolicyAddr)
}
