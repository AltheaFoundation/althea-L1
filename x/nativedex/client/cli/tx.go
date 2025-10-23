package cli

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/AltheaFoundation/althea-L1/contracts"
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
		NewOpsDisableTemplateCmd(),
		NewOpsSetTemplateCmd(),
		NewOpsRevisePoolCmd(),
		NewOpsSetTakeRateCmd(),
		NewOpsSetRelayerTakeRateCmd(),
		NewOpsResyncTakeRateCmd(),
		NewOpsSetNewPoolLiqCmd(),
		NewOpsPegPriceImproveCmd(),
	}...)

	return nativedexTxCmd
}

// NewUpgradeProxyProposalCmd implements the command to submit a UpgradeProxyProposal
// nolint: dupl
func NewUpgradeProxyProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "upgrade-proxy [initial-deposit] [title] [description] [metadata]",
		Args:  cobra.ExactArgs(4),
		Short: "Submit an UpgradeProxy proposal",
		Long: `Submit a proposal to upgrade the native DEX contracts with a deployed contract.
Upon passing, the new contract will be installed as the callpath index on the native DEX contract.
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-legacy-proposal <deposit> <title> <description> <path/to/metadata.json> --from=<key_or_address> --chain-id=<chain-id>

Where metadata.json contains (example):

{
	"callpath_address": "<hex address>",
	"callpath_index": <uint index>,
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return errorsmod.Wrap(err, "bad initial deposit amount")
			}
			title := args[1]
			description := args[2]
			metadataFile := args[3]
			if len(initialDeposit) != 1 {
				return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
			}
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			propMetaData, err := ParseUpgradeProxyMetadata(clientCtx.Codec, metadataFile)
			if err != nil {
				return errorsmod.Wrap(err, "Failure to parse JSON object")
			}
			content := types.NewUpgradeProxyProposal(title, description, propMetaData)
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cosmosAddr := cliCtx.GetFromAddress()
			return GenericProposalCmdBroadcast(cmd, clientCtx, content, initialDeposit, cosmosAddr)
		},
	}

	// AddGenericProposalCommandFlags(cmd)
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCollectTreasuryProposalCmd implements the command to submit a CollectTreasuryProposal
// nolint: dupl
func NewCollectTreasuryProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "collect-treasury [initial-deposit] [title] [description] [metadata] [in-safe-mode bool]",
		Args:  cobra.ExactArgs(5),
		Short: "Submit a CollectTreasury proposal",
		Long: `Submit a proposal to distribute the native DEX protocol take for a single token to the registered 'treasury_' address.
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-legacy-proposal collect-treasury <path/to/metadata.json> <in-safe-mode> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"token_address": "<hex address>",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return errorsmod.Wrap(err, "bad initial deposit amount")
			}
			title := args[1]
			description := args[2]
			metadataFile := args[3]
			inSafeModeStr := args[4]
			if len(initialDeposit) != 1 {
				return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
			}
			inSafeMode, err := strconv.ParseBool(inSafeModeStr)
			if err != nil {
				return errorsmod.Wrap(err, "invalid in-safe-mode")
			}
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cosmosAddr := clientCtx.GetFromAddress()
			propMetaData, err := ParseCollectTreasuryMetadata(clientCtx.Codec, metadataFile)
			if err != nil {
				return errorsmod.Wrap(err, "Failure to parse JSON object")
			}
			content := types.NewCollectTreasuryProposal(title, description, propMetaData, inSafeMode)

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, initialDeposit, cosmosAddr)
		},
	}

	// AddGenericProposalCommandFlags(cmd)
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSetTreasuryProposalCmd implements the command to submit a SetTreasuryProposal
// nolint: dupl
func NewSetTreasuryProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "set-treasury [initial-deposit] [title] [description] [metadata] [in-safe-mode bool]",
		Args:  cobra.ExactArgs(5),
		Short: "Submit a SetTreasury proposal",
		Long: `Submit a proposal to update the 'treasury_' address on the native DEX.
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-legacy-proposal set-treasury <path/to/metadata.json> <in-safe-mode> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"treasury_address": "<hex address>",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return errorsmod.Wrap(err, "bad initial deposit amount")
			}
			title := args[1]
			description := args[2]
			metadataFile := args[3]
			inSafeModeStr := args[4]
			if len(initialDeposit) != 1 {
				return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
			}
			inSafeMode, err := strconv.ParseBool(inSafeModeStr)
			if err != nil {
				return errorsmod.Wrap(err, "invalid in-safe-mode")
			}
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			propMetaData, err := ParseSetTreasuryMetadata(clientCtx.Codec, metadataFile)
			if err != nil {
				return errorsmod.Wrap(err, "Failure to parse JSON object")
			}
			content := types.NewSetTreasuryProposal(title, description, propMetaData, inSafeMode)
			cosmosAddr := clientCtx.GetFromAddress()

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, initialDeposit, cosmosAddr)
		},
	}

	// AddGenericProposalCommandFlags(cmd)
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAuthorityTransferProposalCmd implements the command to submit a AuthorityTransferProposal
// nolint: dupl
func NewAuthorityTransferProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "authority-transfer [initial-deposit] [title] [description] [metadata] [in-safe-mode bool]",
		Args:  cobra.ExactArgs(5),
		Short: "Submit a AuthorityTransfer proposal",
		Long: `Submit a proposal to transfer the 'authority_' address on the native DEX, effectively replacing the CrocPolicy contract with another one.
WARNING: THIS MAY HAVE SEVERE UNINTENDED CONSEQUENCES. ENSURE THE NEW CONTRACT IS COMPATIBLE WITH THIS MODULE BEFORE PROPOSING.
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-legacy-proposal authority-transfer <path/to/metadata.json> <in-safe-mode> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"auth_address": "<hex address>",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return errorsmod.Wrap(err, "bad initial deposit amount")
			}
			title := args[1]
			description := args[2]
			metadataFile := args[3]
			inSafeModeStr := args[4]
			if len(initialDeposit) != 1 {
				return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
			}
			inSafeMode, err := strconv.ParseBool(inSafeModeStr)
			if err != nil {
				return errorsmod.Wrap(err, "invalid in-safe-mode")
			}
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			propMetaData, err := ParseAuthorityTransferMetadata(clientCtx.Codec, metadataFile)
			if err != nil {
				return errorsmod.Wrap(err, "Failure to parse JSON object")
			}
			content := types.NewAuthorityTransferProposal(title, description, propMetaData, inSafeMode)
			cosmosAddr := clientCtx.GetFromAddress()

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, initialDeposit, cosmosAddr)
		},
	}

	// AddGenericProposalCommandFlags(cmd)
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewHotPathOpenProposalCmd implements the command to submit a HotPathOpenProposal
// nolint: dupl
func NewHotPathOpenProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "hot-path-open [initial-deposit] [title] [description] [metadata] [in-safe-mode bool]",
		Args:  cobra.ExactArgs(5),
		Short: "Submit a HotPathOpen proposal",
		Long: `Submit a proposal to enable or disable calling swap() directly on the native DEX.
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-legacy-proposal hot-path-open <path/to/metadata.json> <in-safe-mode> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"open": "<true/false>",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return errorsmod.Wrap(err, "bad initial deposit amount")
			}
			title := args[1]
			description := args[2]
			metadataFile := args[3]
			inSafeModeStr := args[4]
			if len(initialDeposit) != 1 {
				return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
			}
			inSafeMode, err := strconv.ParseBool(inSafeModeStr)
			if err != nil {
				return errorsmod.Wrap(err, "invalid in-safe-mode")
			}
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			propMetaData, err := ParseHotPathOpenMetadata(clientCtx.Codec, metadataFile)
			if err != nil {
				return errorsmod.Wrap(err, "Failure to parse JSON object")
			}
			content := types.NewHotPathOpenProposal(title, description, propMetaData, inSafeMode)
			cosmosAddr := clientCtx.GetFromAddress()

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, initialDeposit, cosmosAddr)
		},
	}

	// AddGenericProposalCommandFlags(cmd)
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSetSafeModeProposalCmd implements the command to submit a SetSafeModeProposal
// nolint: dupl
func NewSetSafeModeProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "set-safe-mode [initial-deposit] [title] [description] [metadata] [in-safe-mode bool]",
		Args:  cobra.ExactArgs(5),
		Short: "Submit a SetSafeMode proposal",
		Long: `Submit a proposal to lock down the native DEX, or unlock it once it has been locked.
The proposal details must be supplied via a JSON file.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-legacy-proposal set-safe-mode <path/to/metadata.json> <in-safe-mode> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"lock_dex": "<true/false>",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return errorsmod.Wrap(err, "bad initial deposit amount")
			}
			title := args[1]
			description := args[2]
			metadataFile := args[3]
			inSafeModeStr := args[4]
			if len(initialDeposit) != 1 {
				return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
			}
			inSafeMode, err := strconv.ParseBool(inSafeModeStr)
			if err != nil {
				return errorsmod.Wrap(err, "invalid in-safe-mode")
			}
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			propMetaData, err := ParseSetSafeModeMetadata(clientCtx.Codec, metadataFile)
			if err != nil {
				return errorsmod.Wrap(err, "Failure to parse JSON object")
			}
			content := types.NewSetSafeModeProposal(title, description, propMetaData, inSafeMode)
			cosmosAddr := clientCtx.GetFromAddress()

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, initialDeposit, cosmosAddr)
		},
	}

	// AddGenericProposalCommandFlags(cmd)
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewTransferGovernanceProposalCmd implements the command to submit a TransferGovernanceProposal
// nolint: dupl
func NewTransferGovernanceProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "transfer-governance [initial-deposit] [title] [description] [metadata]",
		Args:  cobra.ExactArgs(4),
		Short: "Submit a TransferGovernance proposal",
		Long:  `Submit a proposal to set the CrocPolicy Ops and Emergency governance roles.`,
		Example: fmt.Sprintf(`$ %s tx gov submit-legacy-proposal transfer-governance <path/to/metadata.json> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"ops": "<hex address>",
	"emergency": "<hex address>",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return errorsmod.Wrap(err, "bad initial deposit amount")
			}
			title := args[1]
			description := args[2]
			metadataFile := args[3]
			if len(initialDeposit) != 1 {
				return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
			}
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cosmosAddr := clientCtx.GetFromAddress()
			propMetaData, err := ParseTransferGovernanceMetadata(clientCtx.Codec, metadataFile)
			if err != nil {
				return errorsmod.Wrap(err, "Failure to parse JSON object")
			}

			content := types.NewTransferGovernanceProposal(title, description, propMetaData)

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, initialDeposit, cosmosAddr)
		},
	}

	// AddGenericProposalCommandFlags(cmd)
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewOpsProposalCmd implements the command to submit a OpsProposal
// nolint: dupl
func NewOpsProposalCmd() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "ops [initial-deposit] [title] [description] [metadata]",
		Args:  cobra.ExactArgs(4),
		Short: "Submit an Ops proposal",
		Long:  `Submit a proposal to perform a non-sudo protocolCmd() call on the native DEX`,
		Example: fmt.Sprintf(`$ %s tx gov submit-legacy-proposal ops <path/to/metadata.json> --from=<key_or_address> --title=<title> --description=<description> --chain-id=<chain-id> --deposit=<deposit>

Where metadata.json contains (example):

{
	"callpath": "3",
	"cmd_args": "[<ABI Encoded arguments for the protocolCmd() call>]",
}`, version.AppName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return errorsmod.Wrap(err, "bad initial deposit amount")
			}
			title := args[1]
			description := args[2]
			metadataFile := args[3]
			if len(initialDeposit) != 1 {
				return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
			}
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cosmosAddr := clientCtx.GetFromAddress()
			propMetaData, err := ParseOpsMetadata(clientCtx.Codec, metadataFile)
			if err != nil {
				return errorsmod.Wrap(err, "Failure to parse JSON object")
			}

			content := types.NewOpsProposal(title, description, propMetaData)

			return GenericProposalCmdBroadcast(cmd, clientCtx, content, initialDeposit, cosmosAddr)
		},
	}

	// AddGenericProposalCommandFlags(cmd)
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// -----------------------------------------------------------------------------
// ColdPath Ops convenience command codes (mirror ProtocolCmd constants)
// -----------------------------------------------------------------------------
// The following convenience subcommands generate and submit Ops proposals targeting
// the ColdPath (proxy index 3) without requiring a metadata JSON file. Each command
// ABI-encodes the protocol command as (code, <args...>) matching the decoding logic
// in the Solidity ColdPath.protocolCmd() implementation:
//
//	disableTemplate       code=109  abi.encode(uint8,uint256(poolIdx))
//	setTemplate           code=110  abi.encode(uint8,uint256,uint16,uint16,uint8,uint8,uint8)
//	revisePool            code=111  abi.encode(uint8,address,address,uint256,uint16,uint16,uint8,uint8)
//	setNewPoolLiq         code=112  abi.encode(uint8,uint128)
//	pegPriceImprove       code=113  abi.encode(uint8,address,uint128,uint16)
//	setTakeRate           code=114  abi.encode(uint8,uint8)
//	resyncTakeRate        code=115  abi.encode(uint8,address,address,uint256)
//	setRelayerTakeRate    code=116  abi.encode(uint8,uint8)
//
// Example usage:
//
//	althea tx gov submit-proposal ops-set-template 1 25 10 5 0 0 \
//	  --from wallet --title "Set template" --description "Init template params" \
//	  --chain-id althea_1-1 --deposit 1000000aalthea
//
//	althea tx gov submit-proposal ops-revise-pool 0xBase... 0xQuote... 1 30 10 5 0 \
//	  --from wallet --title "Revise pool" --description "Adjust fee" --deposit 1000000aalthea
//
// These commands build a types.OpsMetadata with Callpath=3 and the encoded CmdArgs.
// If future protocol codes are added, replicate the pattern with a new command.
const (
	opsDisableTemplateCode uint8  = 109
	opsSetTemplateCode     uint8  = 110
	opsRevisePoolCode      uint8  = 111
	opsInitPoolLiqCode     uint8  = 112 // exposed for completeness (not used here)
	opsPegPriceImproveCode uint8  = 113
	opsSetTakeRateCode     uint8  = 114
	opsResyncTakeRateCode  uint8  = 115
	opsRelayerTakeRateCode uint8  = 116
	coldPathIndex          uint64 = 3 // proxyPath for ColdPath
)

// encodeOps builds ABI encoded cmd bytes for an Ops proposal.
func encodeOps(code uint8, typeNames []string, values []interface{}) ([]byte, error) {
	names := append([]string{"uint8"}, typeNames...)
	vals := append([]interface{}{code}, values...)
	for i, n := range names {
		if i == 0 {
			continue
		}
		switch n {
		case "address":
			if s, ok := vals[i].(string); ok {
				vals[i] = common.HexToAddress(s)
			}
		case "uint256", "uint128", "uint64", "uint32", "uint16", "uint8":
			switch v := vals[i].(type) {
			case uint64:
				vals[i] = v
			case uint32:
				vals[i] = v
			case uint16:
				vals[i] = v
			case uint8:
				vals[i] = v
			case *big.Int: // ok
			default:
				// best effort: attempt strconv if string
				if s, ok := v.(string); ok {
					if bi, ok2 := new(big.Int).SetString(s, 10); ok2 {
						vals[i] = bi
					}
				}
			}
		}
	}
	return contracts.EncodeTypes(names, vals)
}

// buildAndBroadcastOpsProposal retrieves proposal metadata directly from flags to avoid
// panic issues with reusing GenericProposalCmdSetup across multiple convenience commands.
func buildAndBroadcastOpsProposal(cmd *cobra.Command, title string, description string, initialDeposit sdk.Coins, encoded []byte) error {
	cliCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return err
	}
	cosmosAddr := cliCtx.GetFromAddress()

	proposal := &types.OpsProposal{
		Title:       title,
		Description: description,
		Metadata: types.OpsMetadata{
			Callpath: coldPathIndex,
			CmdArgs:  encoded,
		},
	}
	proposalAny, err := codectypes.NewAnyWithValue(proposal)
	if err != nil {
		return errorsmod.Wrap(err, "invalid proposal details")
	}

	msg := govv1beta1.MsgSubmitProposal{
		Proposer:       cosmosAddr.String(),
		InitialDeposit: initialDeposit,
		Content:        proposalAny,
	}
	if err := msg.ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "invalid proposal message")
	}
	return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), &msg)

}

// ops-disable-template: Disable a pool template (poolIdx)
// nolint: dupl
func NewOpsDisableTemplateCmd() *cobra.Command { //nolint: exhaustruct
	// nolint: exhaustruct
	c := &cobra.Command{
		Use:   "ops-disable-template [initial-deposit] [title] [description] [poolIdx]",
		Short: "Submit Ops proposal: disable pool template",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return errorsmod.Wrap(err, "bad initial deposit amount")
			}
			title := args[1]
			description := args[2]
			if len(initialDeposit) != 1 {
				return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
			}
			poolIdx, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return errorsmod.Wrap(err, "invalid poolIdx")
			}
			encoded, err := encodeOps(opsDisableTemplateCode, []string{"uint256"}, []interface{}{big.NewInt(0).SetUint64(poolIdx)})
			if err != nil {
				return errorsmod.Wrap(err, "encode failure")
			}
			return buildAndBroadcastOpsProposal(cmd, title, description, initialDeposit, encoded)
		},
	}
	flags.AddTxFlagsToCmd(c)
	return c
}

// ops-set-template: Set pool template params
// nolint: dupl
func NewOpsSetTemplateCmd() *cobra.Command { //nolint: exhaustruct
	// nolint: exhaustruct
	c := &cobra.Command{
		Use:   "ops-set-template [initial-deposit] [title] [description] [poolIdx] [feeRate] [tickSize] [jitThresh] [knockout] [oracleFlags]",
		Short: "Submit Ops proposal: set pool template parameters",
		Args:  cobra.ExactArgs(9),
		RunE: func(cmd *cobra.Command, args []string) error {
			initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return errorsmod.Wrap(err, "bad initial deposit amount")
			}

			title := args[1]
			description := args[2]

			if len(initialDeposit) != 1 {
				return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
			}

			poolIdx, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return errorsmod.Wrap(err, "invalid poolIdx")
			}
			feeRateU64, err := strconv.ParseUint(args[4], 10, 16)
			if err != nil {
				return errorsmod.Wrap(err, "invalid feeRate")
			}
			tickSizeU64, err := strconv.ParseUint(args[5], 10, 16)
			if err != nil {
				return errorsmod.Wrap(err, "invalid tickSize")
			}
			jitU64, err := strconv.ParseUint(args[6], 10, 8)
			if err != nil {
				return errorsmod.Wrap(err, "invalid jitThresh")
			}
			knockoutU64, err := strconv.ParseUint(args[7], 10, 8)
			if err != nil {
				return errorsmod.Wrap(err, "invalid knockout")
			}
			oracleU64, err := strconv.ParseUint(args[8], 10, 8)
			if err != nil {
				return errorsmod.Wrap(err, "invalid oracleFlags")
			}
			encoded, err := encodeOps(
				opsSetTemplateCode,
				[]string{"uint256", "uint16", "uint16", "uint8", "uint8", "uint8"},
				[]interface{}{big.NewInt(0).SetUint64(poolIdx), uint16(feeRateU64), uint16(tickSizeU64), uint8(jitU64), uint8(knockoutU64), uint8(oracleU64)},
			)
			if err != nil {
				return errorsmod.Wrap(err, "encode failure")
			}

			return buildAndBroadcastOpsProposal(cmd, title, description, initialDeposit, encoded)
		},
	}
	// AddGenericProposalCommandFlags(c)
	flags.AddTxFlagsToCmd(c)
	return c
}

// ops-revise-pool: Revise an existing pool specs
// nolint: dupl
func NewOpsRevisePoolCmd() *cobra.Command { //nolint: exhaustruct
	// nolint: exhaustruct
	c := &cobra.Command{
		Use:   "ops-revise-pool [initial-deposit] [title] [description] [baseAddr] [quoteAddr] [poolIdx] [feeRate] [tickSize] [jitThresh] [knockout] [oracleFlags]",
		Short: "Submit Ops proposal: revise existing pool parameters",
		Args:  cobra.ExactArgs(11),
		RunE: func(cmd *cobra.Command, args []string) error {
			initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return errorsmod.Wrap(err, "bad initial deposit amount")
			}
			title := args[1]
			description := args[2]
			if len(initialDeposit) != 1 {
				return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
			}
			base := args[3]
			quote := args[4]
			if !isHexAddress(base) {
				return errorsmod.Wrap(types.ErrInvalidEvmAddress, "invalid base address")
			}
			if !isHexAddress(quote) {
				return errorsmod.Wrap(types.ErrInvalidEvmAddress, "invalid quote address")
			}
			poolIdx, err := strconv.ParseUint(args[5], 10, 64)
			if err != nil {
				return errorsmod.Wrap(err, "invalid poolIdx")
			}
			feeRateU64, err := strconv.ParseUint(args[6], 10, 16)
			if err != nil {
				return errorsmod.Wrap(err, "invalid feeRate")
			}
			tickSizeU64, err := strconv.ParseUint(args[7], 10, 16)
			if err != nil {
				return errorsmod.Wrap(err, "invalid tickSize")
			}
			jitU64, err := strconv.ParseUint(args[8], 10, 8)
			if err != nil {
				return errorsmod.Wrap(err, "invalid jitThresh")
			}
			knockoutU64, err := strconv.ParseUint(args[9], 10, 8)
			if err != nil {
				return errorsmod.Wrap(err, "invalid knockout")
			}
			oracleU64, err := strconv.ParseUint(args[10], 10, 8)
			if err != nil {
				return errorsmod.Wrap(err, "invalid oracleFlags")
			}
			encoded, err := encodeOps(opsRevisePoolCode, []string{"address", "address", "uint256", "uint16", "uint16", "uint8", "uint8", "uint8"}, []interface{}{base, quote, big.NewInt(0).SetUint64(poolIdx), uint16(feeRateU64), uint16(tickSizeU64), uint8(jitU64), uint8(knockoutU64), uint8(oracleU64)})
			if err != nil {
				return errorsmod.Wrap(err, "encode failure")
			}
			return buildAndBroadcastOpsProposal(cmd, title, description, initialDeposit, encoded)
		},
	}
	flags.AddTxFlagsToCmd(c)
	return c
}

// ops-set-take-rate: set protocol take rate
// nolint: dupl
func NewOpsSetTakeRateCmd() *cobra.Command { //nolint: exhaustruct
	// nolint: exhaustruct
	c := &cobra.Command{Use: "ops-set-take-rate [initial-deposit] [title] [description] [takeRate]", Short: "Submit Ops proposal: set protocol take rate", Args: cobra.ExactArgs(4), RunE: func(cmd *cobra.Command, args []string) error {
		initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
		if err != nil {
			return errorsmod.Wrap(err, "bad initial deposit amount")
		}
		title := args[1]
		description := args[2]
		if len(initialDeposit) != 1 {
			return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
		}
		takeRateU64, err := strconv.ParseUint(args[3], 10, 8)
		if err != nil {
			return errorsmod.Wrap(err, "invalid takeRate")
		}
		encoded, err := encodeOps(opsSetTakeRateCode, []string{"uint8"}, []interface{}{uint8(takeRateU64)})
		if err != nil {
			return errorsmod.Wrap(err, "encode failure")
		}
		return buildAndBroadcastOpsProposal(cmd, title, description, initialDeposit, encoded)
	}}
	flags.AddTxFlagsToCmd(c)
	return c
}

// ops-set-relayer-take-rate: set relayer take rate
// nolint: dupl
func NewOpsSetRelayerTakeRateCmd() *cobra.Command { //nolint: exhaustruct
	// nolint: exhaustruct
	c := &cobra.Command{Use: "ops-set-relayer-take-rate [initial-deposit] [title] [description] [takeRate]", Short: "Submit Ops proposal: set relayer take rate", Args: cobra.ExactArgs(4), RunE: func(cmd *cobra.Command, args []string) error {
		initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
		if err != nil {
			return errorsmod.Wrap(err, "bad initial deposit amount")
		}
		title := args[1]
		description := args[2]
		if len(initialDeposit) != 1 {
			return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
		}
		takeRateU64, err := strconv.ParseUint(args[3], 10, 8)
		if err != nil {
			return errorsmod.Wrap(err, "invalid takeRate")
		}
		encoded, err := encodeOps(opsRelayerTakeRateCode, []string{"uint8"}, []interface{}{uint8(takeRateU64)})
		if err != nil {
			return errorsmod.Wrap(err, "encode failure")
		}
		return buildAndBroadcastOpsProposal(cmd, title, description, initialDeposit, encoded)
	}}
	flags.AddTxFlagsToCmd(c)
	return c
}

// ops-resync-take-rate: resync protocol take rate on an existing pool
// nolint: dupl
func NewOpsResyncTakeRateCmd() *cobra.Command { //nolint: exhaustruct
	// nolint: exhaustruct
	c := &cobra.Command{Use: "ops-resync-take-rate [initial-deposit] [title] [description] [baseAddr] [quoteAddr] [poolIdx]", Short: "Submit Ops proposal: resync take rate for pool", Args: cobra.ExactArgs(6), RunE: func(cmd *cobra.Command, args []string) error {
		initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
		if err != nil {
			return errorsmod.Wrap(err, "bad initial deposit amount")
		}
		title := args[1]
		description := args[2]
		if len(initialDeposit) != 1 {
			return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
		}
		base := args[3]
		quote := args[4]
		if !isHexAddress(base) {
			return errorsmod.Wrap(types.ErrInvalidEvmAddress, "invalid base address")
		}
		if !isHexAddress(quote) {
			return errorsmod.Wrap(types.ErrInvalidEvmAddress, "invalid quote address")
		}
		poolIdx, err := strconv.ParseUint(args[5], 10, 64)
		if err != nil {
			return errorsmod.Wrap(err, "invalid poolIdx")
		}
		encoded, err := encodeOps(opsResyncTakeRateCode, []string{"address", "address", "uint256"}, []interface{}{base, quote, big.NewInt(0).SetUint64(poolIdx)})
		if err != nil {
			return errorsmod.Wrap(err, "encode failure")
		}
		return buildAndBroadcastOpsProposal(cmd, title, description, initialDeposit, encoded)
	}}
	flags.AddTxFlagsToCmd(c)
	return c
}

// ops-set-new-pool-liq: set initial pool liquidity burn quantity
// nolint: dupl
func NewOpsSetNewPoolLiqCmd() *cobra.Command { //nolint: exhaustruct
	// nolint: exhaustruct
	c := &cobra.Command{Use: "ops-set-new-pool-liq [initial-deposit] [title] [description] [liq]", Short: "Submit Ops proposal: set initial pool liquidity burn quantity", Args: cobra.ExactArgs(4), RunE: func(cmd *cobra.Command, args []string) error {
		initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
		if err != nil {
			return errorsmod.Wrap(err, "bad initial deposit amount")
		}
		title := args[1]
		description := args[2]
		if len(initialDeposit) != 1 {
			return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
		}
		liqStr := args[3]
		liqBig, ok := new(big.Int).SetString(liqStr, 10)
		if !ok || liqBig.Sign() < 0 {
			return errorsmod.Wrap(fmt.Errorf("invalid uint128"), "invalid liq")
		}
		encoded, err := encodeOps(opsInitPoolLiqCode, []string{"uint128"}, []interface{}{liqBig})
		if err != nil {
			return errorsmod.Wrap(err, "encode failure")
		}
		return buildAndBroadcastOpsProposal(cmd, title, description, initialDeposit, encoded)
	}}
	flags.AddTxFlagsToCmd(c)
	return c
}

// ops-peg-price-improve: set off-grid price improvement thresholds
// nolint: dupl
func NewOpsPegPriceImproveCmd() *cobra.Command { //nolint: exhaustruct
	// nolint: exhaustruct
	c := &cobra.Command{Use: "ops-peg-price-improve [initial-deposit] [title] [description] [tokenAddr] [unitTickCollateral] [awayTickTol]", Short: "Submit Ops proposal: set off-grid price improvement thresholds", Args: cobra.ExactArgs(6), RunE: func(cmd *cobra.Command, args []string) error {
		initialDeposit, err := sdk.ParseCoinsNormalized(args[0])
		if err != nil {
			return errorsmod.Wrap(err, "bad initial deposit amount")
		}
		title := args[1]
		description := args[2]
		if len(initialDeposit) != 1 {
			return fmt.Errorf("unexpected coin amounts, expecting just 1 coin amount for initialDeposit")
		}
		token := args[3]
		if !isHexAddress(token) {
			return errorsmod.Wrap(types.ErrInvalidEvmAddress, "invalid token address")
		}
		unitTickStr := args[4]
		unitTickBig, ok := new(big.Int).SetString(unitTickStr, 10)
		if !ok || unitTickBig.Sign() < 0 {
			return errorsmod.Wrap(fmt.Errorf("invalid uint128"), "invalid unitTickCollateral")
		}
		awayTickTol, err := strconv.ParseUint(args[5], 10, 16)
		if err != nil {
			return errorsmod.Wrap(err, "invalid awayTickTol")
		}
		encoded, err := encodeOps(opsPegPriceImproveCode, []string{"address", "uint128", "uint16"}, []interface{}{token, unitTickBig, uint16(awayTickTol)})
		if err != nil {
			return errorsmod.Wrap(err, "encode failure")
		}
		return buildAndBroadcastOpsProposal(cmd, title, description, initialDeposit, encoded)
	}}
	flags.AddTxFlagsToCmd(c)
	return c
}

// isHexAddress performs a simple length/prefix heuristic; keeps CLI independent of go-ethereum common dependency.
func isHexAddress(s string) bool {
	if len(s) != 42 {
		return false
	}
	if s[0:2] != "0x" && s[0:2] != "0X" {
		return false
	}
	// Basic hex digit validation
	for i := 2; i < 42; i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
