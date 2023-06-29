package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// nolint: exhaustruct
var (
	_ sdk.Msg = &MsgXfer{}
)

// NewMsgXfer returns a new MsgXfer
func NewMsgXfer(sender string, reciever string, amounts sdk.Coins) *MsgXfer {
	return &MsgXfer{
		sender,
		reciever,
		amounts,
	}
}

// Route should return the name of the module
func (msg *MsgXfer) Route() string { return RouterKey }

// ValidateBasic checks for valid addresses and amounts
func (msg *MsgXfer) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid sender in microtx msg xfer")
	}
	_, err = sdk.AccAddressFromBech32(msg.Receiver)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid receiver in microtx msg xfer")
	}
	for _, amt := range msg.Amounts {
		if err := amt.Validate(); err != nil {
			return sdkerrors.Wrap(err, "invalid coin in microtx msg xfer")
		}

		if amt.Amount.Equal(sdk.ZeroInt()) {
			return sdkerrors.Wrap(ErrInvalidMicrotx, "zero amount in microtx msg xfer")
		}
	}
	return nil
}

// GetSigners requires the Sender to be the signer
func (msg *MsgXfer) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{acc}
}

// NewMsgTokenizeAccount returns a new MsgTokenizeAccount
func NewMsgTokenizeAccount(sender string) *MsgTokenizeAccount {
	return &MsgTokenizeAccount{
		sender,
	}
}

// Route should return the name of the module
func (msg *MsgTokenizeAccount) Route() string { return RouterKey }

// ValidateBasic checks for valid addresses
func (msg *MsgTokenizeAccount) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid sender in microtx msg tokenize account")
	}

	return nil
}

// GetSigners requires the Sender to be the signer
func (msg *MsgTokenizeAccount) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{acc}
}
