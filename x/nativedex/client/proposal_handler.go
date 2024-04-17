package client

import (
	"github.com/althea-net/althea-L1/x/nativedex/client/cli"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var (
	UpgradeProxyHandler       = govclient.NewProposalHandler(cli.NewUpgradeProxyProposalCmd, nil)
	CollectTreasuryHandler    = govclient.NewProposalHandler(cli.NewCollectTreasuryProposalCmd, nil)
	SetTreasuryHandler        = govclient.NewProposalHandler(cli.NewSetTreasuryProposalCmd, nil)
	AuthorityTransferHandler  = govclient.NewProposalHandler(cli.NewAuthorityTransferProposalCmd, nil)
	HotPathOpenHandler        = govclient.NewProposalHandler(cli.NewHotPathOpenProposalCmd, nil)
	SetSafeModeHandler        = govclient.NewProposalHandler(cli.NewSetSafeModeProposalCmd, nil)
	TransferGovernanceHandler = govclient.NewProposalHandler(cli.NewTransferGovernanceProposalCmd, nil)
)
