package types

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

// constants
const (
	ProposalTypeRegisterCoin          string = "RegisterCoin"
	ProposalTypeRegisterERC20         string = "RegisterERC20"
	ProposalTypeToggleTokenConversion string = "ToggleTokenConversion" // #nosec
)

// Implements Proposal Interface
var (
	//nolint: exhaustruct
	_ govv1beta1.Content = &RegisterCoinProposal{}
	//nolint: exhaustruct
	_ govv1beta1.Content = &RegisterERC20Proposal{}
	//nolint: exhaustruct
	_ govv1beta1.Content = &ToggleTokenConversionProposal{}
)

func init() {
	govv1beta1.RegisterProposalType(ProposalTypeRegisterCoin)
	govv1beta1.RegisterProposalType(ProposalTypeRegisterERC20)
	govv1beta1.RegisterProposalType(ProposalTypeToggleTokenConversion)
}

// CreateDenomDescription generates a string with the coin description
func CreateDenomDescription(address string) string {
	return fmt.Sprintf("Cosmos coin token representation of %s", address)
}

// CreateDenom generates a string the module name plus the address to avoid conflicts with names staring with a number
func CreateDenom(address string) string {
	return fmt.Sprintf("%s/%s", ModuleName, address)
}

// NewRegisterCoinProposal returns new instance of RegisterCoinProposal
func NewRegisterCoinProposal(title, description string, coinMetadata banktypes.Metadata) govv1beta1.Content {
	return &RegisterCoinProposal{
		Title:       title,
		Description: description,
		Metadata:    coinMetadata,
	}
}

// ProposalRoute returns router key for this proposal
func (*RegisterCoinProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
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
// with an IBC voucher denomination.
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

// ValidateErc20Denom checks if a denom is a valid erc20/
// denomination
func ValidateErc20Denom(denom string) error {
	denomSplit := strings.SplitN(denom, "/", 2)

	if len(denomSplit) != 2 || denomSplit[0] != ModuleName {
		return fmt.Errorf("invalid denom. %s denomination should be prefixed with the format 'erc20/", denom)
	}

	return ethermint.ValidateAddress(denomSplit[1])
}

// NewRegisterERC20Proposal returns new instance of RegisterERC20Proposal
func NewRegisterERC20Proposal(title, description, erc20Addr string) govv1beta1.Content {
	return &RegisterERC20Proposal{
		Title:        title,
		Description:  description,
		Erc20Address: erc20Addr,
	}
}

// ProposalRoute returns router key for this proposal
func (*RegisterERC20Proposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
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

// NewToggleTokenConversionProposal returns new instance of ToggleTokenConversionProposal
func NewToggleTokenConversionProposal(title, description string, token string) govv1beta1.Content {
	return &ToggleTokenConversionProposal{
		Title:       title,
		Description: description,
		Token:       token,
	}
}

// ProposalRoute returns router key for this proposal
func (*ToggleTokenConversionProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*ToggleTokenConversionProposal) ProposalType() string {
	return ProposalTypeToggleTokenConversion
}

// ValidateBasic performs a stateless check of the proposal fields
func (ttcp *ToggleTokenConversionProposal) ValidateBasic() error {
	// check if the token is a hex address, if not, check if it is a valid SDK
	// denom
	if err := ethermint.ValidateAddress(ttcp.Token); err != nil {
		if err := sdk.ValidateDenom(ttcp.Token); err != nil {
			return err
		}
	}

	return govv1beta1.ValidateAbstract(ttcp)
}
