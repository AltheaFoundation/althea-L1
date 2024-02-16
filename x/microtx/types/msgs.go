package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authlegacy "github.com/cosmos/cosmos-sdk/x/auth/legacy/legacytx"
)

const (
	TypeMsgMicrotx = "microtx"
	TypeMsgLiquify = "liquify"
)

// nolint: exhaustruct
var (
	_ sdk.Msg              = &MsgMicrotx{}
	_ sdk.Msg              = &MsgLiquify{}
	_ authlegacy.LegacyMsg = &MsgMicrotx{}
	_ authlegacy.LegacyMsg = &MsgLiquify{}
)

// NewMsgMicrotx returns a new MsgMicrotx
func NewMsgMicrotx(sender string, reciever string, amount sdk.Coin) *MsgMicrotx {
	return &MsgMicrotx{
		sender,
		reciever,
		amount,
	}
}

// Route should return the name of the module
func (msg *MsgMicrotx) Route() string { return RouterKey }

func (msg MsgMicrotx) Type() string { return TypeMsgMicrotx }

// ValidateBasic checks for valid addresses and amounts
func (msg *MsgMicrotx) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid sender in microtx msg microtx")
	}
	_, err = sdk.AccAddressFromBech32(msg.Receiver)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid receiver in microtx msg microtx")
	}
	if err := msg.Amount.Validate(); err != nil {
		return sdkerrors.Wrap(err, "invalid coin in microtx msg microtx")
	}

	if msg.Amount.Amount.Equal(sdk.ZeroInt()) {
		return sdkerrors.Wrap(ErrInvalidMicrotx, "zero amount in microtx msg microtx")
	}
	return nil
}

// GetSigners requires the Sender to be the signer
func (msg *MsgMicrotx) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{acc}
}

// GetSignBytes Implements Msg.
func (msg MsgMicrotx) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// NewMsgLiquify returns a new MsgLiquify
func NewMsgLiquify(sender string) *MsgLiquify {
	return &MsgLiquify{
		sender,
	}
}

// Route should return the name of the module
func (msg *MsgLiquify) Route() string { return RouterKey }

func (msg MsgLiquify) Type() string { return TypeMsgLiquify }

// ValidateBasic checks for valid addresses
func (msg *MsgLiquify) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid sender in microtx msg liquify")
	}

	return nil
}

// GetSigners requires the Sender to be the signer
func (msg *MsgLiquify) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{acc}
}

// GetSignBytes Implements Msg.
func (msg MsgLiquify) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}
