package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/version"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/AltheaFoundation/althea-L1/x/nativedex/types"
)

// GetTxCmd bundles all the subcmds together so they appear under `gravity tx`
func GetTxCmd(storeKey string) *cobra.Command {
	// nolint: exhaustruct
	nativedexTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "nativedex transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	nativedexTxCmd.AddCommand([]*cobra.Command{
		NewUpgradeProxyProposalCmd(),
		NewCollectTreasuryProposalCmd(),
		NewSetTreasuryProposalCmd(),
		NewAuthorityTransferProposalCmd(),
		NewHotPathOpenProposalCmd(),
		NewSetSafeModeProposalCmd(),
		NewTransferGovernanceProposalCmd(),
		NewOpsProposalCmd(),
	}...)

	return nativedexTxCmd
}

// NewUpgradeProxyProposalCmd implements the command to submit a UpgradeProxyProposal
// nolint: dupl
func NewUpgradeProxyProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "upgrade-proxy [metadata]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit an UpgradeProxy proposal",
		Long: `Submit a proposal to upgrade the native DEX contracts with a deployed contract.
Upon passing, the new contract will be installed as the callpath index on the native DEX contract.
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-proposal upgrade-proxy <path/to/metadata.json> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"CallpathAddress": "<hex address>",
	"CallpathIndex": <uint index>,
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {

			setup, err := GenericProposalCmdSetup(cmd)
			if err != nil {
				return sdkerrors.Wrap(err, "invalid arguments to command")
			}
			var clientCtx, title, description, deposit, from = setup.ClientCtx, setup.Title, setup.Description, setup.Deposit, setup.From

			propMetaData, err := ParseUpgradeProxyMetadata(clientCtx.Codec, args[0])
			if err != nil {
				return sdkerrors.Wrap(err, "Failure to parse JSON object")
			}

			content := types.NewUpgradeProxyProposal(title, description, propMetaData)

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, deposit, from)
		},
	}

	AddGenericProposalCommandFlags(cmd)
	return cmd
}

// NewCollectTreasuryProposalCmd implements the command to submit a CollectTreasuryProposal
// nolint: dupl
func NewCollectTreasuryProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "collect-treasury [metadata] [in-safe-mode bool]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a CollectTreasury proposal",
		Long: `Submit a proposal to distribute the native DEX protocol take for a single token to the registered 'treasury_' address.
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-proposal collect-treasury <path/to/metadata.json> <in-safe-mode> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"TokenAddress": "<hex address>",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {

			setup, err := GenericProposalCmdSetup(cmd)
			if err != nil {
				return sdkerrors.Wrap(err, "invalid arguments to command")
			}
			var clientCtx, title, description, deposit, from = setup.ClientCtx, setup.Title, setup.Description, setup.Deposit, setup.From

			propMetaData, err := ParseCollectTreasuryMetadata(clientCtx.Codec, args[0])
			if err != nil {
				return sdkerrors.Wrap(err, "Failure to parse JSON object")
			}

			inSafeMode, err := strconv.ParseBool(args[1])
			if err != nil {
				return sdkerrors.Wrap(err, "invalid in-safe-mode")
			}

			content := types.NewCollectTreasuryProposal(title, description, propMetaData, inSafeMode)

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, deposit, from)
		},
	}

	AddGenericProposalCommandFlags(cmd)
	return cmd
}

// NewSetTreasuryProposalCmd implements the command to submit a SetTreasuryProposal
// nolint: dupl
func NewSetTreasuryProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "set-treasury [metadata] [in-safe-mode bool]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a SetTreasury proposal",
		Long: `Submit a proposal to update the 'treasury_' address on the native DEX.
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-proposal set-treasury <path/to/metadata.json> <in-safe-mode> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"TreasuryAddress": "<hex address>",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {

			setup, err := GenericProposalCmdSetup(cmd)
			if err != nil {
				return sdkerrors.Wrap(err, "invalid arguments to command")
			}
			var clientCtx, title, description, deposit, from = setup.ClientCtx, setup.Title, setup.Description, setup.Deposit, setup.From

			propMetaData, err := ParseSetTreasuryMetadata(clientCtx.Codec, args[0])
			if err != nil {
				return sdkerrors.Wrap(err, "Failure to parse JSON object")
			}

			inSafeMode, err := strconv.ParseBool(args[1])
			if err != nil {
				return sdkerrors.Wrap(err, "invalid in-safe-mode")
			}

			content := types.NewSetTreasuryProposal(title, description, propMetaData, inSafeMode)

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, deposit, from)
		},
	}

	AddGenericProposalCommandFlags(cmd)
	return cmd
}

// NewAuthorityTransferProposalCmd implements the command to submit a AuthorityTransferProposal
// nolint: dupl
func NewAuthorityTransferProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "authority-transfer [metadata] [in-safe-mode bool]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a AuthorityTransfer proposal",
		Long: `Submit a proposal to transfer the 'authority_' address on the native DEX, effectively replacing the CrocPolicy contract with another one.
WARNING: THIS MAY HAVE SEVERE UNINTENDED CONSEQUENCES. ENSURE THE NEW CONTRACT IS COMPATIBLE WITH THIS MODULE BEFORE PROPOSING.
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-proposal authority-transfer <path/to/metadata.json> <in-safe-mode> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"AuthAddress": "<hex address>",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {

			setup, err := GenericProposalCmdSetup(cmd)
			if err != nil {
				return sdkerrors.Wrap(err, "invalid arguments to command")
			}
			var clientCtx, title, description, deposit, from = setup.ClientCtx, setup.Title, setup.Description, setup.Deposit, setup.From

			propMetaData, err := ParseAuthorityTransferMetadata(clientCtx.Codec, args[0])
			if err != nil {
				return sdkerrors.Wrap(err, "Failure to parse JSON object")
			}

			inSafeMode, err := strconv.ParseBool(args[1])
			if err != nil {
				return sdkerrors.Wrap(err, "invalid in-safe-mode")
			}

			content := types.NewAuthorityTransferProposal(title, description, propMetaData, inSafeMode)

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, deposit, from)
		},
	}

	AddGenericProposalCommandFlags(cmd)
	return cmd
}

// NewHotPathOpenProposalCmd implements the command to submit a HotPathOpenProposal
// nolint: dupl
func NewHotPathOpenProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "hot-path-open [metadata] [in-safe-mode bool]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a HotPathOpen proposal",
		Long: `Submit a proposal to enable or disable calling swap() directly on the native DEX.
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-proposal hot-path-open <path/to/metadata.json> <in-safe-mode> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"Open": "<true/false>",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {

			setup, err := GenericProposalCmdSetup(cmd)
			if err != nil {
				return sdkerrors.Wrap(err, "invalid arguments to command")
			}
			var clientCtx, title, description, deposit, from = setup.ClientCtx, setup.Title, setup.Description, setup.Deposit, setup.From

			propMetaData, err := ParseHotPathOpenMetadata(clientCtx.Codec, args[0])
			if err != nil {
				return sdkerrors.Wrap(err, "Failure to parse JSON object")
			}

			inSafeMode, err := strconv.ParseBool(args[1])
			if err != nil {
				return sdkerrors.Wrap(err, "invalid in-safe-mode")
			}

			content := types.NewHotPathOpenProposal(title, description, propMetaData, inSafeMode)

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, deposit, from)
		},
	}

	AddGenericProposalCommandFlags(cmd)
	return cmd
}

// NewSetSafeModeProposalCmd implements the command to submit a SetSafeModeProposal
// nolint: dupl
func NewSetSafeModeProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "set-safe-mode [metadata] [in-safe-mode bool]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a SetSafeMode proposal",
		Long: `Submit a proposal to lock down the native DEX, or unlock it once it has been locked.
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-proposal set-safe-mode <path/to/metadata.json> <in-safe-mode> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"LockDex": "<true/false>",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {

			setup, err := GenericProposalCmdSetup(cmd)
			if err != nil {
				return sdkerrors.Wrap(err, "invalid arguments to command")
			}
			var clientCtx, title, description, deposit, from = setup.ClientCtx, setup.Title, setup.Description, setup.Deposit, setup.From

			propMetaData, err := ParseSetSafeModeMetadata(clientCtx.Codec, args[0])
			if err != nil {
				return sdkerrors.Wrap(err, "Failure to parse JSON object")
			}

			inSafeMode, err := strconv.ParseBool(args[1])
			if err != nil {
				return sdkerrors.Wrap(err, "invalid in-safe-mode")
			}

			content := types.NewSetSafeModeProposal(title, description, propMetaData, inSafeMode)

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, deposit, from)
		},
	}

	AddGenericProposalCommandFlags(cmd)
	return cmd
}

// NewTransferGovernanceProposalCmd implements the command to submit a TransferGovernanceProposal
// nolint: dupl
func NewTransferGovernanceProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "transfer-governance [metadata]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a TransferGovernance proposal",
		Long:  `Submit a proposal to set the CrocPolicy Ops and Emergency governance roles.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-proposal transfer-governance <path/to/metadata.json> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"Ops": "<hex address>",
	"Emergency": "<hex address>",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {

			setup, err := GenericProposalCmdSetup(cmd)
			if err != nil {
				return sdkerrors.Wrap(err, "invalid arguments to command")
			}
			var clientCtx, title, description, deposit, from = setup.ClientCtx, setup.Title, setup.Description, setup.Deposit, setup.From

			propMetaData, err := ParseTransferGovernanceMetadata(clientCtx.Codec, args[0])
			if err != nil {
				return sdkerrors.Wrap(err, "Failure to parse JSON object")
			}

			content := types.NewTransferGovernanceProposal(title, description, propMetaData)

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, deposit, from)
		},
	}

	AddGenericProposalCommandFlags(cmd)
	return cmd
}

// NewOpsProposalCmd implements the command to submit a OpsProposal
// nolint: dupl
func NewOpsProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "ops [metadata]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit an Ops proposal",
		Long:  `Submit a proposal to perform a non-sudo protocolCmd() call on the native DEX`,
		Example: fmt.Sprintf(`$ %s tx gov submit-proposal ops <path/to/metadata.json> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"Callpath": "3",
	"CmdArgs": "[<ABI Encoded arguments for the protocolCmd() call>]",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {

			setup, err := GenericProposalCmdSetup(cmd)
			if err != nil {
				return sdkerrors.Wrap(err, "invalid arguments to command")
			}
			var clientCtx, title, description, deposit, from = setup.ClientCtx, setup.Title, setup.Description, setup.Deposit, setup.From

			propMetaData, err := ParseOpsMetadata(clientCtx.Codec, args[0])
			if err != nil {
				return sdkerrors.Wrap(err, "Failure to parse JSON object")
			}

			content := types.NewOpsProposal(title, description, propMetaData)

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, deposit, from)
		},
	}

	AddGenericProposalCommandFlags(cmd)
	return cmd
}
