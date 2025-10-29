package ante

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	ibckeeper "github.com/cosmos/ibc-go/v6/modules/core/keeper"

	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarketkeeper "github.com/evmos/ethermint/x/feemarket/keeper"

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
	DisabledGroupMsgs      []string
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
