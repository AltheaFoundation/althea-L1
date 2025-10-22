package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	erc20types "github.com/AltheaFoundation/althea-L1/x/erc20/types"
	microtxtypes "github.com/AltheaFoundation/althea-L1/x/microtx/types"
)

// DefaultGenesisState creates a simple GenesisState suitible for testing
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

func DefaultParams() *Params {
	return &Params{
		GasFreeMessageTypes: []string{
			// nolint: exhaustruct
			sdk.MsgTypeURL(&microtxtypes.MsgMicrotx{}),
			// nolint: exhaustruct
			sdk.MsgTypeURL(&erc20types.MsgSendCoinToEVM{}),
			// nolint: exhaustruct
			sdk.MsgTypeURL(&erc20types.MsgSendERC20ToCosmos{}),
			// nolint: exhaustruct
			sdk.MsgTypeURL(&erc20types.MsgSendERC20ToCosmosAndIBCTransfer{}),
		},
		GasFreeErc20InteropTokens:         []string{},
		GasFreeErc20InteropFeeBasisPoints: 100, // 1%
	}
}

func (s GenesisState) ValidateBasic() error {
	if s.Params == nil {
		return ErrInvalidParams
	}
	if err := ValidateGasFreeMessageTypes(s.Params.GasFreeMessageTypes); err != nil {
		return errorsmod.Wrap(err, "Invalid GasFreeMessageTypes GenesisState")
	}
	return nil
}

func ValidateGasFreeMessageTypes(i interface{}) error {
	_, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid gas free message types type: %T", i)
	}

	return nil
}

func ValidateGasFreeErc20InteropTokens(i interface{}) error {
	_, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid gas free erc20 interop tokens type: %T", i)
	}

	for _, entry := range i.([]string) {
		if entry == "" {
			return fmt.Errorf("erc20 interop token cannot be empty string")
		}
		denomErr := sdk.ValidateDenom(entry)
		erc20Ok := common.IsHexAddress(entry)
		if denomErr != nil && !erc20Ok {
			return fmt.Errorf("erc20 interop token must be a valid cosmos denom or erc20 address: %s", entry)
		}
	}

	return nil
}

func ValidateGasFreeErc20InteropFeeBasisPoints(i interface{}) error {
	basisPoints, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid gas free erc20 interop fee basis points type (expect uint64): %T", i)
	}
	if basisPoints > 10000 {
		return fmt.Errorf("erc20 interop fee basis points cannot be greater than 10000 (100%%), got %d", basisPoints)
	}
	return nil
}

// ParamKeyTable for auth module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{
		GasFreeMessageTypes:               []string{},
		GasFreeErc20InteropTokens:         []string{},
		GasFreeErc20InteropFeeBasisPoints: 100,
	})
}

func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(GasFreeMessageTypesKey, &p.GasFreeMessageTypes, ValidateGasFreeMessageTypes),
		paramtypes.NewParamSetPair(GasFreeErc20InteropTokensKey, &p.GasFreeErc20InteropTokens, ValidateGasFreeErc20InteropTokens),
		paramtypes.NewParamSetPair(GasFreeErc20InteropFeeBasisPointsKey, &p.GasFreeErc20InteropFeeBasisPoints, ValidateGasFreeErc20InteropFeeBasisPoints),
	}
}
