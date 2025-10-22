package testutil

import (
	"context"

	"github.com/stretchr/testify/mock"

	tmbytes "github.com/tendermint/tendermint/libs/bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"

	erc20types "github.com/AltheaFoundation/althea-L1/x/erc20/types"
	onboardingtypes "github.com/AltheaFoundation/althea-L1/x/onboarding/types"
)

// nolint: exhaustruct
var _ onboardingtypes.TransferKeeper = &MockTransferKeeper{}

// nolint: exhaustruct
var _ erc20types.IBCTransferKeeper = &MockTransferKeeper{}

// MockTransferKeeper defines a mocked object that implements the TransferKeeper
// interface. It's used on tests to abstract the complexity of IBC transfers.
// NOTE: Bank keeper logic is not mocked since we want to test that balance has
// been updated for sender and recipient.
type MockTransferKeeper struct {
	mock.Mock
	bankkeeper.Keeper
	Sequences map[string]uint64
}

func (m *MockTransferKeeper) GetDenomTrace(ctx sdk.Context, denomTraceHash tmbytes.HexBytes) (transfertypes.DenomTrace, bool) {
	args := m.Called(mock.Anything, denomTraceHash)
	return args.Get(0).(transfertypes.DenomTrace), args.Bool(1)
}

func (m *MockTransferKeeper) SendTransfer(
	ctx sdk.Context,
	sourcePort,
	sourceChannel string,
	token sdk.Coin,
	sender sdk.AccAddress,
	receiver string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
) error {
	args := m.Called(mock.Anything, sourcePort, sourceChannel, token, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	err := m.SendCoinsFromAccountToModule(ctx, sender, transfertypes.ModuleName, sdk.Coins{token})
	if err != nil {
		return err
	}

	return args.Error(0)
}

// Transfer implements types.IBCTransferKeeper.
func (m *MockTransferKeeper) Transfer(goCtx context.Context, msg *transfertypes.MsgTransfer) (*transfertypes.MsgTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	sequence := m.Sequences[msg.SourceChannel+msg.SourcePort] + 1
	m.Sequences[msg.SourceChannel+msg.SourcePort] = sequence
	return &transfertypes.MsgTransferResponse{Sequence: sequence}, m.SendTransfer(ctx, msg.SourcePort, msg.SourceChannel, msg.Token, sender, msg.Receiver, msg.TimeoutHeight, msg.TimeoutTimestamp)
}
