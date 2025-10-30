package ante_test

import (
	"fmt"
	"testing"
	"time"

	ante "github.com/AltheaFoundation/althea-L1/app/ante"
	"github.com/stretchr/testify/suite"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	grouptypes "github.com/cosmos/cosmos-sdk/x/group"

	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

type GroupLimiterTestSuite struct {
	AnteTestSuite
}

func TestGroupLimiterTestSuite(t *testing.T) {
	// nolint: exhaustruct
	suite.Run(t, &GroupLimiterTestSuite{})
}

func (suite *GroupLimiterTestSuite) TestGroupLimiterDecorator() {
	testPrivKeys, testAddresses, err := generatePrivKeyAddressPairs(5)
	suite.Require().NoError(err)

	decorator := ante.NewGroupLimiterDecorator(
		[]string{
			// nolint: exhaustruct
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
		},
	)

	testMsgSend := createMsgSend(testAddresses)
	// nolint: exhaustruct
	testMsgEthereumTx := &evmtypes.MsgEthereumTx{}

	// Create a group with members
	members := []grouptypes.MemberRequest{
		{
			Address:  testAddresses[1].String(),
			Weight:   "1",
			Metadata: "member 1",
		},
		{
			Address:  testAddresses[2].String(),
			Weight:   "1",
			Metadata: "member 2",
		},
	}

	msgCreateGroup := &grouptypes.MsgCreateGroup{
		Admin:    testAddresses[0].String(),
		Members:  members,
		Metadata: "test group",
	}

	// Create a group policy with threshold decision policy
	policy := grouptypes.NewThresholdDecisionPolicy(
		"1",
		time.Second*10,
		0,
	)

	msgCreateGroupPolicy, err := grouptypes.NewMsgCreateGroupPolicy(
		testAddresses[0],
		1, // group id
		"test group policy",
		policy,
	)
	suite.Require().NoError(err)

	// Use a placeholder group policy address for the test
	groupPolicyAddress := testAddresses[3].String()

	testCases := []struct {
		name        string
		msgs        []sdk.Msg
		expectedErr error
	}{
		{
			"setup - create group",
			[]sdk.Msg{
				msgCreateGroup,
			},
			nil,
		},
		{
			"setup - create group policy",
			[]sdk.Msg{
				msgCreateGroupPolicy,
			},
			nil,
		},
		{
			"enabled msg - non blocked msg",
			[]sdk.Msg{
				testMsgSend,
			},
			nil,
		},
		{
			"enabled msg - MsgEthereumTx not wrapped in MsgSubmitProposal",
			[]sdk.Msg{
				testMsgEthereumTx,
			},
			nil,
		},
		{
			"enabled msg - MsgSubmitProposal contains non-blocked msg",
			[]sdk.Msg{
				createMsgSubmitProposal(
					groupPolicyAddress,
					[]string{testAddresses[1].String()},
					[]sdk.Msg{testMsgSend},
				),
			},
			nil,
		},
		{
			"disabled msg - MsgSubmitProposal contains MsgEthereumTx",
			[]sdk.Msg{
				createMsgSubmitProposal(
					groupPolicyAddress,
					[]string{testAddresses[1].String()},
					[]sdk.Msg{testMsgEthereumTx},
				),
			},
			sdkerrors.ErrUnauthorized,
		},
		{
			"disabled msg - MsgSubmitProposal contains multiple msgs including MsgEthereumTx",
			[]sdk.Msg{
				createMsgSubmitProposal(
					groupPolicyAddress,
					[]string{testAddresses[1].String()},
					[]sdk.Msg{
						testMsgSend,
						testMsgEthereumTx,
					},
				),
			},
			sdkerrors.ErrUnauthorized,
		},
		{
			"disabled msg - nested MsgSubmitProposal",
			[]sdk.Msg{
				createMsgSubmitProposal(
					groupPolicyAddress,
					[]string{testAddresses[1].String()},
					[]sdk.Msg{
						createMsgSubmitProposal(
							groupPolicyAddress,
							[]string{testAddresses[1].String()},
							[]sdk.Msg{testMsgSend},
						),
					},
				),
			},
			sdkerrors.ErrUnauthorized,
		},
		{
			"disabled msg - MsgSubmitProposal with nested levels over limit",
			[]sdk.Msg{
				createNestedMsgSubmitProposal(
					groupPolicyAddress,
					[]string{testAddresses[1].String()},
					6,
					testMsgSend,
				),
			},
			sdkerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest()
			tx, err := suite.createTx(testPrivKeys[0], tc.msgs...)
			suite.Require().NoError(err)

			_, err = decorator.AnteHandle(suite.ctx, tx, false, NextFn)
			if tc.expectedErr != nil {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expectedErr)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

// createMsgSubmitProposal creates a MsgSubmitProposal with the given parameters
func createMsgSubmitProposal(groupPolicyAddress string, proposers []string, msgs []sdk.Msg) *grouptypes.MsgSubmitProposal {
	msg, err := grouptypes.NewMsgSubmitProposal(
		groupPolicyAddress,
		proposers,
		msgs,
		"test metadata",
		grouptypes.Exec_EXEC_UNSPECIFIED,
	)
	if err != nil {
		panic(err)
	}
	return msg
}

// createNestedMsgSubmitProposal creates nested MsgSubmitProposal messages to test depth limits
func createNestedMsgSubmitProposal(groupPolicyAddress string, proposers []string, depth int, innerMsg sdk.Msg) *grouptypes.MsgSubmitProposal {
	if depth == 1 {
		return createMsgSubmitProposal(groupPolicyAddress, proposers, []sdk.Msg{innerMsg})
	}

	nestedProposal := createNestedMsgSubmitProposal(groupPolicyAddress, proposers, depth-1, innerMsg)
	return createMsgSubmitProposal(groupPolicyAddress, proposers, []sdk.Msg{nestedProposal})
}

// generatePrivKeyAddressPairs generates private keys and addresses for testing
func generatePrivKeyAddressPairs(accCount int) ([]*ethsecp256k1.PrivKey, []sdk.AccAddress, error) {
	var (
		err           error
		testPrivKeys  = make([]*ethsecp256k1.PrivKey, accCount)
		testAddresses = make([]sdk.AccAddress, accCount)
	)

	for i := range testPrivKeys {
		testPrivKeys[i], err = ethsecp256k1.GenerateKey()
		if err != nil {
			return nil, nil, err
		}
		testAddresses[i] = testPrivKeys[i].PubKey().Address().Bytes()
	}
	return testPrivKeys, testAddresses, nil
}

// createMsgSend creates a MsgSend for testing
func createMsgSend(testAddresses []sdk.AccAddress) *banktypes.MsgSend {
	return banktypes.NewMsgSend(
		testAddresses[0],
		testAddresses[3],
		sdk.NewCoins(sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 1e8)),
	)
}

// createTx creates a transaction for testing
func (suite *GroupLimiterTestSuite) createTx(priv cryptotypes.PrivKey, msgs ...sdk.Msg) (sdk.Tx, error) {
	txBuilder := suite.clientCtx.TxConfig.NewTxBuilder()

	err := txBuilder.SetMsgs(msgs...)
	if err != nil {
		return nil, err
	}

	txBuilder.SetGasLimit(1000000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 1000)))

	return txBuilder.GetTx(), nil
}
