package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"

	"github.com/AltheaFoundation/althea-L1/x/nativedex/client/cli"
)

var (
	UpgradeProxyHandler       = govclient.NewProposalHandler(cli.NewUpgradeProxyProposalCmd)
	CollectTreasuryHandler    = govclient.NewProposalHandler(cli.NewCollectTreasuryProposalCmd)
	SetTreasuryHandler        = govclient.NewProposalHandler(cli.NewSetTreasuryProposalCmd)
	AuthorityTransferHandler  = govclient.NewProposalHandler(cli.NewAuthorityTransferProposalCmd)
	HotPathOpenHandler        = govclient.NewProposalHandler(cli.NewHotPathOpenProposalCmd)
	SetSafeModeHandler        = govclient.NewProposalHandler(cli.NewSetSafeModeProposalCmd)
	TransferGovernanceHandler = govclient.NewProposalHandler(cli.NewTransferGovernanceProposalCmd)
)
