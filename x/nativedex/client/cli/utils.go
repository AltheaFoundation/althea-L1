package cli

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/AltheaFoundation/althea-L1/x/nativedex/types"
)

type GenericProposalSetup struct {
	ClientCtx   client.Context
	Title       string
	Description string
	Deposit     sdk.Coins
	From        sdk.AccAddress
}

func GenericProposalCmdSetup(cmd *cobra.Command) (setup GenericProposalSetup, err error) {
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return
	}

	title, err := cmd.Flags().GetString(cli.FlagTitle)
	if err != nil {
		return
	}

	description, err := cmd.Flags().GetString(cli.FlagDescription)
	if err != nil {
		return
	}

	depositStr, err := cmd.Flags().GetString(cli.FlagDeposit)
	if err != nil {
		return
	}

	deposit, err := sdk.ParseCoinsNormalized(depositStr)
	if err != nil {
		return
	}

	from := clientCtx.GetFromAddress()

	setup = GenericProposalSetup{
		ClientCtx:   clientCtx,
		Title:       title,
		Description: description,
		Deposit:     deposit,
		From:        from,
	}
	return
}

func GenericProposalCmdBroadcast(cmd *cobra.Command, clientCtx client.Context, content govtypes.Content, deposit sdk.Coins, from sdk.AccAddress) error {

	msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
	if err != nil {
		return err
	}

	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
}

func AddGenericProposalCommandFlags(cmd *cobra.Command) {
	cmd.Flags().String(cli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "1aalthea", "deposit of proposal")
	if err := cmd.MarkFlagRequired(cli.FlagTitle); err != nil {
		panic(sdkerrors.Wrap(err, "No title provided"))
	}
	if err := cmd.MarkFlagRequired(cli.FlagDescription); err != nil {
		panic(sdkerrors.Wrap(err, "No description provided"))
	}
	if err := cmd.MarkFlagRequired(cli.FlagDeposit); err != nil {
		panic(sdkerrors.Wrap(err, "No deposit provided"))
	}
}

// Parse Metadata structs from the input files

func ParseUpgradeProxyMetadata(cdc codec.JSONCodec, metadataFile string) (types.UpgradeProxyMetadata, error) {
	// nolint: exhaustruct
	propMetaData := types.UpgradeProxyMetadata{}

	contents, err := os.ReadFile(filepath.Clean(metadataFile))
	if err != nil {
		return propMetaData, err
	}

	if err = json.Unmarshal(contents, &propMetaData); err != nil {
		return propMetaData, err
	}

	return propMetaData, nil
}

func ParseCollectTreasuryMetadata(cdc codec.JSONCodec, metadataFile string) (types.CollectTreasuryMetadata, error) {
	// nolint: exhaustruct
	propMetaData := types.CollectTreasuryMetadata{}

	contents, err := os.ReadFile(filepath.Clean(metadataFile))
	if err != nil {
		return propMetaData, err
	}

	if err = json.Unmarshal(contents, &propMetaData); err != nil {
		return propMetaData, err
	}

	return propMetaData, nil
}

func ParseSetTreasuryMetadata(cdc codec.JSONCodec, metadataFile string) (types.SetTreasuryMetadata, error) {
	// nolint: exhaustruct
	propMetaData := types.SetTreasuryMetadata{}

	contents, err := os.ReadFile(filepath.Clean(metadataFile))
	if err != nil {
		return propMetaData, err
	}

	if err = json.Unmarshal(contents, &propMetaData); err != nil {
		return propMetaData, err
	}

	return propMetaData, nil
}

func ParseAuthorityTransferMetadata(cdc codec.JSONCodec, metadataFile string) (types.AuthorityTransferMetadata, error) {
	// nolint: exhaustruct
	propMetaData := types.AuthorityTransferMetadata{}

	contents, err := os.ReadFile(filepath.Clean(metadataFile))
	if err != nil {
		return propMetaData, err
	}

	if err = json.Unmarshal(contents, &propMetaData); err != nil {
		return propMetaData, err
	}

	return propMetaData, nil
}

func ParseHotPathOpenMetadata(cdc codec.JSONCodec, metadataFile string) (types.HotPathOpenMetadata, error) {
	// nolint: exhaustruct
	propMetaData := types.HotPathOpenMetadata{}

	contents, err := os.ReadFile(filepath.Clean(metadataFile))
	if err != nil {
		return propMetaData, err
	}

	if err = json.Unmarshal(contents, &propMetaData); err != nil {
		return propMetaData, err
	}

	return propMetaData, nil
}

func ParseSetSafeModeMetadata(cdc codec.JSONCodec, metadataFile string) (types.SetSafeModeMetadata, error) {
	// nolint: exhaustruct
	propMetaData := types.SetSafeModeMetadata{}

	contents, err := os.ReadFile(filepath.Clean(metadataFile))
	if err != nil {
		return propMetaData, err
	}

	if err = json.Unmarshal(contents, &propMetaData); err != nil {
		return propMetaData, err
	}

	return propMetaData, nil
}

func ParseTransferGovernanceMetadata(cdc codec.JSONCodec, metadataFile string) (types.TransferGovernanceMetadata, error) {
	// nolint: exhaustruct
	propMetaData := types.TransferGovernanceMetadata{}

	contents, err := os.ReadFile(filepath.Clean(metadataFile))
	if err != nil {
		return propMetaData, err
	}

	if err = json.Unmarshal(contents, &propMetaData); err != nil {
		return propMetaData, err
	}

	return propMetaData, nil
}

func ParseOpsMetadata(cdc codec.JSONCodec, metadataFile string) (types.OpsMetadata, error) {
	// nolint: exhaustruct
	propMetaData := types.OpsMetadata{}

	contents, err := os.ReadFile(filepath.Clean(metadataFile))
	if err != nil {
		return propMetaData, err
	}

	if err = json.Unmarshal(contents, &propMetaData); err != nil {
		return propMetaData, err
	}

	return propMetaData, nil
}
