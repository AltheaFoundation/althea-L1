package onboarding_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	ibcgotesting "github.com/cosmos/ibc-go/v6/testing"

	evmtypes "github.com/evmos/ethermint/x/evm/types"

	"github.com/AltheaFoundation/althea-L1/contracts"

	althea "github.com/AltheaFoundation/althea-L1/app"
	ibctesting "github.com/AltheaFoundation/althea-L1/ibcutils/testing"
	onboardingtest "github.com/AltheaFoundation/althea-L1/x/onboarding/testutil"
)

var (
	ibcBase     = "ibc/C053D637CCA2A2BA030E2C5EE1B28A16F71CCB0E45E8BE52766DC1B241B77878"
	metadataIbc = banktypes.Metadata{
		Description: "IBC voucher (channel 0)",
		Base:        ibcBase,
		// NOTE: Denom units MUST be increasing
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    ibcBase,
				Exponent: 0,
			},
		},
		Name:    "Ibc Token channel-0",
		Symbol:  "ibcToken-0",
		Display: ibcBase,
	}
)

type TransferTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (suite *TransferTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1, 1)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainIDAlthea(1))
}

func NewTransferPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
	path.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort
	path.EndpointA.ChannelConfig.Version = transfertypes.Version
	path.EndpointB.ChannelConfig.Version = transfertypes.Version

	return path
}

// constructs a send from chainA to chainB on the established channel/connection
// and sends the same coin again from chainA to chainB.
func (suite *TransferTestSuite) TestHandleMsgTransfer() {
	// setup path between chainA and chainB
	path := NewTransferPath(suite.chainA, suite.chainB)
	suite.coordinator.Setup(path)

	// Fund chainB
	coins := sdk.NewCoins(
		sdk.NewCoin(ibcBase, sdk.NewInt(1000000000000)),
		sdk.NewCoin("aalthea", sdk.NewInt(10000000000)),
	)
	suite.Require().NoError(suite.chainB.App.(*althea.AltheaApp).BankKeeper.MintCoins(suite.chainB.GetContext(), evmtypes.ModuleName, coins))
	suite.Require().NoError(suite.chainB.App.(*althea.AltheaApp).BankKeeper.SendCoinsFromModuleToAccount(suite.chainB.GetContext(), evmtypes.ModuleName, suite.chainB.SenderAccount.GetAddress(), coins))

	middlewareParams := suite.chainB.App.(*althea.AltheaApp).OnboardingKeeper.GetParams(suite.chainB.GetContext())
	middlewareParams.WhitelistedChannels = []string{path.EndpointB.ChannelID}
	suite.chainB.App.(*althea.AltheaApp).OnboardingKeeper.SetParams(suite.chainB.GetContext(), middlewareParams)

	erc20Keeper := suite.chainB.App.(*althea.AltheaApp).Erc20Keeper
	pair, err := erc20Keeper.RegisterCoin(suite.chainB.GetContext(), metadataIbc)
	suite.Require().NoError(err)

	timeoutHeight := clienttypes.NewHeight(10, 100)

	amount, ok := sdk.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
	suite.Require().True(ok)
	coinToSendToB := sdk.NewCoin(sdk.DefaultBondDenom, amount)

	// send coins from chainA to chainB
	// auto swap and auto convert should happen
	msg := transfertypes.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, coinToSendToB, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	voucherDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(packet.GetDestPort(), packet.GetDestChannel(), sdk.DefaultBondDenom))

	// check balances on chainB before the IBC transfer
	balanceVoucherBefore := suite.chainB.App.(*althea.AltheaApp).BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	balanceErc20Before := erc20Keeper.BalanceOf(suite.chainB.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(suite.chainB.SenderAccount.GetAddress().Bytes()))

	// relay send
	_, err = onboardingtest.RelayPacket(path, packet)
	suite.Require().NoError(err) // relay committed

	// check balances on chainB after the IBC transfer
	balanceVoucher := suite.chainB.App.(*althea.AltheaApp).BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	balanceErc20 := erc20Keeper.BalanceOf(suite.chainB.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(suite.chainB.SenderAccount.GetAddress().Bytes()))

	coinSentFromAToB := transfertypes.GetTransferCoin(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sdk.DefaultBondDenom, amount)

	// Check that the IBC voucher balance is same
	suite.Require().Equal(balanceVoucherBefore, balanceVoucher)

	// Check that the convert is successful
	before := sdk.NewIntFromBigInt(balanceErc20Before)
	suite.Require().True(before.IsZero())
	suite.Require().Equal(coinSentFromAToB.Amount, sdk.NewIntFromBigInt(balanceErc20))

	// IBC transfer to blocked address
	blockedAddr := "althea10d07y265gmmuvt4z0w9aw880jnsr700jwqkt6k"
	coinToSendToB = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	msg = transfertypes.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, coinToSendToB, suite.chainA.SenderAccount.GetAddress().String(), blockedAddr, timeoutHeight, 0, "")

	res, err = suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err = ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	// relay send
	res, err = onboardingtest.RelayPacket(path, packet)
	suite.Require().NoError(err)
	ack, err := ibcgotesting.ParseAckFromEvents(res.GetEvents())
	suite.Require().NoError(err)
	suite.Require().Equal(ack, []byte(`{"error":"ABCI code: 1: error handling packet: see events for details"}`))

	// Send again from chainA to chainB
	// auto swap should not happen
	// auto convert all transferred IBC vouchers to ERC20
	coinToSendToB = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	balanceVoucherBefore = suite.chainB.App.(*althea.AltheaApp).BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	balanceErc20Before = erc20Keeper.BalanceOf(suite.chainB.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(suite.chainB.SenderAccount.GetAddress().Bytes()))

	msg = transfertypes.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, coinToSendToB, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")

	res, err = suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err = ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	// relay send
	err = path.RelayPacket(packet)
	suite.Require().NoError(err) // relay committed

	coinSentFromAToB = transfertypes.GetTransferCoin(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sdk.DefaultBondDenom, coinToSendToB.Amount)
	balanceVoucher = suite.chainB.App.(*althea.AltheaApp).BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	balanceErc20 = erc20Keeper.BalanceOf(suite.chainB.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(suite.chainB.SenderAccount.GetAddress().Bytes()))

	suite.Require().Equal(balanceVoucherBefore, balanceVoucher)
	suite.Require().Equal(sdk.NewIntFromBigInt(balanceErc20Before).Add(coinSentFromAToB.Amount), sdk.NewIntFromBigInt(balanceErc20))

}

func TestTransferTestSuite(t *testing.T) {
	suite.Run(t, new(TransferTestSuite))
}
