package types

import (
	errorsmod "cosmossdk.io/errors"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ibchost "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/ethereum/go-ethereum/common"
)

var (
	//nolint: exhaustruct
	_ sdk.Msg = &MsgConvertCoin{}
	//nolint: exhaustruct
	_ sdk.Msg = &MsgConvertERC20{}
	//nolint: exhaustruct
	_ sdk.Msg = &MsgSendCoinToEVM{}
	//nolint: exhaustruct
	_ sdk.Msg = &MsgSendERC20ToCosmos{}
	//nolint: exhaustruct
	_ sdk.Msg = &MsgSendERC20ToCosmosAndIBCTransfer{}
)

const (
	TypeMsgConvertCoin                     = "convert_coin"
	TypeMsgConvertERC20                    = "convert_ERC20"
	TypeMsgSendCoinToEVM                   = "send_coin_to_evm"
	TypeMsgSendERC20ToCosmos               = "send_ERC20_to_cosmos"
	TypeMsgSendERC20ToCosmosAndIBCTransfer = "send_ERC20_to_cosmos_and_ibc_transfer"
)

// NewMsgConvertCoin creates a new instance of MsgConvertCoin
func NewMsgConvertCoin(coin sdk.Coin, receiver common.Address, sender sdk.AccAddress) *MsgConvertCoin { // nolint: interfacer
	return &MsgConvertCoin{
		Coin:     coin,
		Receiver: receiver.Hex(),
		Sender:   sender.String(),
	}
}

// Route should return the name of the module
func (msg MsgConvertCoin) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConvertCoin) Type() string { return TypeMsgConvertCoin }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertCoin) ValidateBasic() error {
	if err := ValidateErc20Denom(msg.Coin.Denom); err != nil {
		if err := ibctransfertypes.ValidateIBCDenom(msg.Coin.Denom); err != nil {
			return err
		}
	}

	if !msg.Coin.Amount.IsPositive() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "cannot mint a non-positive amount")
	}
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errorsmod.Wrap(err, "invalid sender address")
	}
	if !common.IsHexAddress(msg.Receiver) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid receiver hex address %s", msg.Receiver)
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgConvertCoin) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertCoin) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

// NewMsgConvertERC20 creates a new instance of MsgConvertERC20
func NewMsgConvertERC20(amount sdkmath.Int, receiver sdk.AccAddress, contract, sender common.Address) *MsgConvertERC20 { // nolint: interfacer
	return &MsgConvertERC20{
		ContractAddress: contract.String(),
		Amount:          amount,
		Receiver:        receiver.String(),
		Sender:          sender.Hex(),
	}
}

// Route should return the name of the module
func (msg MsgConvertERC20) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConvertERC20) Type() string { return TypeMsgConvertERC20 }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertERC20) ValidateBasic() error {
	if !common.IsHexAddress(msg.ContractAddress) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid contract hex address '%s'", msg.ContractAddress)
	}
	if !msg.Amount.IsPositive() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "cannot mint a non-positive amount")
	}
	_, err := sdk.AccAddressFromBech32(msg.Receiver)
	if err != nil {
		return errorsmod.Wrap(err, "invalid receiver address")
	}
	if !common.IsHexAddress(msg.Sender) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender hex address %s", msg.Sender)
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgConvertERC20) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertERC20) GetSigners() []sdk.AccAddress {
	addr := common.HexToAddress(msg.Sender)
	return []sdk.AccAddress{addr.Bytes()}
}

// NewMsgSendCoinToEVM creates a new instance of MsgSendCoinToEVM
func NewMsgSendCoinToEVM(coin sdk.Coin, sender sdk.AccAddress) *MsgSendCoinToEVM { // nolint: interfacer
	return &MsgSendCoinToEVM{
		Coin:   coin,
		Sender: sender.String(),
	}
}

// Route should return the name of the module
func (msg MsgSendCoinToEVM) Route() string { return RouterKey }

// Type should return the action
func (msg MsgSendCoinToEVM) Type() string { return TypeMsgSendCoinToEVM }

// ValidateBasic runs stateless checks on the message
func (msg MsgSendCoinToEVM) ValidateBasic() error {
	if err := ValidateErc20Denom(msg.Coin.Denom); err != nil {
		if err := ibctransfertypes.ValidateIBCDenom(msg.Coin.Denom); err != nil {
			return err
		}
	}

	if !msg.Coin.Amount.IsPositive() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "cannot mint a non-positive amount")
	}
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errorsmod.Wrap(err, "invalid sender address")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgSendCoinToEVM) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&msg))
}

// GetSigners defines whose signature is required
func (msg MsgSendCoinToEVM) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

// NewMsgSendERC20ToCosmos creates a new instance of MsgSendERC20ToCosmos
func NewMsgSendERC20ToCosmos(amount sdkmath.Int, contract, sender common.Address) *MsgSendERC20ToCosmos { // nolint: interfacer
	return &MsgSendERC20ToCosmos{
		Erc20:  contract.Hex(),
		Amount: amount,
		Sender: sender.Hex(),
	}
}

// Route should return the name of the module
func (msg MsgSendERC20ToCosmos) Route() string { return RouterKey }

// Type should return the action
func (msg MsgSendERC20ToCosmos) Type() string { return TypeMsgSendERC20ToCosmos }

// ValidateBasic runs stateless checks on the message
func (msg MsgSendERC20ToCosmos) ValidateBasic() error {
	if !common.IsHexAddress(msg.Erc20) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid contract hex address '%s'", msg.Erc20)
	}
	if !msg.Amount.IsPositive() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "cannot mint a non-positive amount")
	}
	if !common.IsHexAddress(msg.Sender) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender hex address %s", msg.Sender)
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgSendERC20ToCosmos) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&msg))
}

// GetSigners defines whose signature is required
func (msg MsgSendERC20ToCosmos) GetSigners() []sdk.AccAddress {
	addr := common.HexToAddress(msg.Sender)
	return []sdk.AccAddress{addr.Bytes()}
}

// NewMsgSendERC20ToCosmosAndIBCTransfer creates a new instance of MsgSendERC20ToCosmosAndIBCTransfer
func NewMsgSendERC20ToCosmosAndIBCTransfer(amount sdkmath.Int, contract, sender common.Address, destPort, destChannel string, destReceiver sdk.AccAddress) *MsgSendERC20ToCosmosAndIBCTransfer { // nolint: interfacer
	return &MsgSendERC20ToCosmosAndIBCTransfer{
		Erc20:               contract.Hex(),
		Amount:              amount,
		Sender:              sender.Hex(),
		DestinationPort:     destPort,
		DestinationChannel:  destChannel,
		DestinationReceiver: destReceiver.String(),
	}
}

// Route should return the name of the module
func (msg MsgSendERC20ToCosmosAndIBCTransfer) Route() string { return RouterKey }

// Type should return the action
func (msg MsgSendERC20ToCosmosAndIBCTransfer) Type() string {
	return TypeMsgSendERC20ToCosmosAndIBCTransfer
}

// ValidateBasic runs stateless checks on the message
func (msg MsgSendERC20ToCosmosAndIBCTransfer) ValidateBasic() error {
	if !common.IsHexAddress(msg.Erc20) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid contract hex address '%s'", msg.Erc20)
	}
	if !msg.Amount.IsPositive() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "cannot convert a non-positive amount")
	}
	if !common.IsHexAddress(msg.Sender) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender hex address %s", msg.Sender)
	}
	if err := ibchost.PortIdentifierValidator(msg.DestinationPort); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid destination port: %v", err)
	}
	if err := ibchost.ChannelIdentifierValidator(msg.DestinationChannel); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid destination channel: %v", err)
	}
	_, _, err := bech32.DecodeAndConvert(msg.DestinationReceiver)
	if err != nil {
		return errorsmod.Wrap(err, "invalid destination receiver address")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgSendERC20ToCosmosAndIBCTransfer) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&msg))
}

// GetSigners defines whose signature is required
func (msg MsgSendERC20ToCosmosAndIBCTransfer) GetSigners() []sdk.AccAddress {
	addr := common.HexToAddress(msg.Sender)
	return []sdk.AccAddress{addr.Bytes()}
}
