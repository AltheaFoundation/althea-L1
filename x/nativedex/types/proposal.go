package types

import (
	"github.com/ethereum/go-ethereum/common"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalTypeUpgradeProxy       string = "UpgradeProxy"
	ProposalTypeCollectTreasury    string = "CollectTreasury"
	ProposalTypeSetTreasury        string = "SetTreasury"
	ProposalTypeAuthorityTransfer  string = "AuthorityTransfer"
	ProposalTypeHotPathOpen        string = "HotPathOpen"
	ProposalTypeSetSafeMode        string = "SetSafeMode"
	ProposalTypeTransferGovernance string = "TransferGovernance"
	MaxDescriptionLength           int    = 1000
	MaxTitleLength                 int    = 140
)

var AcceptableCallpathIndexes []uint64 = []uint64{1, 2, 3, 4, 5, 6, 7, 3500, 9999}

// nolint: exhaustruct
var (
	_ govtypes.Content = &UpgradeProxyProposal{}
	_ govtypes.Content = &CollectTreasuryProposal{}
	_ govtypes.Content = &SetTreasuryProposal{}
	_ govtypes.Content = &AuthorityTransferProposal{}
	_ govtypes.Content = &HotPathOpenProposal{}
	_ govtypes.Content = &SetSafeModeProposal{}
	_ govtypes.Content = &TransferGovernanceProposal{}
)

// Register Compound Proposal type as a valid proposal type in goveranance module
// nolint: exhaustruct
func init() {
	govtypes.RegisterProposalType(ProposalTypeUpgradeProxy)
	govtypes.RegisterProposalTypeCodec(&UpgradeProxyProposal{}, "nativedex/UpgradeProxyProposal")
	govtypes.RegisterProposalType(ProposalTypeCollectTreasury)
	govtypes.RegisterProposalTypeCodec(&CollectTreasuryProposal{}, "nativedex/CollectTreasuryProposal")
	govtypes.RegisterProposalType(ProposalTypeSetTreasury)
	govtypes.RegisterProposalTypeCodec(&SetTreasuryProposal{}, "nativedex/SetTreasuryProposal")
	govtypes.RegisterProposalType(ProposalTypeAuthorityTransfer)
	govtypes.RegisterProposalTypeCodec(&AuthorityTransferProposal{}, "nativedex/AuthorityTransferProposal")
	govtypes.RegisterProposalType(ProposalTypeHotPathOpen)
	govtypes.RegisterProposalTypeCodec(&HotPathOpenProposal{}, "nativedex/HotPathOpenProposal")
	govtypes.RegisterProposalType(ProposalTypeSetSafeMode)
	govtypes.RegisterProposalTypeCodec(&SetSafeModeProposal{}, "nativedex/SetSafeModeProposal")
	govtypes.RegisterProposalType(ProposalTypeTransferGovernance)
	govtypes.RegisterProposalTypeCodec(&TransferGovernanceProposal{}, "nativedex/TransferGovernanceProposal")
}

func NewUpgradeProxyProposal(title, description string, md UpgradeProxyMetadata) govtypes.Content {
	return &UpgradeProxyProposal{
		Title:       title,
		Description: description,
		Metadata:    md,
	}
}

func (*UpgradeProxyProposal) ProposalRoute() string { return RouterKey }

func (*UpgradeProxyProposal) ProposalType() string {
	return ProposalTypeUpgradeProxy
}

func (p *UpgradeProxyProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}

	md := p.GetMetadata()
	if !common.IsHexAddress(md.CallpathAddress) {
		return sdkerrors.Wrap(ErrInvalidEvmAddress, "invalid callpath address")
	}

	if md.CallpathIndex == 0 {
		return ErrInvalidCallpath
	}

	return nil
}

func NewCollectTreasuryProposal(title, description string, md CollectTreasuryMetadata, inSafeMode bool) govtypes.Content {
	return &CollectTreasuryProposal{
		Title:       title,
		Description: description,
		Metadata:    md,
		InSafeMode:  inSafeMode,
	}
}

func (*CollectTreasuryProposal) ProposalRoute() string { return RouterKey }

func (*CollectTreasuryProposal) ProposalType() string {
	return ProposalTypeCollectTreasury
}

func (p *CollectTreasuryProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}

	md := p.GetMetadata()
	if !common.IsHexAddress(md.TokenAddress) {
		return sdkerrors.Wrap(ErrInvalidEvmAddress, "invalid token address")
	}

	return nil
}

func NewSetTreasuryProposal(title, description string, md SetTreasuryMetadata, inSafeMode bool) govtypes.Content {
	return &SetTreasuryProposal{
		Title:       title,
		Description: description,
		Metadata:    md,
		InSafeMode:  inSafeMode,
	}
}

func (*SetTreasuryProposal) ProposalRoute() string { return RouterKey }

func (*SetTreasuryProposal) ProposalType() string {
	return ProposalTypeSetTreasury
}

func (p *SetTreasuryProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}

	md := p.GetMetadata()
	if !common.IsHexAddress(md.TreasuryAddress) {
		return sdkerrors.Wrap(ErrInvalidEvmAddress, "invalid treasury address")
	}

	return nil
}

func NewAuthorityTransferProposal(title, description string, md AuthorityTransferMetadata, inSafeMode bool) govtypes.Content {
	return &AuthorityTransferProposal{
		Title:       title,
		Description: description,
		Metadata:    md,
		InSafeMode:  inSafeMode,
	}
}

func (*AuthorityTransferProposal) ProposalRoute() string { return RouterKey }

func (*AuthorityTransferProposal) ProposalType() string {
	return ProposalTypeAuthorityTransfer
}

func (p *AuthorityTransferProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}

	md := p.GetMetadata()
	if !common.IsHexAddress(md.AuthAddress) {
		return sdkerrors.Wrap(ErrInvalidEvmAddress, "invalid auth address")
	}

	return nil
}

func NewHotPathOpenProposal(title, description string, md HotPathOpenMetadata, inSafeMode bool) govtypes.Content {
	return &HotPathOpenProposal{
		Title:       title,
		Description: description,
		Metadata:    md,
		InSafeMode:  inSafeMode,
	}
}

func (*HotPathOpenProposal) ProposalRoute() string { return RouterKey }

func (*HotPathOpenProposal) ProposalType() string {
	return ProposalTypeHotPathOpen
}

func (p *HotPathOpenProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}

	// The only check to perform here is that the type is valid, which is already a given
	_ = p.GetMetadata()

	return nil
}

func NewSetSafeModeProposal(title, description string, md SetSafeModeMetadata, inSafeMode bool) govtypes.Content {
	return &SetSafeModeProposal{
		Title:       title,
		Description: description,
		Metadata:    md,
		InSafeMode:  inSafeMode,
	}
}

func (*SetSafeModeProposal) ProposalRoute() string { return RouterKey }

func (*SetSafeModeProposal) ProposalType() string {
	return ProposalTypeSetSafeMode
}

func (p *SetSafeModeProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}

	// The only check to perform here is that the type is valid, which is already a given
	_ = p.GetMetadata()

	return nil
}

func NewTransferGovernanceProposal(title, description string, md TransferGovernanceMetadata) govtypes.Content {
	return &TransferGovernanceProposal{
		Title:       title,
		Description: description,
		Metadata:    md,
	}
}

func (*TransferGovernanceProposal) ProposalRoute() string { return RouterKey }

func (*TransferGovernanceProposal) ProposalType() string {
	return ProposalTypeTransferGovernance
}

func (p *TransferGovernanceProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}

	md := p.GetMetadata()

	if !common.IsHexAddress(md.Ops) {
		return sdkerrors.Wrap(ErrInvalidEvmAddress, "invalid ops address")
	}

	if !common.IsHexAddress(md.Emergency) {
		return sdkerrors.Wrap(ErrInvalidEvmAddress, "invalid emergency address")
	}

	return nil
}
