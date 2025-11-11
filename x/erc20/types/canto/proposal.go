// Package canto provides backward compatibility for legacy Canto ERC20 proposal types.
//
// This package contains type definitions for proposals that were originally stored on-chain
// with canto.erc20.v1 type URLs before the Cardinal upgrade. When the module was renamed
// from "canto" to "althea", these historical proposals could no longer be deserialized.
//
// These types enable the codec to properly deserialize historical governance proposals
// without requiring a state migration. The types mirror the original Canto proposal
// definitions and implement the same govv1beta1.Content interface.
//
// Note: This package is for backward compatibility only. New proposals should use the
// types defined in the parent package (github.com/AltheaFoundation/althea-L1/x/erc20/types).
//
// The legacy types can be removed only after performing a state migration to convert
// all historical proposals to use the new type URLs.
package canto

import (
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ethermint "github.com/evmos/ethermint/types"
)

const (
	// RouterKey is the module name router key
	RouterKey = "erc20"
)

// Ensure the Canto proposal types implement the gov Content interface.
// These legacy types correspond to the following current types in the parent package:
//   - RegisterCoinProposal          -> types.RegisterCoinProposal
//   - RegisterERC20Proposal         -> types.RegisterERC20Proposal
//   - ToggleTokenConversionProposal -> types.ToggleTokenConversionProposal
//
// The main difference is the protobuf package namespace (canto.erc20.v1 vs althea.erc20.v1).
//
//nolint:exhaustruct
var (
	_ govv1beta1.Content = &RegisterCoinProposal{}
	_ govv1beta1.Content = &RegisterERC20Proposal{}
	_ govv1beta1.Content = &ToggleTokenConversionProposal{}
)

const (
	ProposalTypeRegisterCoin          string = "RegisterCoin"
	ProposalTypeRegisterERC20         string = "RegisterERC20"
	ProposalTypeToggleTokenConversion string = "ToggleTokenConversion"
)

// ProposalRoute returns router key for RegisterCoinProposal
func (*RegisterCoinProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for RegisterCoinProposal
func (*RegisterCoinProposal) ProposalType() string {
	return ProposalTypeRegisterCoin
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *RegisterCoinProposal) ValidateBasic() error {
	if err := rtbp.Metadata.Validate(); err != nil {
		return err
	}

	if err := ibctransfertypes.ValidateIBCDenom(rtbp.Metadata.Base); err != nil {
		return err
	}

	if err := validateIBCVoucherMetadata(rtbp.Metadata); err != nil {
		return err
	}

	return govv1beta1.ValidateAbstract(rtbp)
}

// validateIBCVoucherMetadata checks that the coin metadata fields are consistent
// with an IBC voucher denomination. This is a copy of the validation function
// from the parent types package to avoid import cycles.
func validateIBCVoucherMetadata(metadata banktypes.Metadata) error {
	// Check ibc/ denom
	denomSplit := strings.SplitN(metadata.Base, "/", 2)

	if denomSplit[0] == metadata.Base && strings.TrimSpace(metadata.Base) != "" {
		// Not IBC
		return nil
	}

	if len(denomSplit) != 2 || denomSplit[0] != ibctransfertypes.DenomPrefix {
		// NOTE: should be unaccessible (covered on ValidateIBCDenom)
		return fmt.Errorf("invalid metadata. %s denomination should be prefixed with the format 'ibc/", metadata.Base)
	}

	return nil
}

// ProposalRoute returns router key for RegisterERC20Proposal
func (*RegisterERC20Proposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for RegisterERC20Proposal
func (*RegisterERC20Proposal) ProposalType() string {
	return ProposalTypeRegisterERC20
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *RegisterERC20Proposal) ValidateBasic() error {
	if err := ethermint.ValidateAddress(rtbp.Erc20Address); err != nil {
		return errorsmod.Wrap(err, "ERC20 address")
	}
	return govv1beta1.ValidateAbstract(rtbp)
}

// ProposalRoute returns router key for ToggleTokenConversionProposal
func (*ToggleTokenConversionProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for ToggleTokenConversionProposal
func (*ToggleTokenConversionProposal) ProposalType() string {
	return ProposalTypeToggleTokenConversion
}

// ValidateBasic performs a stateless check of the proposal fields
func (ttcp *ToggleTokenConversionProposal) ValidateBasic() error {
	// check if the token is a hex address, if not, check if it is a valid SDK denom
	if err := ethermint.ValidateAddress(ttcp.Token); err != nil {
		if err := sdk.ValidateDenom(ttcp.Token); err != nil {
			return err
		}
	}

	return govv1beta1.ValidateAbstract(ttcp)
}
