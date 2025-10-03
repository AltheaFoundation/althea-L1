package keeper

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/armon/go-metrics"

	"github.com/ethereum/go-ethereum/common"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"

	"github.com/AltheaFoundation/althea-L1/contracts"
	altheacommon "github.com/AltheaFoundation/althea-L1/x/common"
	"github.com/AltheaFoundation/althea-L1/x/erc20/types"
)

// nolint: exhaustruct
var _ types.MsgServer = &Keeper{}

// ConvertCoin converts native Cosmos coins into ERC20 tokens for both
// Cosmos-native and ERC20 TokenPair Owners
func (k Keeper) ConvertCoin(
	goCtx context.Context,
	msg *types.MsgConvertCoin,
) (*types.MsgConvertCoinResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Error checked during msg validation
	receiver := common.HexToAddress(msg.Receiver)
	sender := sdk.MustAccAddressFromBech32(msg.Sender)

	pair, err := k.MintingEnabled(ctx, sender, receiver.Bytes(), msg.Coin.Denom)
	if err != nil {
		return nil, err
	}

	// Remove token pair if contract is suicided
	erc20 := common.HexToAddress(pair.Erc20Address)
	acc := k.evmKeeper.GetAccountWithoutBalance(ctx, erc20)

	if acc == nil || !acc.IsContract() {
		k.DeleteTokenPair(ctx, pair)
		k.Logger(ctx).Debug(
			"deleting selfdestructed token pair from state",
			"contract", pair.Erc20Address,
		)
		// NOTE: return nil error to persist the changes from the deletion
		return nil, nil
	}

	// Check ownership and execute conversion
	switch {
	case pair.IsNativeCoin():
		return k.convertCoinNativeCoin(ctx, pair, msg, receiver, sender) // case 1.1
	case pair.IsNativeERC20():
		return k.convertCoinNativeERC20(ctx, pair, msg, receiver, sender) // case 2.2
	default:
		return nil, types.ErrUndefinedOwner
	}
}

// ConvertERC20 converts ERC20 tokens into native Cosmos coins for both
// Cosmos-native and ERC20 TokenPair Owners
func (k Keeper) ConvertERC20(
	goCtx context.Context,
	msg *types.MsgConvertERC20,
) (*types.MsgConvertERC20Response, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Error checked during msg validation
	receiver := sdk.MustAccAddressFromBech32(msg.Receiver)
	sender := common.HexToAddress(msg.Sender)

	pair, err := k.MintingEnabled(ctx, sender.Bytes(), receiver, msg.ContractAddress)
	if err != nil {
		return nil, err
	}

	// Remove token pair if contract is suicided
	erc20 := common.HexToAddress(pair.Erc20Address)
	acc := k.evmKeeper.GetAccountWithoutBalance(ctx, erc20)

	if acc == nil || !acc.IsContract() {
		k.DeleteTokenPair(ctx, pair)
		k.Logger(ctx).Debug(
			"deleting selfdestructed token pair from state",
			"contract", pair.Erc20Address,
		)
		// NOTE: return nil error to persist the changes from the deletion
		return nil, nil
	}

	// Check ownership and execute conversion
	switch {
	case pair.IsNativeCoin():
		return k.convertERC20NativeCoin(ctx, pair, msg, receiver, sender) // case 1.2
	case pair.IsNativeERC20():
		return k.convertERC20NativeToken(ctx, pair, msg, receiver, sender) // case 2.1
	default:
		return nil, types.ErrUndefinedOwner
	}
}

// convertCoinNativeCoin handles the coin conversion for a native Cosmos coin
// token pair:
//   - escrow coins on module account
//   - mint tokens and send to receiver
//   - check if token balance increased by amount
func (k Keeper) convertCoinNativeCoin(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertCoin,
	receiver common.Address,
	sender sdk.AccAddress,
) (*types.MsgConvertCoinResponse, error) {
	// NOTE: ignore validation from NewCoin constructor
	coins := sdk.Coins{msg.Coin}
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	contract := pair.GetERC20Contract()
	balanceToken := k.BalanceOf(ctx, erc20, contract, receiver)
	if balanceToken == nil {
		return nil, errorsmod.Wrap(types.ErrEVMCall, "failed to retrieve balance")
	}

	// Escrow coins on module account
	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, coins)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to escrow coins")
	}

	// Mint tokens and send to receiver
	_, err = k.CallEVM(ctx, erc20, types.ModuleAddress, contract, true, "mint", receiver, msg.Coin.Amount.BigInt())
	if err != nil {
		return nil, err
	}

	// Check expected receiver balance after transfer
	tokens := msg.Coin.Amount.BigInt()
	balanceTokenAfter := k.BalanceOf(ctx, erc20, contract, receiver)
	if balanceTokenAfter == nil {
		return nil, errorsmod.Wrap(types.ErrEVMCall, "failed to retrieve balance")
	}
	expToken := big.NewInt(0).Add(balanceToken, tokens)

	if r := balanceTokenAfter.Cmp(expToken); r != 0 {
		return nil, errorsmod.Wrapf(
			types.ErrBalanceInvariance,
			"invalid token balance - expected: %v, actual: %v", expToken, balanceTokenAfter,
		)
	}

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"tx", "msg", "convert", "coin", "total"},
			1,
			[]metrics.Label{
				telemetry.NewLabel("denom", pair.Denom),
				telemetry.NewLabel("erc20", pair.Erc20Address),
			},
		)

		if msg.Coin.Amount.IsInt64() {
			telemetry.IncrCounterWithLabels(
				[]string{"tx", "msg", "convert", "coin", "amount", "total"},
				float32(msg.Coin.Amount.Int64()),
				[]metrics.Label{
					telemetry.NewLabel("denom", pair.Denom),
					telemetry.NewLabel("erc20", pair.Erc20Address),
				},
			)
		}
	}()

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertCoin,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Coin.Amount.String()),
				sdk.NewAttribute(types.AttributeKeyCosmosCoin, msg.Coin.Denom),
				sdk.NewAttribute(types.AttributeKeyERC20Token, pair.Erc20Address),
			),
		},
	)

	return &types.MsgConvertCoinResponse{}, nil
}

// convertERC20NativeCoin handles the erc20 conversion for a native Cosmos coin
// token pair:
//   - burn escrowed tokens
//   - unescrow coins that have been previously escrowed with ConvertCoin
//   - check if coin balance increased by amount
//   - check if token balance decreased by amount
func (k Keeper) convertERC20NativeCoin(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertERC20,
	receiver sdk.AccAddress,
	sender common.Address,
) (*types.MsgConvertERC20Response, error) {
	// NOTE: coin fields already validated
	coins := sdk.Coins{sdk.Coin{Denom: pair.Denom, Amount: msg.Amount}}
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	contract := pair.GetERC20Contract()
	balanceCoin := k.bankKeeper.GetBalance(ctx, receiver, pair.Denom)
	balanceToken := k.BalanceOf(ctx, erc20, contract, sender)
	if balanceToken == nil {
		return nil, errorsmod.Wrap(types.ErrEVMCall, "failed to retrieve balance")
	}

	// Burn escrowed tokens
	_, err := k.CallEVM(ctx, erc20, types.ModuleAddress, contract, true, "burnCoins", sender, msg.Amount.BigInt())
	if err != nil {
		return nil, err
	}

	// Unescrow coins and send to receiver
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, coins)
	if err != nil {
		return nil, err
	}

	// Check expected receiver balance after transfer
	balanceCoinAfter := k.bankKeeper.GetBalance(ctx, receiver, pair.Denom)
	expCoin := balanceCoin.Add(coins[0])
	if ok := balanceCoinAfter.IsEqual(expCoin); !ok {
		return nil, errorsmod.Wrapf(
			types.ErrBalanceInvariance,
			"invalid coin balance - expected: %v, actual: %v",
			expCoin, balanceCoinAfter,
		)
	}

	// Check expected Sender balance after transfer
	tokens := coins[0].Amount.BigInt()
	balanceTokenAfter := k.BalanceOf(ctx, erc20, contract, sender)
	if balanceTokenAfter == nil {
		return nil, errorsmod.Wrap(types.ErrEVMCall, "failed to retrieve balance")
	}

	expToken := big.NewInt(0).Sub(balanceToken, tokens)
	if r := balanceTokenAfter.Cmp(expToken); r != 0 {
		return nil, errorsmod.Wrapf(
			types.ErrBalanceInvariance,
			"invalid token balance - expected: %v, actual: %v",
			expToken, balanceTokenAfter,
		)
	}

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"tx", "msg", "convert", "erc20", "total"},
			1,
			[]metrics.Label{
				telemetry.NewLabel("denom", pair.Denom),
				telemetry.NewLabel("erc20", pair.Erc20Address),
			},
		)

		if msg.Amount.IsInt64() {
			telemetry.IncrCounterWithLabels(
				[]string{"tx", "msg", "convert", "erc20", "amount", "total"},
				float32(msg.Amount.Int64()),
				[]metrics.Label{
					telemetry.NewLabel("denom", pair.Denom),
					telemetry.NewLabel("erc20", pair.Erc20Address),
				},
			)
		}
	}()

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertERC20,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
				sdk.NewAttribute(types.AttributeKeyCosmosCoin, pair.Denom),
				sdk.NewAttribute(types.AttributeKeyERC20Token, msg.ContractAddress),
			),
		},
	)

	return &types.MsgConvertERC20Response{}, nil
}

// convertERC20NativeToken handles the erc20 conversion for a native erc20 token
// pair:
//   - escrow tokens on module account
//   - mint coins on bank module
//   - send minted coins to the receiver
//   - check if coin balance increased by amount
//   - check if token balance decreased by amount
//   - check for unexpected `Approval` event in logs
func (k Keeper) convertERC20NativeToken(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertERC20,
	receiver sdk.AccAddress,
	sender common.Address,
) (*types.MsgConvertERC20Response, error) {
	// NOTE: coin fields already validated
	coins := sdk.Coins{sdk.Coin{Denom: pair.Denom, Amount: msg.Amount}}
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	contract := pair.GetERC20Contract()
	balanceCoin := k.bankKeeper.GetBalance(ctx, receiver, pair.Denom)
	balanceToken := k.BalanceOf(ctx, erc20, contract, types.ModuleAddress)
	if balanceToken == nil {
		return nil, errorsmod.Wrap(types.ErrEVMCall, "failed to retrieve balance")
	}

	// Escrow tokens on module account
	transferData, err := erc20.Pack("transfer", types.ModuleAddress, msg.Amount.BigInt())
	if err != nil {
		return nil, err
	}

	res, err := k.CallEVMWithData(ctx, sender, &contract, transferData, true)
	if err != nil {
		return nil, err
	}

	// Check evm call response
	var unpackedRet types.ERC20BoolResponse
	if err := erc20.UnpackIntoInterface(&unpackedRet, "transfer", res.Ret); err != nil {
		return nil, err
	}

	if !unpackedRet.Value {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to execute transfer")
	}

	// Check expected escrow balance after transfer execution
	tokens := coins[0].Amount.BigInt()
	balanceTokenAfter := k.BalanceOf(ctx, erc20, contract, types.ModuleAddress)
	if balanceTokenAfter == nil {
		return nil, errorsmod.Wrap(types.ErrEVMCall, "failed to retrieve balance")
	}

	expToken := big.NewInt(0).Add(balanceToken, tokens)

	if r := balanceTokenAfter.Cmp(expToken); r != 0 {
		return nil, errorsmod.Wrapf(
			types.ErrBalanceInvariance,
			"invalid token balance - expected: %v, actual: %v",
			expToken, balanceTokenAfter,
		)
	}

	// Mint coins
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, err
	}

	// Send minted coins to the receiver
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, coins); err != nil {
		return nil, err
	}

	// Check expected receiver balance after transfer
	balanceCoinAfter := k.bankKeeper.GetBalance(ctx, receiver, pair.Denom)
	expCoin := balanceCoin.Add(coins[0])

	if ok := balanceCoinAfter.IsEqual(expCoin); !ok {
		return nil, errorsmod.Wrapf(
			types.ErrBalanceInvariance,
			"invalid coin balance - expected: %v, actual: %v",
			expCoin, balanceCoinAfter,
		)
	}

	// Check for unexpected `Approval` event in logs
	if err := k.monitorApprovalEvent(res); err != nil {
		return nil, err
	}

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"tx", "msg", "convert", "erc20", "total"},
			1,
			[]metrics.Label{
				telemetry.NewLabel("coin", pair.Denom),
				telemetry.NewLabel("erc20", pair.Erc20Address),
			},
		)

		if msg.Amount.IsInt64() {
			telemetry.IncrCounterWithLabels(
				[]string{"tx", "msg", "convert", "erc20", "amount", "total"},
				float32(msg.Amount.Int64()),
				[]metrics.Label{
					telemetry.NewLabel("denom", pair.Denom),
					telemetry.NewLabel("erc20", pair.Erc20Address),
				},
			)
		}
	}()

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertERC20,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
				sdk.NewAttribute(types.AttributeKeyCosmosCoin, pair.Denom),
				sdk.NewAttribute(types.AttributeKeyERC20Token, msg.ContractAddress),
			),
		},
	)

	return &types.MsgConvertERC20Response{}, nil
}

// convertCoinNativeERC20 handles the coin conversion for a native ERC20 token
// pair:
//   - escrow Coins on module account
//   - unescrow Tokens that have been previously escrowed with ConvertERC20 and send to receiver
//   - burn escrowed Coins
//   - check if token balance increased by amount
//   - check for unexpected `Approval` event in logs
func (k Keeper) convertCoinNativeERC20(
	ctx sdk.Context,
	pair types.TokenPair,
	msg *types.MsgConvertCoin,
	receiver common.Address,
	sender sdk.AccAddress,
) (*types.MsgConvertCoinResponse, error) {
	// NOTE: ignore validation from NewCoin constructor
	coins := sdk.Coins{msg.Coin}

	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	contract := pair.GetERC20Contract()
	balanceToken := k.BalanceOf(ctx, erc20, contract, receiver)
	if balanceToken == nil {
		return nil, errorsmod.Wrap(types.ErrEVMCall, "failed to retrieve balance")
	}

	// Escrow Coins on module account
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, coins); err != nil {
		return nil, errorsmod.Wrap(err, "failed to escrow coins")
	}

	// Unescrow Tokens and send to receiver
	res, err := k.CallEVM(ctx, erc20, types.ModuleAddress, contract, true, "transfer", receiver, msg.Coin.Amount.BigInt())
	if err != nil {
		return nil, err
	}

	// Check unpackedRet execution
	var unpackedRet types.ERC20BoolResponse
	if err := erc20.UnpackIntoInterface(&unpackedRet, "transfer", res.Ret); err != nil {
		return nil, err
	}

	if !unpackedRet.Value {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to execute unescrow tokens from user")
	}

	// Check expected Receiver balance after transfer execution
	tokens := msg.Coin.Amount.BigInt()
	balanceTokenAfter := k.BalanceOf(ctx, erc20, contract, receiver)
	if balanceTokenAfter == nil {
		return nil, errorsmod.Wrap(types.ErrEVMCall, "failed to retrieve balance")
	}

	exp := big.NewInt(0).Add(balanceToken, tokens)

	if r := balanceTokenAfter.Cmp(exp); r != 0 {
		return nil, errorsmod.Wrapf(
			types.ErrBalanceInvariance,
			"invalid token balance - expected: %v, actual: %v", exp, balanceTokenAfter,
		)
	}

	// Burn escrowed Coins
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to burn coins")
	}

	// Check for unexpected `Approval` event in logs
	if err := k.monitorApprovalEvent(res); err != nil {
		return nil, err
	}

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"tx", "msg", "convert", "coin", "total"},
			1,
			[]metrics.Label{
				telemetry.NewLabel("denom", pair.Denom),
				telemetry.NewLabel("erc20", pair.Erc20Address),
			},
		)

		if msg.Coin.Amount.IsInt64() {
			telemetry.IncrCounterWithLabels(
				[]string{"tx", "msg", "convert", "coin", "amount", "total"},
				float32(msg.Coin.Amount.Int64()),
				[]metrics.Label{
					telemetry.NewLabel("denom", pair.Denom),
					telemetry.NewLabel("erc20", pair.Erc20Address),
				},
			)
		}
	}()

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeConvertCoin,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Coin.Amount.String()),
				sdk.NewAttribute(types.AttributeKeyCosmosCoin, msg.Coin.Denom),
				sdk.NewAttribute(types.AttributeKeyERC20Token, pair.Erc20Address),
			),
		},
	)

	return &types.MsgConvertCoinResponse{}, nil
}

// SendCoinToEVM handles the conversion from Cosmos coin to EVM ERC20 and collects a fee
// This is only possible for whitelisted coin denoms that also have a registered ERC20 token
func (k Keeper) SendCoinToEVM(goCtx context.Context, msg *types.MsgSendCoinToEVM) (*types.MsgSendCoinToEVMResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Validate that denom or its ERC20 address is present on gasfree interop list; fail early if not.
	if err := k.validateGasfreeInteropToken(ctx, msg.Coin.Denom); err != nil {
		return nil, err
	}

	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	receiver := common.Address(sender.Bytes())

	_, err := k.ConvertCoin(goCtx, &types.MsgConvertCoin{
		Sender:   msg.Sender,
		Receiver: receiver.Hex(),
		Coin:     msg.Coin,
	})
	if err != nil {
		return nil, err
	}

	feesCollected, err := k.DeductGasfreeErc20Fee(ctx, sender, msg.Coin)
	if err != nil {
		err = errorsmod.Wrap(err, "unable to collect gasfree fees")
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{sdk.NewEvent(
		types.EventTypeSendCoinToEVM,
		sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
		sdk.NewAttribute(types.AttributeKeyReceiver, receiver.Hex()),
		sdk.NewAttribute(types.AttributeKeyCosmosCoin, msg.Coin.Denom),
		sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Coin.Amount.String()),
		sdk.NewAttribute(types.AttributeKeyFeesCollected, feesCollected.String()),
	)})
	k.Logger(ctx).Info("SendCoinToEVM executed",
		"sender", msg.Sender,
		"evm_receiver", receiver.Hex(),
		"denom", msg.Coin.Denom,
		"amount", msg.Coin.Amount.String(),
	)
	return &types.MsgSendCoinToEVMResponse{}, nil
}

// SendERC20ToCosmos handles the conversion from EVM ERC20 to Cosmos coin and collects a fee
// This is only possible for whitelisted ERC20s that also have a registered Cosmos coin
func (k Keeper) SendERC20ToCosmos(goCtx context.Context, msg *types.MsgSendERC20ToCosmos) (*types.MsgSendERC20ToCosmosResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Validate that denom or its ERC20 address is present on gasfree interop list; fail early if not.
	if err := k.validateGasfreeInteropToken(ctx, msg.Erc20); err != nil {
		return nil, err
	}

	senderEthAddress := common.HexToAddress(msg.Sender)
	senderAccAddress := sdk.AccAddress(senderEthAddress.Bytes())
	if err := sdk.VerifyAddressFormat(senderAccAddress); err != nil {
		return nil, errorsmod.Wrap(err, "invalid converted sender address from eip55")
	}

	feeBasisPoints, err := k.gasfreeKeeper.GetGasfreeErc20InteropFeeBasisPoints(ctx)
	if err != nil {
		return nil, err
	}
	feeAmount := altheacommon.CalculateBasisPointFee(msg.Amount, feeBasisPoints)
	totalAmount := msg.Amount.Add(feeAmount)

	_, err = k.ConvertERC20(goCtx, &types.MsgConvertERC20{
		Sender:          msg.Sender,
		Receiver:        senderAccAddress.String(),
		ContractAddress: msg.Erc20,
		Amount:          totalAmount,
	})
	if err != nil {
		return nil, err
	}

	pair, ok := k.GetTokenPair(ctx, k.GetTokenPairID(ctx, msg.Erc20))
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrTokenPairNotFound, "no token pair for ERC20 %s, this should be impossible since the token was already converted", msg.Erc20)
	}
	coin := sdk.NewCoin(pair.Denom, msg.Amount)
	feesCollected, err := k.DeductGasfreeErc20Fee(ctx, senderAccAddress, coin)
	if err != nil {
		err = errorsmod.Wrap(err, "unable to collect gasfree fees")
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{sdk.NewEvent(
		types.EventTypeSendERC20ToCosmos,
		sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
		sdk.NewAttribute(types.AttributeKeyERC20Token, msg.Erc20),
		sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
		sdk.NewAttribute(types.AttributeKeyFeesCollected, feesCollected.String()),
	)})
	k.Logger(ctx).Info("SendERC20ToCosmos executed",
		"sender", msg.Sender,
		"erc20", msg.Erc20,
		"amount", msg.Amount.String(),
		"cosmos_receiver", senderAccAddress.String(),
		"fees_collected", feesCollected.String(),
	)
	return &types.MsgSendERC20ToCosmosResponse{}, nil
}

// SendERC20ToCosmos handles the conversion from EVM ERC20 to Cosmos coin, queues an IBC transfer, and collects a fee
// This is only possible for whitelisted ERC20s that also have a registered Cosmos coin
func (k Keeper) SendERC20ToCosmosAndIBCTransfer(goCtx context.Context, msg *types.MsgSendERC20ToCosmosAndIBCTransfer) (*types.MsgSendERC20ToCosmosAndIBCTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Validate that denom or its ERC20 address is present on gasfree interop list; fail early if not.
	if err := k.validateGasfreeInteropToken(ctx, msg.Erc20); err != nil {
		return nil, err
	}

	senderEthAddress := common.HexToAddress(msg.Sender)
	senderAccAddress := sdk.AccAddress(senderEthAddress.Bytes())
	if err := sdk.VerifyAddressFormat(senderAccAddress); err != nil {
		return nil, errorsmod.Wrap(err, "invalid converted sender address from eip55")
	}

	feeBasisPoints, err := k.gasfreeKeeper.GetGasfreeErc20InteropFeeBasisPoints(ctx)
	if err != nil {
		return nil, err
	}
	feeAmount := altheacommon.CalculateBasisPointFee(msg.Amount, feeBasisPoints)
	totalAmount := msg.Amount.Add(feeAmount)

	_, err = k.ConvertERC20(goCtx, &types.MsgConvertERC20{
		Sender:          msg.Sender,
		Receiver:        senderAccAddress.String(),
		ContractAddress: msg.Erc20,
		Amount:          totalAmount,
	})
	if err != nil {
		return nil, err
	}

	// After successful conversion, perform an IBC transfer of the newly minted Cosmos coin
	sequence, err := k.initiateIBCTransfer(ctx, msg)
	if err != nil {
		return nil, err
	}
	pair, ok := k.GetTokenPair(ctx, k.GetTokenPairID(ctx, msg.Erc20))
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrTokenPairNotFound, "no token pair for ERC20 %s, this should be impossible since the token was already converted", msg.Erc20)
	}
	coin := sdk.NewCoin(pair.Denom, msg.Amount)
	feesCollected, err := k.DeductGasfreeErc20Fee(ctx, senderAccAddress, coin)
	if err != nil {
		err = errorsmod.Wrap(err, "unable to collect gasfree fees")
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{sdk.NewEvent(
		types.EventTypeSendERC20IBCTransfer,
		sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
		sdk.NewAttribute(types.AttributeKeyERC20Token, msg.Erc20),
		sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
		sdk.NewAttribute(types.AttributeKeyPort, msg.DestinationPort),
		sdk.NewAttribute(types.AttributeKeyChannel, msg.DestinationChannel),
		sdk.NewAttribute(types.AttributeKeyReceiver, msg.DestinationReceiver),
		sdk.NewAttribute(types.AttributeKeyFeesCollected, feesCollected.String()),
	)})
	k.Logger(ctx).Info("SendERC20ToCosmosAndIBCTransfer executed",
		"sender", msg.Sender,
		"erc20", msg.Erc20,
		"amount", msg.Amount.String(),
		"port", msg.DestinationPort,
		"channel", msg.DestinationChannel,
		"dest_receiver", msg.DestinationReceiver,
		"ibc_sequence", sequence,
		"fees_collected", feesCollected.String(),
	)

	return &types.MsgSendERC20ToCosmosAndIBCTransferResponse{}, nil
}

func (k Keeper) initiateIBCTransfer(ctx sdk.Context, msg *types.MsgSendERC20ToCosmosAndIBCTransfer) (uint64, error) {
	// Resolve sender (who received the converted coins)
	senderEth := common.HexToAddress(msg.Sender)
	senderAcc := sdk.AccAddress(senderEth.Bytes())

	// Validate port/channel
	if msg.DestinationPort == "" || msg.DestinationChannel == "" {
		return 0, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "destination port or channel cannot be empty")
	}

	// Re-derive the coin denom from the ERC20 address
	pair, ok := k.GetTokenPair(ctx, k.GetTokenPairID(ctx, msg.Erc20))
	if !ok {
		return 0, errorsmod.Wrapf(types.ErrTokenPairNotFound, "no token pair for ERC20 %s, this should be impossible since the token was already converted", msg.Erc20)
	}
	coin := sdk.NewCoin(pair.Denom, msg.Amount)
	if !coin.Amount.IsPositive() {
		return 0, errorsmod.Wrap(sdkerrors.ErrInvalidCoins, "amount must be positive for IBC transfer")
	}

	// Timeout (30 days)
	timeoutTimestamp := uint64(ctx.BlockTime().Add(30 * 24 * time.Hour).UnixNano())
	zeroHeight := ibcclienttypes.ZeroHeight()

	transferMsg := ibctransfertypes.NewMsgTransfer(
		msg.DestinationPort,
		msg.DestinationChannel,
		coin,
		senderAcc.String(),      // sender on this chain
		msg.DestinationReceiver, // receiver on the foreign chain
		zeroHeight,              // no timeout height
		timeoutTimestamp,        // timeout timestamp
		"Sent via MsgSendERC20ToCosmosAndIBCTransfer", // memo
	)

	resp, ibcErr := k.ibcTransferKeeper.Transfer(ctx, transferMsg)
	if ibcErr != nil {
		return 0, ibcErr
	}
	return resp.Sequence, nil
}

// DeductGasfreeErc20Fee will check and deduct the fee for the given sendAmount, based on the gasfree module's GasfreeErc20InteropFeeBasisPoints param value
// If the amount is insufficient for a fee to be collected (and feeBasisPoints > 0), an error is returned
func (k Keeper) DeductGasfreeErc20Fee(ctx sdk.Context, sender sdk.AccAddress, sendAmount sdk.Coin) (feeCollected *sdk.Coin, err error) {
	// Compute the minimum fees which must be paid
	feeBasisPoints, err := k.gasfreeKeeper.GetGasfreeErc20InteropFeeBasisPoints(ctx)
	if err != nil {
		return nil, err
	}
	collectedFee, err := altheacommon.DeductBasisPointFee(ctx, k.accountKeeper, k.bankKeeper, feeBasisPoints, sendAmount, sender)
	if err != nil {
		return nil, err
	}

	if collectedFee.IsZero() && feeBasisPoints > 0 {
		return nil, types.ErrInsufficientAmount
	}

	return &collectedFee, nil
}

// validateGasfreeInteropToken ensures either the denom or the mapped ERC20 address is present
// in the gasfree module's interop token list. Returns error if neither is found.
func (k Keeper) validateGasfreeInteropToken(ctx sdk.Context, token string) error {
	// Resolve token pair via denom
	pair, ok := k.GetTokenPair(ctx, k.GetTokenPairID(ctx, token))
	if !ok {
		return errorsmod.Wrapf(types.ErrTokenPairNotFound, "no token pair for denom %s", token)
	}
	list, err := k.gasfreeKeeper.GetGasfreeErc20InteropTokens(ctx)
	if err != nil {
		return err
	}
	allowed := false
	for _, entry := range list {
		lowercaseEntry, lowercaseDenom, lowercaseERC20 := strings.ToLower(entry), strings.ToLower(pair.Denom), strings.ToLower(pair.Erc20Address)
		if lowercaseEntry == lowercaseDenom {
			allowed = true
			break
		}
		if lowercaseEntry == lowercaseERC20 {
			allowed = true
			break
		}
	}
	if !allowed {
		return errorsmod.Wrapf(types.ErrGasfreeTokenNotAllowed, "token not on gasfree interop list: denom=%s erc20=%s", token, pair.Erc20Address)
	}
	return nil
}
