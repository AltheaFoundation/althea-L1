package lockup

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	evmtypes "github.com/evmos/ethermint/x/evm/types"

	"github.com/AltheaFoundation/althea-L1/x/lockup/keeper"
	"github.com/AltheaFoundation/althea-L1/x/lockup/types"
	microtxtypes "github.com/AltheaFoundation/althea-L1/x/microtx/types"
)

func TestLockAnteHandler(t *testing.T) {
	// Test with the default of locked, only 0x0000.. is exempt, block bank's MsgSend and MsgMultiSend
	sdk.GetConfig().SetBech32PrefixForAccount("althea", "altheapub")
	input := keeper.CreateTestEnv(t)
	ctx := input.Context
	appCodec := keeper.MakeTestMarshaler()
	txCfg := authtx.NewTxConfig(appCodec, authtx.DefaultSignModes)
	keys := sdk.NewKVStoreKeys(types.StoreKey)
	subspace, _ := input.ParamsKeeper.GetSubspace(types.ModuleName)
	keeper := keeper.NewKeeper(
		appCodec, keys[types.StoreKey], subspace,
	)
	handler := NewLockupAnteHandler(keeper, appCodec)
	txFct := tx.Factory{}.WithTxConfig(txCfg).WithChainID("Gold-Chain")

	// Lock the chain
	keeper.SetChainLocked(ctx, true)
	// 0x0 and its ethermint interpretation
	lockExemptAddrs := []string{"0x0000000000000000000000000000000000000000", "althea1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq8p93tc"}
	keeper.SetLockExemptAddresses(ctx, lockExemptAddrs)
	keeper.SetLockedTokenDenoms(ctx, []string{"aalthea"})

	AnteHandlerLockedHappy(t, handler, keeper, ctx, txCfg, txFct)
	AnteHandlerLockedUnhappy(t, handler, keeper, ctx, txCfg, txFct)

	// Unlock the chain
	keeper.SetChainLocked(ctx, false)

	AnteHandlerUnlockedHappy(t, handler, keeper, ctx, txCfg, txFct)
}

// nolint: dupl
// Test successful messages on a locked chain
func AnteHandlerLockedHappy(t *testing.T, handler sdk.AnteHandler, keeper keeper.Keeper, ctx sdk.Context, txCfg client.TxConfig, txFct tx.Factory) {
	allowedMsgSendTx := GetAllowedMsgSendTx(keeper, ctx, txFct, txCfg)
	allowedCtx, allowedErr := handler(ctx, allowedMsgSendTx, false)
	assert.Equal(t, ctx, allowedCtx)
	assert.Nil(t, allowedErr)
	t.Log("Successful good MsgSend")

	allowedMultiSendTx := GetAllowedMultiSendTx(keeper, ctx, txFct, txCfg)
	allMultCtx, allMultErr := handler(ctx, allowedMultiSendTx, false)
	assert.Equal(t, ctx, allMultCtx)
	assert.Nil(t, allMultErr)
	t.Log("Successful good MsgMultiSend")

	unimportantTx := GetUnimportantTx(txFct, txCfg)
	// blocks a transaction coming from 0x1 but not one coming from 0x0.
	unimpCtx, unimpErr := handler(ctx, unimportantTx, false)
	assert.Equal(t, ctx, unimpCtx)
	assert.Nil(t, unimpErr)
	t.Log("Successful unimportant message")

	largeTx := GetAllowedLargeTx(keeper, ctx, txFct, txCfg)
	largeCtx, largeErr := handler(ctx, largeTx, false)
	assert.Equal(t, ctx, largeCtx)
	assert.Nil(t, largeErr)
	t.Log("Successful good large transaction")

	allowedMsgMicrotxTx := GetAllowedMsgMicrotxTx(keeper, ctx, txFct, txCfg)
	allMicrotxCtx, allMicrotxErr := handler(ctx, allowedMsgMicrotxTx, false)
	assert.Equal(t, ctx, allMicrotxCtx)
	assert.Nil(t, allMicrotxErr)
	t.Log("Successful good MsgMicrotx")

	allowedMsgTransferTx := GetAllowedMsgTransferTx(keeper, ctx, txFct, txCfg)
	allTransCtx, allTransErr := handler(ctx, allowedMsgTransferTx, false)
	assert.Equal(t, ctx, allTransCtx)
	assert.Nil(t, allTransErr)
	t.Log("Successful good MsgTransfer")

	allowedMsgEthereumTx := GetAllowedMsgEthereumTxTx(keeper, ctx, txFct, txCfg)
	allEvmCtx, allEvmErr := handler(ctx, allowedMsgEthereumTx, false)
	assert.Equal(t, ctx, allEvmCtx)
	assert.Nil(t, allEvmErr)
	t.Log("Successful good MsgEthereumTx")
}

// Test failing messages on a locked chain
func AnteHandlerLockedUnhappy(t *testing.T, handler sdk.AnteHandler, keeper keeper.Keeper, ctx sdk.Context, txCfg client.TxConfig, txFct tx.Factory) {
	unallowedMsgSendTx := GetUnallowedMsgSendTx(keeper, ctx, txFct, txCfg)
	// blocks a transaction coming from 0x11...
	unallowedCtx, unallowedErr := handler(ctx, unallowedMsgSendTx, false)
	assert.Equal(t, ctx, unallowedCtx)
	assert.NotNil(t, unallowedErr)
	t.Log("Successful bad MsgSend")

	unallowedMultiSendTx := GetUnallowedMultiSendTx(keeper, ctx, txFct, txCfg)
	// blocks a transaction coming from 0x11...
	unallMultCtx, unallMultErr := handler(ctx, unallowedMultiSendTx, false)
	assert.Equal(t, ctx, unallMultCtx)
	assert.NotNil(t, unallMultErr)
	t.Log("Successful bad MsgMultiSend")

	unallowedLargeTx := GetUnallowedLargeTx(keeper, ctx, txFct, txCfg)
	unallLargeCtx, unallLargeErr := handler(ctx, unallowedLargeTx, false)
	assert.Equal(t, ctx, unallLargeCtx)
	assert.NotNil(t, unallLargeErr)
	t.Log("Successful bad large Tx")

	unallowedMsgMicrotxTx := GetUnallowedMsgMicrotxTx(keeper, ctx, txFct, txCfg)
	unallMicrotxCtx, unallMicrotxErr := handler(ctx, unallowedMsgMicrotxTx, false)
	assert.Equal(t, ctx, unallMicrotxCtx)
	assert.NotNil(t, unallMicrotxErr)
	t.Log("Successful bad MsgMicrotx")

	unallowedMsgTransferTx := GetUnallowedMsgTransferTx(keeper, ctx, txFct, txCfg)
	unallTransCtx, unallTransErr := handler(ctx, unallowedMsgTransferTx, false)
	assert.Equal(t, ctx, unallTransCtx)
	assert.NotNil(t, unallTransErr)
	t.Log("Successful bad MsgTransfer")

	unallowedMsgEvmTx := GetUnallowedMsgEthereumTxTx(keeper, ctx, txFct, txCfg)
	unallEvmCtx, unallEvmErr := handler(ctx, unallowedMsgEvmTx, false)
	assert.Equal(t, ctx, unallEvmCtx)
	assert.NotNil(t, unallEvmErr)
	t.Log("Successful bad MsgEthereumTx")
}

// nolint: dupl
func AnteHandlerUnlockedHappy(t *testing.T, handler sdk.AnteHandler, keeper keeper.Keeper, ctx sdk.Context, txCfg client.TxConfig, txFct tx.Factory) {
	unallowedMsgSendTx := GetUnallowedMsgSendTx(keeper, ctx, txFct, txCfg)
	// blocks a transaction coming from 0x11...
	allowedCtx, allowedErr := handler(ctx, unallowedMsgSendTx, false)
	assert.Equal(t, ctx, allowedCtx)
	assert.Nil(t, allowedErr)
	t.Log("Successful bad MsgSend")

	unallowedMultiSendTx := GetUnallowedMultiSendTx(keeper, ctx, txFct, txCfg)
	// blocks a transaction coming from 0x11...
	allMultCtx, allMultErr := handler(ctx, unallowedMultiSendTx, false)
	assert.Equal(t, ctx, allMultCtx)
	assert.Nil(t, allMultErr)
	t.Log("Successful bad MsgMultiSend")

	unimportantTx := GetUnimportantTx(txFct, txCfg)
	unimpCtx, unimpErr := handler(ctx, unimportantTx, false)
	assert.Equal(t, ctx, unimpCtx)
	assert.Nil(t, unimpErr)
	t.Log("Successful unimportant message")

	largeTx := GetUnallowedLargeTx(keeper, ctx, txFct, txCfg)
	largeCtx, largeErr := handler(ctx, largeTx, false)
	assert.Equal(t, ctx, largeCtx)
	assert.Nil(t, largeErr)
	t.Log("Successful large bad Tx")

	unallowedMsgMicrotxTx := GetUnallowedMsgMicrotxTx(keeper, ctx, txFct, txCfg)
	unallMicrotxCtx, unallMicrotxErr := handler(ctx, unallowedMsgMicrotxTx, false)
	assert.Equal(t, ctx, unallMicrotxCtx)
	assert.Nil(t, unallMicrotxErr)
	t.Log("Successful bad MsgMicrotx")

	unallowedMsgTransferTx := GetUnallowedMsgTransferTx(keeper, ctx, txFct, txCfg)
	unallTransCtx, unallTransErr := handler(ctx, unallowedMsgTransferTx, false)
	assert.Equal(t, ctx, unallTransCtx)
	assert.Nil(t, unallTransErr)
	t.Log("Successful bad MsgTransfer")

	unallowedMsgEvmTx := GetUnallowedMsgEthereumTxTx(keeper, ctx, txFct, txCfg)
	unallEvmCtx, unallEvmErr := handler(ctx, unallowedMsgEvmTx, false)
	assert.Equal(t, ctx, unallEvmCtx)
	assert.Nil(t, unallEvmErr)
	t.Log("Successful bad MsgEthereumTx")
}

func GetAllowedMsgSendTx(keeper keeper.Keeper, ctx sdk.Context, txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	msgSend := GetAllowedMsgSend(keeper, ctx)
	txBld, err := txFct.BuildUnsignedTx(&msgSend)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", msgSend, err))
	}

	return txBld.GetTx()
}

func GetAllowedMsgSend(keeper keeper.Keeper, ctx sdk.Context) banktypes.MsgSend {
	// nolint: goconst
	fromAddr := "0x0000000000000000000000000000000000000000"
	exemptSet := keeper.GetLockExemptAddressesSet(ctx)
	if _, ok := exemptSet[fromAddr]; !ok {
		panic(fmt.Sprintf("The exemptSet has been changed, it needs to contain %v", fromAddr))
	}
	// nolint: goconst
	toAddr := "0x1111111111111111111111111111111111111111"
	amount := sdk.NewCoins(sdk.NewCoin("aalthea", sdk.NewInt(1000000000000000000)))
	return banktypes.MsgSend{FromAddress: fromAddr, ToAddress: toAddr, Amount: amount}
}

func GetAllowedLargeTx(keeper keeper.Keeper, ctx sdk.Context, txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	msgSend := GetAllowedMsgSend(keeper, ctx)
	multiSend := GetAllowedMultiSendMsg(keeper, ctx)
	unimportant := GetUnimportantMsg()
	txBld, err := txFct.BuildUnsignedTx(&msgSend, &multiSend, &msgSend, &multiSend, &unimportant, &unimportant)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", msgSend, err))
	}

	return txBld.GetTx()
}

func GetAllowedMultiSendTx(keeper keeper.Keeper, ctx sdk.Context, txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	multiSend := GetAllowedMultiSendMsg(keeper, ctx)
	txBld, err := txFct.BuildUnsignedTx(&multiSend)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", multiSend, err))
	}

	return txBld.GetTx()
}

func GetAllowedMultiSendMsg(keeper keeper.Keeper, ctx sdk.Context) banktypes.MsgMultiSend {
	fromAddr := "0x0000000000000000000000000000000000000000"
	exemptSet := keeper.GetLockExemptAddressesSet(ctx)
	if _, ok := exemptSet[fromAddr]; !ok {
		panic(fmt.Sprintf("The exemptSet has been changed, it needs to contain %v", fromAddr))
	}
	toAddr := "0x1111111111111111111111111111111111111111"
	amount := sdk.NewCoins(sdk.NewCoin("aalthea", sdk.NewInt(1000000000000000000)))
	inputs := []banktypes.Input{{Address: fromAddr, Coins: amount}}
	outputs := []banktypes.Output{{Address: toAddr, Coins: amount}}
	return banktypes.MsgMultiSend{Inputs: inputs, Outputs: outputs}
}

func GetUnimportantTx(txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	unimportantMsg := GetUnimportantMsg()
	txBld, err := txFct.BuildUnsignedTx(&unimportantMsg)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", unimportantMsg, err))
	}

	return txBld.GetTx()
}

func GetUnimportantMsg() stakingtypes.MsgCreateValidator {
	// nolint: exhaustruct
	return stakingtypes.MsgCreateValidator{
		// nolint: exhaustruct
		Description:      stakingtypes.Description{},
		DelegatorAddress: "0x0000000000000000000000000000000000000000",
		ValidatorAddress: "0x0000000000000000000000000000000000000000",
	}
}

func GetAllowedMsgTransferTx(keeper keeper.Keeper, ctx sdk.Context, txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	msgTransfer := GetAllowedMsgTransfer(keeper, ctx)
	txBld, err := txFct.BuildUnsignedTx(&msgTransfer)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", msgTransfer, err))
	}

	return txBld.GetTx()
}

func GetAllowedMsgTransfer(keeper keeper.Keeper, ctx sdk.Context) ibctransfertypes.MsgTransfer {
	fromAddr := "0x0000000000000000000000000000000000000000"
	exemptSet := keeper.GetLockExemptAddressesSet(ctx)
	if _, ok := exemptSet[fromAddr]; !ok {
		panic(fmt.Sprintf("The exemptSet has been changed, it needs to contain %v", fromAddr))
	}
	// nolint: goconst
	toAddr := "0x1111111111111111111111111111111111111111"
	amount := sdk.NewCoin("aalthea", sdk.NewInt(1000000000000000000))
	return ibctransfertypes.MsgTransfer{
		SourcePort:    "transfer",
		SourceChannel: "channel-5",
		Token:         amount,
		Sender:        fromAddr,
		Receiver:      toAddr,
		// nolint: exhaustruct
		TimeoutHeight:    ibcclienttypes.Height{},
		TimeoutTimestamp: 0, // We don't care about timestamp as it's generally avoided
		Memo:             "",
	}
}

func GetAllowedMsgMicrotxTx(keeper keeper.Keeper, ctx sdk.Context, txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	msgMicrotx := GetAllowedMsgMicrotx(keeper, ctx)
	txBld, err := txFct.BuildUnsignedTx(&msgMicrotx)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", msgMicrotx, err))
	}

	return txBld.GetTx()
}

func GetAllowedMsgMicrotx(keeper keeper.Keeper, ctx sdk.Context) microtxtypes.MsgMicrotx {
	fromAddr := "0x0000000000000000000000000000000000000000"
	exemptSet := keeper.GetLockExemptAddressesSet(ctx)
	if _, ok := exemptSet[fromAddr]; !ok {
		panic(fmt.Sprintf("The exemptSet has been changed, it needs to contain %v", fromAddr))
	}
	// nolint: goconst
	toAddr := "0x1111111111111111111111111111111111111111"
	amount := sdk.NewCoin("aalthea", sdk.NewInt(1000000000000000000))
	return microtxtypes.MsgMicrotx{
		Sender:   fromAddr,
		Receiver: toAddr,
		Amount:   amount,
	}
}

func GetAllowedMsgEthereumTx(keeper keeper.Keeper, ctx sdk.Context) evmtypes.MsgEthereumTx {
	fromAddr := "0x0000000000000000000000000000000000000000"
	exemptSet := keeper.GetLockExemptAddressesSet(ctx)
	if _, ok := exemptSet[fromAddr]; !ok {
		panic(fmt.Sprintf("The exemptSet has been changed, it needs to contain %v", fromAddr))
	}
	fromAddress := common.HexToAddress(fromAddr)
	return *evmtypes.NewTx(big.NewInt(1234), 0, &fromAddress, big.NewInt(1234), 2000000, big.NewInt(1234), big.NewInt(4567), big.NewInt(7890), []byte{}, nil)
}

func GetAllowedMsgEthereumTxTx(keeper keeper.Keeper, ctx sdk.Context, txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	msgEvmTx := GetAllowedMsgEthereumTx(keeper, ctx)
	fromAddr := "0x0000000000000000000000000000000000000000"
	msgEvmTx.From = fromAddr
	txBld, err := txFct.BuildUnsignedTx(&msgEvmTx)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", msgEvmTx, err))
	}

	return txBld.GetTx()
}

func GetUnallowedMsgSendTx(keeper keeper.Keeper, ctx sdk.Context, txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	fromAddr := "0x1111111111111111111111111111111111111111"
	exemptSet := keeper.GetLockExemptAddressesSet(ctx)
	if _, ok := exemptSet[fromAddr]; ok {
		panic(fmt.Sprintf("The exemptSet has been changed, it MUST NOT contain %v", fromAddr))
	}
	toAddr := "0x0000000000000000000000000000000000000000"
	amount := sdk.NewCoins(sdk.NewCoin("aalthea", sdk.NewInt(1000000000000000000)))
	msgSend := banktypes.MsgSend{FromAddress: fromAddr, ToAddress: toAddr, Amount: amount}
	txBld, err := txFct.BuildUnsignedTx(&msgSend)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", msgSend, err))
	}

	return txBld.GetTx()
}

func GetUnallowedMultiSendTx(keeper keeper.Keeper, ctx sdk.Context, txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	multiSend := GetUnallowedMultiSendMsg(keeper, ctx)
	txBld, err := txFct.BuildUnsignedTx(&multiSend)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", multiSend, err))
	}

	return txBld.GetTx()
}

func GetUnallowedMultiSendMsg(keeper keeper.Keeper, ctx sdk.Context) banktypes.MsgMultiSend {
	fromAddr := "0x1111111111111111111111111111111111111111"
	exemptSet := keeper.GetLockExemptAddressesSet(ctx)
	if _, ok := exemptSet[fromAddr]; ok {
		panic(fmt.Sprintf("The exemptSet has been changed, it MUST NOT contain %v", fromAddr))
	}
	toAddr := "0x0000000000000000000000000000000000000000"
	amount := sdk.NewCoins(sdk.NewCoin("aalthea", sdk.NewInt(1000000000000000000)))
	inputs := []banktypes.Input{{Address: fromAddr, Coins: amount}}
	outputs := []banktypes.Output{{Address: toAddr, Coins: amount}}
	return banktypes.MsgMultiSend{Inputs: inputs, Outputs: outputs}
}

func GetUnallowedLargeTx(keeper keeper.Keeper, ctx sdk.Context, txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	msgSend := GetAllowedMsgSend(keeper, ctx)
	multiSend := GetAllowedMultiSendMsg(keeper, ctx)
	badMultiSend := GetUnallowedMultiSendMsg(keeper, ctx)
	unimportant := GetUnimportantMsg()
	txBld, err := txFct.BuildUnsignedTx(&msgSend, &multiSend, &msgSend, &badMultiSend, &multiSend, &unimportant, &unimportant)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", msgSend, err))
	}

	return txBld.GetTx()
}

func GetUnallowedMsgTransferTx(keeper keeper.Keeper, ctx sdk.Context, txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	msgTransfer := GetUnallowedMsgTransfer(keeper, ctx)
	txBld, err := txFct.BuildUnsignedTx(&msgTransfer)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", msgTransfer, err))
	}

	return txBld.GetTx()
}

func GetUnallowedMsgTransfer(keeper keeper.Keeper, ctx sdk.Context) ibctransfertypes.MsgTransfer {
	fromAddr := "0x1111111111111111111111111111111111111111"
	exemptSet := keeper.GetLockExemptAddressesSet(ctx)
	if _, ok := exemptSet[fromAddr]; ok {
		panic(fmt.Sprintf("The exemptSet has been changed, it MUST NOT contain %v", fromAddr))
	}
	// nolint: goconst
	toAddr := "0x0000000000000000000000000000000000000000"
	amount := sdk.NewCoin("aalthea", sdk.NewInt(1000000000000000000))
	return ibctransfertypes.MsgTransfer{
		SourcePort:    "transfer",
		SourceChannel: "channel-5",
		Token:         amount,
		Sender:        fromAddr,
		Receiver:      toAddr,
		// nolint: exhaustruct
		TimeoutHeight:    ibcclienttypes.Height{},
		TimeoutTimestamp: 0, // We don't care about timestamp as it's generally avoided
		Memo:             "",
	}
}

func GetUnallowedMsgMicrotxTx(keeper keeper.Keeper, ctx sdk.Context, txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	msgMicrotx := GetUnallowedMsgMicrotx(keeper, ctx)
	txBld, err := txFct.BuildUnsignedTx(&msgMicrotx)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", msgMicrotx, err))
	}

	return txBld.GetTx()
}

func GetUnallowedMsgMicrotx(keeper keeper.Keeper, ctx sdk.Context) microtxtypes.MsgMicrotx {
	fromAddr := "0x1111111111111111111111111111111111111111"
	exemptSet := keeper.GetLockExemptAddressesSet(ctx)
	if _, ok := exemptSet[fromAddr]; ok {
		panic(fmt.Sprintf("The exemptSet has been changed, it MUST NOT contain %v", fromAddr))
	}
	// nolint: goconst
	toAddr := "0x0000000000000000000000000000000000000000"
	amount := sdk.NewCoin("aalthea", sdk.NewInt(1000000000000000000))
	return microtxtypes.MsgMicrotx{
		Sender:   fromAddr,
		Receiver: toAddr,
		Amount:   amount,
	}
}

func GetUnallowedMsgEthereumTx(keeper keeper.Keeper, ctx sdk.Context) evmtypes.MsgEthereumTx {
	fromAddr := "0x1111111111111111111111111111111111111111"
	exemptSet := keeper.GetLockExemptAddressesSet(ctx)
	if _, ok := exemptSet[fromAddr]; ok {
		panic(fmt.Sprintf("The exemptSet has been changed, it MUST NOT contain %v", fromAddr))
	}
	fromAddress := common.HexToAddress(fromAddr)
	return *evmtypes.NewTx(big.NewInt(1234), 0, &fromAddress, big.NewInt(1234), 2000000, big.NewInt(1234), big.NewInt(4567), big.NewInt(7890), []byte{}, nil)
}

func GetUnallowedMsgEthereumTxTx(keeper keeper.Keeper, ctx sdk.Context, txFct tx.Factory, txCfg client.TxConfig) sdk.Tx {
	msgEvmTx := GetUnallowedMsgEthereumTx(keeper, ctx)
	fromAddr := "0x1111111111111111111111111111111111111111"
	msgEvmTx.From = fromAddr
	txBld, err := txFct.BuildUnsignedTx(&msgEvmTx)
	if err != nil {
		panic(fmt.Sprintf("Unable to build unsigned transaction containing %v: %v", msgEvmTx, err))
	}

	return txBld.GetTx()
}
