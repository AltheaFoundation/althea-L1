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
	OpsHandler                = govclient.NewProposalHandler(cli.NewOpsProposalCmd)
	OpsDisableTemplateHandler = govclient.NewProposalHandler(cli.NewOpsDisableTemplateCmd)
	OpsSetTemplateHandler     = govclient.NewProposalHandler(cli.NewOpsSetTemplateCmd)
	OpsRevisePoolHandler      = govclient.NewProposalHandler(cli.NewOpsRevisePoolCmd)
	OpsSetTakeRateHandler     = govclient.NewProposalHandler(cli.NewOpsSetTakeRateCmd)
	OpsResyncTakeRateHandler  = govclient.NewProposalHandler(cli.NewOpsResyncTakeRateCmd)
	OpsSetNewPoolLiqHandler   = govclient.NewProposalHandler(cli.NewOpsSetNewPoolLiqCmd)
	OpsPegPriceImproveHandler = govclient.NewProposalHandler(cli.NewOpsPegPriceImproveCmd)
	ExecuteContractHandler    = govclient.NewProposalHandler(cli.NewExecuteContractProposalCmd)
)
