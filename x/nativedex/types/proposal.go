package types

import (
	"github.com/ethereum/go-ethereum/common"

	errorsmod "cosmossdk.io/errors"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	ProposalTypeUpgradeProxy       string = "UpgradeProxy"
	ProposalTypeCollectTreasury    string = "CollectTreasury"
	ProposalTypeSetTreasury        string = "SetTreasury"
	ProposalTypeAuthorityTransfer  string = "AuthorityTransfer"
	ProposalTypeHotPathOpen        string = "HotPathOpen"
	ProposalTypeSetSafeMode        string = "SetSafeMode"
	ProposalTypeTransferGovernance string = "TransferGovernance"
	ProposalTypeOps                string = "Ops"
	ProposalTypeExecuteContract    string = "ExecuteContract"
	MaxDescriptionLength           int    = 1000
	MaxTitleLength                 int    = 140
)

var AcceptableCallpathIndexes []uint64 = []uint64{1, 2, 3, 4, 5, 6, 7, 3500, 9999}

// nolint: exhaustruct
var (
	_ govv1beta1.Content = &UpgradeProxyProposal{}
	_ govv1beta1.Content = &CollectTreasuryProposal{}
	_ govv1beta1.Content = &SetTreasuryProposal{}
	_ govv1beta1.Content = &AuthorityTransferProposal{}
	_ govv1beta1.Content = &HotPathOpenProposal{}
	_ govv1beta1.Content = &SetSafeModeProposal{}
	_ govv1beta1.Content = &TransferGovernanceProposal{}
	_ govv1beta1.Content = &ExecuteContractProposal{}
)

// Register Compound Proposal type as a valid proposal type in goveranance module
// nolint: exhaustruct
func init() {
	govv1beta1.RegisterProposalType(ProposalTypeUpgradeProxy)
	govv1beta1.RegisterProposalType(ProposalTypeCollectTreasury)
	govv1beta1.RegisterProposalType(ProposalTypeSetTreasury)
	govv1beta1.RegisterProposalType(ProposalTypeAuthorityTransfer)
	govv1beta1.RegisterProposalType(ProposalTypeHotPathOpen)
	govv1beta1.RegisterProposalType(ProposalTypeSetSafeMode)
	govv1beta1.RegisterProposalType(ProposalTypeTransferGovernance)
	govv1beta1.RegisterProposalType(ProposalTypeOps)
	govv1beta1.RegisterProposalType(ProposalTypeExecuteContract)
}

func NewUpgradeProxyProposal(title, description string, md UpgradeProxyMetadata) govv1beta1.Content {
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
	if err := govv1beta1.ValidateAbstract(p); err != nil {
		return err
	}

	md := p.GetMetadata()
	if !common.IsHexAddress(md.CallpathAddress) {
		return errorsmod.Wrap(ErrInvalidEvmAddress, "invalid callpath address")
	}

	if md.CallpathIndex == 0 {
		return ErrInvalidCallpath
	}

	return nil
}

func NewCollectTreasuryProposal(title, description string, md CollectTreasuryMetadata, inSafeMode bool) govv1beta1.Content {
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
	if err := govv1beta1.ValidateAbstract(p); err != nil {
		return err
	}

	md := p.GetMetadata()
	if !common.IsHexAddress(md.TokenAddress) {
		return errorsmod.Wrap(ErrInvalidEvmAddress, "invalid token address")
	}

	return nil
}

func NewSetTreasuryProposal(title, description string, md SetTreasuryMetadata, inSafeMode bool) govv1beta1.Content {
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
	if err := govv1beta1.ValidateAbstract(p); err != nil {
		return err
	}

	md := p.GetMetadata()
	if !common.IsHexAddress(md.TreasuryAddress) {
		return errorsmod.Wrap(ErrInvalidEvmAddress, "invalid treasury address")
	}

	return nil
}

func NewAuthorityTransferProposal(title, description string, md AuthorityTransferMetadata, inSafeMode bool) govv1beta1.Content {
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
	if err := govv1beta1.ValidateAbstract(p); err != nil {
		return err
	}

	md := p.GetMetadata()
	if !common.IsHexAddress(md.AuthAddress) {
		return errorsmod.Wrap(ErrInvalidEvmAddress, "invalid auth address")
	}

	return nil
}

func NewHotPathOpenProposal(title, description string, md HotPathOpenMetadata, inSafeMode bool) govv1beta1.Content {
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
	if err := govv1beta1.ValidateAbstract(p); err != nil {
		return err
	}

	// The only check to perform here is that the type is valid, which is already a given
	_ = p.GetMetadata()

	return nil
}

func NewSetSafeModeProposal(title, description string, md SetSafeModeMetadata, inSafeMode bool) govv1beta1.Content {
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
	if err := govv1beta1.ValidateAbstract(p); err != nil {
		return err
	}

	// The only check to perform here is that the type is valid, which is already a given
	_ = p.GetMetadata()

	return nil
}

func NewTransferGovernanceProposal(title, description string, md TransferGovernanceMetadata) govv1beta1.Content {
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
	if err := govv1beta1.ValidateAbstract(p); err != nil {
		return err
	}

	md := p.GetMetadata()

	if !common.IsHexAddress(md.Ops) {
		return errorsmod.Wrap(ErrInvalidEvmAddress, "invalid ops address")
	}

	if !common.IsHexAddress(md.Emergency) {
		return errorsmod.Wrap(ErrInvalidEvmAddress, "invalid emergency address")
	}

	return nil
}

func NewOpsProposal(title, description string, md OpsMetadata) govv1beta1.Content {
	return &OpsProposal{
		Title:       title,
		Description: description,
		Metadata:    md,
	}
}

func (*OpsProposal) ProposalRoute() string { return RouterKey }

func (*OpsProposal) ProposalType() string {
	return ProposalTypeOps
}

func (p *OpsProposal) ValidateBasic() error {
	if err := govv1beta1.ValidateAbstract(p); err != nil {
		return err
	}

	md := p.GetMetadata()

	if md.Callpath == 0 {
		return ErrInvalidCallpath
	}

	if len(md.CmdArgs) == 0 {
		return errorsmod.Wrap(govtypes.ErrInvalidProposalContent, "cmd args has zero length")
	}

	return nil
}

func NewExecuteContractProposal(title, description string, md ExecuteContractMetadata) govv1beta1.Content {
	return &ExecuteContractProposal{
		Title:       title,
		Description: description,
		Metadata:    md,
	}
}

func (*ExecuteContractProposal) ProposalRoute() string { return RouterKey }

func (*ExecuteContractProposal) ProposalType() string {
	return ProposalTypeExecuteContract
}

func (p *ExecuteContractProposal) ValidateBasic() error {
	if err := govv1beta1.ValidateAbstract(p); err != nil {
		return err
	}

	md := p.GetMetadata()

	if !common.IsHexAddress(md.ContractAddress) {
		return errorsmod.Wrap(ErrInvalidEvmAddress, "invalid contract address")
	}

	if len(md.Data) == 0 {
		return errorsmod.Wrap(govtypes.ErrInvalidProposalContent, "data has zero length")
	}

	// Validate that data is a valid hex string
	if len(md.Data) < 2 || md.Data[0:2] != "0x" {
		return errorsmod.Wrap(govtypes.ErrInvalidProposalContent, "data must be a hex string starting with 0x")
	}

	return nil
}
