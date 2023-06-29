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
	EventTypeXfer = "xfer"

	XferKeySender   = "sender"
	XferKeyReceiver = "receiver"
	XferKeyAmounts  = "amounts"
	XferKeyFee      = "fee"

	EventTypeBalanceRedirect = "balance-redirect"
	RedirectKeyReceiver      = "receiver"
	RedirectKeyAmounts       = "amounts"

	EventTypeTokenizedAccount = "tokenized-account"

	TokenizedAccountKeyAccount    = "account"
	TokenizedAccountKeyNFTAddress = "nft-address"
)

func NewEventXfer(sender string, receiver string, amounts sdk.Coins, fees sdk.Coins) sdk.Event {
	return sdk.NewEvent(
		EventTypeXfer,
		sdk.NewAttribute(XferKeySender, sender),
		sdk.NewAttribute(XferKeyReceiver, receiver),
		sdk.NewAttribute(XferKeyAmounts, amounts.String()),
		sdk.NewAttribute(XferKeyFee, fees.String()),
	)
}

func NewEventBalanceRedirect(receiver string, amounts sdk.Coins) sdk.Event {
	return sdk.NewEvent(
		EventTypeBalanceRedirect,
		sdk.NewAttribute(RedirectKeyReceiver, receiver),
		sdk.NewAttribute(RedirectKeyAmounts, amounts.String()),
	)
}

func NewEventTokenizedAccount(account string, nftAddress common.Address) sdk.Event {
	return sdk.NewEvent(
		EventTypeTokenizedAccount,
		sdk.NewAttribute(TokenizedAccountKeyAccount, account),
		sdk.NewAttribute(TokenizedAccountKeyNFTAddress, nftAddress.Hex()),
	)
}
