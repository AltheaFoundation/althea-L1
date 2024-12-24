package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	common "github.com/ethereum/go-ethereum/common"
)

// TODO: Specify JSON encoding for the typed events to match this format
// This file is a temporary measure to ensure the event log is readable and searchable.
// Event types have been defined in the microtx msgs.proto file but the EmitTypedEvents() function
// will produce incomprehensible results due to proto encoding.

const (
	EventTypeMicrotx = "microtx"

	MicrotxKeySender   = "sender"
	MicrotxKeyReceiver = "receiver"
	MicrotxKeyAmount   = "amount"

	EventTypeMicrotxFeeCollected = "microtx-fee-collected"
	MicrotxFeeCollectedKeySender = "sender"
	MicrotxFeeCollectedKeyFee    = "fee"

	EventTypeBalanceRedirect = "balance-redirect"
	RedirectKeyReceiver      = "receiver"
	RedirectKeyAmount        = "amount"

	EventTypeLiquify = "liquify"

	LiquifyKeyAccount    = "account"
	LiquifyKeyNFTAddress = "nft-address"

	EventTypeProposerReward = "proposer_reward"
	AttributeKeyValidator   = "validator"
)

func NewEventMicrotx(sender string, receiver string, amount sdk.Coin) sdk.Event {
	return sdk.NewEvent(
		EventTypeMicrotx,
		sdk.NewAttribute(MicrotxKeySender, sender),
		sdk.NewAttribute(MicrotxKeyReceiver, receiver),
		sdk.NewAttribute(MicrotxKeyAmount, amount.String()),
	)
}

func NewEventMicrotxFeeCollected(sender string, fee sdk.Coin) sdk.Event {
	return sdk.NewEvent(
		EventTypeMicrotxFeeCollected,
		sdk.NewAttribute(MicrotxFeeCollectedKeySender, sender),
		sdk.NewAttribute(MicrotxFeeCollectedKeyFee, fee.String()),
	)
}

func NewEventBalanceRedirect(receiver string, amount sdk.Coin) sdk.Event {
	return sdk.NewEvent(
		EventTypeBalanceRedirect,
		sdk.NewAttribute(RedirectKeyReceiver, receiver),
		sdk.NewAttribute(RedirectKeyAmount, amount.String()),
	)
}

func NewEventLiquify(account string, nftAddress common.Address) sdk.Event {
	return sdk.NewEvent(
		EventTypeLiquify,
		sdk.NewAttribute(LiquifyKeyAccount, account),
		sdk.NewAttribute(LiquifyKeyNFTAddress, nftAddress.Hex()),
	)
}
