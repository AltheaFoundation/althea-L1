package keeper_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"

	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	"github.com/tendermint/tendermint/version"

	althea "github.com/AltheaFoundation/althea-L1/app"
	"github.com/AltheaFoundation/althea-L1/contracts"
	"github.com/AltheaFoundation/althea-L1/x/nativedex"
	"github.com/AltheaFoundation/althea-L1/x/nativedex/types"
)

type ProposalHandlerTestSuite struct {
	suite.Suite
	ctx            sdk.Context
	app            *althea.AltheaApp
	queryClientEvm evmtypes.QueryClient
}

func TestProposalHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ProposalHandlerTestSuite))
}

func (suite *ProposalHandlerTestSuite) SetupTest() {
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

func (suite *ProposalHandlerTestSuite) TestExecuteContractProposal() {
	// Deploy an ERC20 token
	contractAddr := suite.DeployERC20("TestToken", "TEST", 18)
	suite.Require().NotEqual(common.Address{}, contractAddr)

	moduleBalancePreMint := suite.GetERC20Balance(contractAddr, types.ModuleEVMAddress)
	suite.T().Logf("Module balance (%s) must be %s before minting\n", moduleBalancePreMint.String(), "0")
	suite.Require().Equal("0", moduleBalancePreMint.String(), "Module balance should be zero before minting")

	// Create a recipient address
	recipientAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	// Amount to transfer (100 tokens with 18 decimals)
	transferAmount := new(big.Int).Mul(big.NewInt(100), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

	// First, mint tokens to the nativedex module address
	suite.MintERC20Tokens(contractAddr, types.ModuleEVMAddress, transferAmount)

	// Verify the module has the tokens
	moduleBalance := suite.GetERC20Balance(contractAddr, types.ModuleEVMAddress)
	suite.T().Logf("Module balance (%s) should have updated to %s\n", moduleBalance.String(), transferAmount.String())
	suite.Require().Equal(transferAmount.String(), moduleBalance.String(), "Module should have minted tokens")

	// Encode the transfer function call
	transferData, err := contracts.ERC20MinterBurnerDecimalsContract.ABI.Pack("transfer", recipientAddr, transferAmount)
	suite.Require().NoError(err)

	// Convert to hex string
	hexData := "0x" + common.Bytes2Hex(transferData)

	// Update params to whitelist the ERC20 contract
	params := suite.app.NativedexKeeper.GetParams(suite.ctx)
	params.WhitelistedContractAddresses = []string{contractAddr.Hex()}
	suite.app.NativedexKeeper.SetParams(suite.ctx, params)

	// Create and execute the proposal
	metadata := types.ExecuteContractMetadata{
		ContractAddress: contractAddr.Hex(),
		Data:            hexData,
	}

	proposal := types.NewExecuteContractProposal(
		"Transfer ERC20 Tokens",
		"Transfer 100 TEST tokens to recipient",
		metadata,
	)

	// Get the proposal handler
	handler := nativedex.NewNativeDexProposalHandler(suite.app.NativedexKeeper)

	// Execute the proposal
	err = handler(suite.ctx, proposal)
	suite.Require().NoError(err, "Proposal execution should succeed")

	// Verify the recipient received the tokens
	recipientBalance := suite.GetERC20Balance(contractAddr, recipientAddr)
	suite.T().Logf("Recipient balance (%s) should have updated to %s\n", recipientBalance.String(), transferAmount.String())
	suite.Require().Equal(transferAmount.String(), recipientBalance.String(), "Recipient should have received tokens")

	// Verify the module balance is now zero
	moduleBalanceAfter := suite.GetERC20Balance(contractAddr, types.ModuleEVMAddress)
	suite.T().Logf("Module balance (%s) should have updated to %s\n", moduleBalanceAfter.String(), "0")
	suite.Require().Equal("0", moduleBalanceAfter.String(), "Module balance should be zero after transfer")
}

func (suite *ProposalHandlerTestSuite) TestExecuteContractProposal_NotWhitelisted() {
	// Deploy an ERC20 token
	contractAddr := suite.DeployERC20("TestToken", "TEST", 18)
	suite.Require().NotEqual(common.Address{}, contractAddr)

	moduleBalancePreMint := suite.GetERC20Balance(contractAddr, types.ModuleEVMAddress)
	suite.T().Logf("Module balance (%s) must be %s before minting\n", moduleBalancePreMint.String(), "0")
	suite.Require().Equal("0", moduleBalancePreMint.String(), "Module balance should be zero before minting")

	// Mint the ERC20 to the module so that it could plausibly transfer
	transferAmount := big.NewInt(100)

	suite.MintERC20Tokens(contractAddr, types.ModuleEVMAddress, transferAmount)

	// Verify the module has the tokens
	moduleBalance := suite.GetERC20Balance(contractAddr, types.ModuleEVMAddress)
	suite.T().Logf("Module balance (%s) should have updated to %s\n", moduleBalance.String(), transferAmount.String())
	suite.Require().Equal(transferAmount.String(), moduleBalance.String(), "Module should have minted tokens")

	// Create a recipient address
	recipientAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	// Encode the transfer function call
	transferData, err := contracts.ERC20MinterBurnerDecimalsContract.ABI.Pack("transfer", recipientAddr, transferAmount)
	suite.Require().NoError(err)

	// Convert to hex string
	hexData := "0x" + common.Bytes2Hex(transferData)

	// DON'T whitelist the contract - should fail

	// Create the proposal
	metadata := types.ExecuteContractMetadata{
		ContractAddress: contractAddr.Hex(),
		Data:            hexData,
	}

	proposal := types.NewExecuteContractProposal(
		"Transfer ERC20 Tokens",
		"Transfer tokens to recipient",
		metadata,
	)

	// Get the proposal handler
	handler := nativedex.NewNativeDexProposalHandler(suite.app.NativedexKeeper)

	// Execute the proposal - should fail because contract is not whitelisted
	err = handler(suite.ctx, proposal)
	suite.Require().Error(err, "Proposal execution should fail for non-whitelisted contract")
	suite.Require().Contains(err.Error(), "not whitelisted")

	moduleBalance = suite.GetERC20Balance(contractAddr, types.ModuleEVMAddress)
	suite.T().Logf("Module balance (%s) must remain at %s after the proposal failure\n", moduleBalance.String(), transferAmount.String())
	suite.Require().Equal(transferAmount.String(), moduleBalance.String(), "Module should still have the tokens")
	panic("Hi there")
}

func (suite *ProposalHandlerTestSuite) DeployERC20(name, symbol string, decimals uint8) common.Address {
	// Prepare constructor arguments
	ctorArgs, err := contracts.ERC20MinterBurnerDecimalsContract.ABI.Pack("", name, symbol, decimals)
	suite.Require().NoError(err)

	// Combine bytecode and constructor arguments
	data := append(contracts.ERC20MinterBurnerDecimalsContract.Bin, ctorArgs...)

	// Get nonce for the module account
	nonce, err := suite.app.AccountKeeper.GetSequence(suite.ctx, types.ModuleEVMAddress.Bytes())
	suite.Require().NoError(err)

	// Calculate the contract address
	contractAddr := crypto.CreateAddress(types.ModuleEVMAddress, nonce)

	// Deploy the contract
	_, err = suite.app.NativedexKeeper.EVMKeeper.CallEVMWithData(suite.ctx, types.ModuleEVMAddress, nil, data, true)
	suite.Require().NoError(err)

	return contractAddr
}

// MintERC20Tokens mints tokens to a specified address
func (suite *ProposalHandlerTestSuite) MintERC20Tokens(contractAddr, to common.Address, amount *big.Int) {
	// Pack the mint function call
	mintData, err := contracts.ERC20MinterBurnerDecimalsContract.ABI.Pack("mint", to, amount)
	suite.Require().NoError(err)

	// Call the mint function
	res, err := suite.app.NativedexKeeper.EVMKeeper.CallEVMWithData(suite.ctx, types.ModuleEVMAddress, &contractAddr, mintData, true)
	suite.Require().NoError(err)
	suite.Require().Empty(res.VmError)
}

// GetERC20Balance returns the ERC20 balance of an address
func (suite *ProposalHandlerTestSuite) GetERC20Balance(contractAddr, account common.Address) *big.Int {
	// Pack the balanceOf function call
	balanceData, err := contracts.ERC20MinterBurnerDecimalsContract.ABI.Pack("balanceOf", account)
	suite.Require().NoError(err)

	// Call balanceOf
	res, err := suite.app.NativedexKeeper.EVMKeeper.CallEVMWithData(suite.ctx, types.ModuleEVMAddress, &contractAddr, balanceData, false)
	suite.Require().NoError(err)

	// Unpack the result
	unpacked, err := contracts.ERC20MinterBurnerDecimalsContract.ABI.Unpack("balanceOf", res.Ret)
	suite.Require().NoError(err)
	suite.Require().Len(unpacked, 1)

	balance, ok := unpacked[0].(*big.Int)
	suite.Require().True(ok)

	return balance
}
