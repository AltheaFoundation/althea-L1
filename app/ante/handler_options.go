package ante

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	ibcante "github.com/cosmos/ibc-go/v6/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v6/modules/core/keeper"

	ethante "github.com/evmos/ethermint/app/ante"
	ethtypes "github.com/evmos/ethermint/types"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarketkeeper "github.com/evmos/ethermint/x/feemarket/keeper"

	"github.com/AltheaFoundation/althea-L1/x/gasfree"
	gasfreekeeper "github.com/AltheaFoundation/althea-L1/x/gasfree/keeper"
	microtxkeeper "github.com/AltheaFoundation/althea-L1/x/microtx/keeper"
)

// HandlerOptions defines the list of module keepers required to run the canto
// AnteHandler decorators.
type HandlerOptions struct {
	AccountKeeper          AccountKeeper
	BankKeeper             evmtypes.BankKeeper
	IBCKeeper              *ibckeeper.Keeper
	FeeMarketKeeper        feemarketkeeper.Keeper
	EvmKeeper              *evmkeeper.Keeper
	FeegrantKeeper         ante.FeegrantKeeper
	SignModeHandler        authsigning.SignModeHandler
	SigGasConsumer         func(meter sdk.GasMeter, sig signing.SignatureV2, params authtypes.Params) error
	MaxTxGasWanted         uint64
	ExtensionOptionChecker ante.ExtensionOptionChecker
	TxFeeChecker           ante.TxFeeChecker
	DisabledAuthzMsgs      []string
	Cdc                    codec.BinaryCodec
	GasfreeKeeper          *gasfreekeeper.Keeper
	MicrotxKeeper          *microtxkeeper.Keeper
}

// Validate checks if the keepers are defined
func (options HandlerOptions) Validate() error {
	if options.AccountKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "account keeper is required for AnteHandler")
	}
	if options.BankKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if options.SignModeHandler == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for ante builder")
	}
	if options.EvmKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "evm keeper is required for AnteHandler")
	}
	if options.GasfreeKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "gasfree keeper is required for AnteHandler")
	}
	if options.MicrotxKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "microtx keeper is required for AnteHandler")
	}
	return nil
}

// newCosmosAnteHandler creates the default ante handler for Ethereum transactions
func newEthAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		ethante.NewEthSetUpContextDecorator(options.EvmKeeper),                         // outermost AnteDecorator. SetUpContext must be called first
		ethante.NewEthMempoolFeeDecorator(options.EvmKeeper),                           // Check eth effective gas price against the node's minimal-gas-prices config
		ethante.NewEthMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper), // Check eth effective gas price against the global MinGasPrice
		ethante.NewEthValidateBasicDecorator(options.EvmKeeper),
		ethante.NewEthSigVerificationDecorator(options.EvmKeeper),
		NewEthAccountVerificationDecorator(options.AccountKeeper, options.EvmKeeper),
		NewEthSetPubkeyDecorator(options.AccountKeeper, options.EvmKeeper),
		NewSetAccountTypeDecorator(options.AccountKeeper, options.EvmKeeper.AccountProtoFn),
		ethante.NewCanTransferDecorator(options.EvmKeeper),
		ethante.NewEthGasConsumeDecorator(options.EvmKeeper, options.MaxTxGasWanted),
		ethante.NewEthIncrementSenderSequenceDecorator(options.AccountKeeper),
		ethante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
		ethante.NewEthEmitEventDecorator(options.EvmKeeper), // emit eth tx hash and index at the very last ante handler.
	)
}

func CosmosExtensionOptionChecker(any *codectypes.Any) bool {
	a := ethtypes.HasDynamicFeeExtensionOption(any)
	b := hasEIP712ExtensionOption(any)

	return a || b
}

func hasEIP712ExtensionOption(any *codectypes.Any) bool {
	_, ok := any.GetCachedValue().(*ethtypes.ExtensionOptionsWeb3Tx)
	return ok
}

// newCosmosAnteHandler creates the default ante handler for Cosmos transactions
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		ethante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		ethante.NewAuthzLimiterDecorator(options.DisabledAuthzMsgs),
		ante.NewSetUpContextDecorator(),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		// Gasfree txs ignore the min gas price requirement
		gasfree.NewSelectiveBypassDecorator(*options.GasfreeKeeper, ethante.NewMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper)),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		// Gasfree txs do not have fees deducted the normal way, their fees will be deducted separately
		gasfree.NewSelectiveBypassDecorator(*options.GasfreeKeeper, ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, nil)),
		// Charge gas fees for gasfree messages
		NewChargeGasfreeFeesDecorator(options.AccountKeeper, *options.GasfreeKeeper, *options.MicrotxKeeper),
		NewValidatorCommissionDecorator(options.Cdc),
		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		ethante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
		NewSetAccountTypeDecorator(options.AccountKeeper, options.EvmKeeper.AccountProtoFn),
	)
}

// newCosmosAnteHandlerEip712 creates the ante handler for transactions signed with EIP712
func newCosmosAnteHandlerEip712(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		ethante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		ethante.NewAuthzLimiterDecorator(options.DisabledAuthzMsgs),
		ante.NewSetUpContextDecorator(),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ethante.NewMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, nil),
		NewValidatorCommissionDecorator(options.Cdc),
		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		// Note: signature verification uses EIP instead of the cosmos signature validator
		ethante.NewLegacyEip712SigVerificationDecorator(options.AccountKeeper, options.SignModeHandler, ""), // Pass no chain id to have it parsed from the Cosmos chain id
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		ethante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
		NewSetAccountTypeDecorator(options.AccountKeeper, options.EvmKeeper.AccountProtoFn),
	)
}
