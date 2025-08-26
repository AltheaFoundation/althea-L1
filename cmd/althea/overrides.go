package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	module "github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/group"
	groupcli "github.com/cosmos/cosmos-sdk/x/group/client/cli"
)

// Overrides the TxCommands from moduleBasics before applying them to commands
func AddOverrideTxCommands(commands *cobra.Command, moduleBasics module.BasicManager) {
	tempCmd := &cobra.Command{Use: "temp", Short: "Do not use this command!"}
	moduleBasics.AddTxCommands(tempCmd)

	for _, command := range tempCmd.Commands() {
		// nolint: go-staticcheck
		overwrite := OverrideGroupModuleTxCommand(*command)
		commands.AddCommand(overwrite)
	}
}

// If the command is from the group module, overrides it with a custom implementation
func OverrideGroupModuleTxCommand(command cobra.Command) *cobra.Command {
	if command.Name() == group.ModuleName {
		return CustomGroupTxCommand()
	}

	return &command
}

// Overrides the submit-proposal command from the group module
func CustomGroupTxCommand() *cobra.Command {
	originalTxCommand := groupcli.TxCmd(group.ModuleName)
	custom := &cobra.Command{
		Use:                        group.ModuleName,
		Short:                      "Group transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	for _, command := range originalTxCommand.Commands() {
		if strings.Contains(command.Use, "submit-proposal") {
			// Override the submit-proposal command
			custom.AddCommand(CustomMsgSubmitProposalCmd())
			// Skip the original command we overrode
			continue
		}
		if strings.Contains(command.Use, "vote") && strings.Contains(command.Use, "voter") {
			// Override the submit-proposal command
			custom.AddCommand(CustomMsgSubmitVoteCmd())
			// Skip the original command we overrode
			continue
		}
		// If we made it here, just add the original command
		custom.AddCommand(command)
	}

	return custom
}

// CustomMsgSubmitProposalCmd is a direct copy of the original MsgSubmitProposalCmd
// however it does NOT make the assumption that the user will never provide the --from flag
func CustomMsgSubmitProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-proposal [proposal_json_file]",
		Short: "Submit a new proposal",
		Long: `Submit a new proposal.
Parameters:
			msg_tx_json_file: path to json file with messages that will be executed if the proposal is accepted.`,
		Example: fmt.Sprintf(`
%s tx group submit-proposal path/to/proposal.json

	Where proposal.json contains:

{
	"group_policy_address": "cosmos1...",
	// array of proto-JSON-encoded sdk.Msgs
	"messages": [
	{
		"@type": "/cosmos.bank.v1beta1.MsgSend",
		"from_address": "cosmos1...",
		"to_address": "cosmos1...",
		"amount":[{"denom": "stake","amount": "10"}]
	}
	],
	"metadata": "4pIMOgIGx1vZGU=", // base64-encoded metadata
	"proposers": ["cosmos1...", "cosmos1..."],
}`, version.AppName),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prop, err := getCLIProposal(args[0])
			if err != nil {
				return err
			}

			// ------------------- CHANGED CONTENTS START -------------------------
			// Since the --from flag is not required on this CLI command, we
			// *check to see if it was used, and if not we* just use the 1st proposer in the JSON file.
			from, err := cmd.Flags().GetString(flags.FlagFrom)
			if err != nil {
				panic("Unable to get from?")
			}
			if len(from) == 0 {
				cmd.Flags().Set(flags.FlagFrom, prop.Proposers[0])
			}
			// ------------------- CHANGED CONTENTS END   -------------------------

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msgs, err := parseMsgs(clientCtx.Codec, prop)
			if err != nil {
				return err
			}

			execStr, _ := cmd.Flags().GetString(groupcli.FlagExec)

			msg, err := group.NewMsgSubmitProposal(
				prop.GroupPolicyAddress,
				prop.Proposers,
				msgs,
				prop.Metadata,
				execFromString(execStr),
			)
			if err != nil {
				return err
			}

			if err = msg.ValidateBasic(); err != nil {
				return fmt.Errorf("message validation failed: %w", err)
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(groupcli.FlagExec, "", "Set to 1 to try to execute proposal immediately after creation (proposers signatures are considered as Yes votes)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CustomMsgVoteCmd is a direct copy of the original MsgSubmitVoteCmd
// however it does NOT make the assumption that the user will never provide the --from flag
func CustomMsgSubmitVoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote [proposal-id] [voter] [vote-option] [metadata]",
		Short: "Vote on a proposal",
		Long: `Vote on a proposal.

Parameters:
			proposal-id: unique ID of the proposal
			voter: voter account addresses.
			vote-option: choice of the voter(s)
				VOTE_OPTION_UNSPECIFIED: no-op
				VOTE_OPTION_NO: no
				VOTE_OPTION_YES: yes
				VOTE_OPTION_ABSTAIN: abstain
				VOTE_OPTION_NO_WITH_VETO: no-with-veto
			Metadata: metadata for the vote
`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			// ------------------- CHANGED CONTENTS START -------------------------
			// Since the --from flag is not required on this CLI command, we
			// *check to see if it was used, and if not we* just use the 1st proposer in the JSON file.
			from, err := cmd.Flags().GetString(flags.FlagFrom)
			if err != nil {
				panic("Unable to get from?")
			}
			if len(from) == 0 {
				cmd.Flags().Set(flags.FlagFrom, args[1])
			}
			// ------------------- CHANGED CONTENTS END   -------------------------

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposalID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			voteOption, err := group.VoteOptionFromString(args[2])
			if err != nil {
				return err
			}

			execStr, _ := cmd.Flags().GetString(groupcli.FlagExec)

			msg := &group.MsgVote{
				ProposalId: proposalID,
				Voter:      args[1],
				Option:     voteOption,
				Metadata:   args[3],
				Exec:       execFromString(execStr),
			}

			if err = msg.ValidateBasic(); err != nil {
				return fmt.Errorf("message validation failed: %w", err)
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(groupcli.FlagExec, "", "Set to 1 to try to execute proposal immediately after voting")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// The following were non-public functions in the original file, so I am forced to duplicate them here

// Proposal defines a Msg-based group proposal for CLI purposes.
type Proposal struct {
	GroupPolicyAddress string `json:"group_policy_address"`
	// Messages defines an array of sdk.Msgs proto-JSON-encoded as Anys.
	Messages  []json.RawMessage `json:"messages,omitempty"`
	Metadata  string            `json:"metadata"`
	Proposers []string          `json:"proposers,omitempty"`
}

func getCLIProposal(path string) (Proposal, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Proposal{}, err
	}

	return parseCLIProposal(contents)
}

func parseCLIProposal(contents []byte) (Proposal, error) {
	var p Proposal
	if err := json.Unmarshal(contents, &p); err != nil {
		return Proposal{}, err
	}

	return p, nil
}

func parseMsgs(cdc codec.Codec, p Proposal) ([]sdk.Msg, error) {
	msgs := make([]sdk.Msg, len(p.Messages))
	for i, anyJSON := range p.Messages {
		var msg sdk.Msg
		err := cdc.UnmarshalInterfaceJSON(anyJSON, &msg)
		if err != nil {
			return nil, err
		}

		msgs[i] = msg
	}

	return msgs, nil
}

func execFromString(execStr string) group.Exec {
	exec := group.Exec_EXEC_UNSPECIFIED
	switch execStr { //nolint:gocritic
	case groupcli.ExecTry:
		exec = group.Exec_EXEC_TRY
	}
	return exec
}
