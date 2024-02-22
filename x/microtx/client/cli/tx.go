package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/althea-net/althea-L1/x/microtx/types"
)

// GetTxCmd bundles all the subcmds together so they appear under `gravity tx`
func GetTxCmd(storeKey string) *cobra.Command {
	// nolint: exhaustruct
	microtxTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "microtx transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	microtxTxCmd.AddCommand([]*cobra.Command{
		CmdMicrotx(),
		CmdLiquify(),
	}...)

	return microtxTxCmd
}

// CmdMicrotx crafts and submits a MsgMicrotx to the chain
func CmdMicrotx() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "microtx [sender] [receiver] [amount]",
		Short: "microtx sends the provided amount from sender to receiver",
		Long:  "microtx will send amount (e.g. 1althea) from the bech32 address specified for `sender` to the bech32 address specified for `receiver`",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			sender, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return sdkerrors.Wrapf(err, "provided sender address is invalid: %v", args[0])
			}

			receiver, err := sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return sdkerrors.Wrapf(err, "provided receiver address is invalid: %v", args[1])
			}

			amount := args[2]
			coin, err := sdk.ParseCoinNormalized(amount)
			if err != nil {
				return sdkerrors.Wrapf(err, "invalid amount provided: %v", amount)
			}

			// Make the message
			msg := types.NewMsgMicrotx(sender.String(), receiver.String(), coin)
			if err := msg.ValidateBasic(); err != nil {
				return sdkerrors.Wrap(err, "invalid argument provided")
			}

			// Send it
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdLiquify crafts and submits a MsgLiquify to the chain
func CmdLiquify() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "liquify --from <account>",
		Short: "liquify will convert the account to a Liquid Infrastructure Account",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			if _, err := cmd.Flags().GetString(flags.FlagFrom); err != nil {
				return sdkerrors.Wrap(err, "--from value missing or incorrect")
			}
			from := cliCtx.GetFromAddress().String()

			// Make the message
			msg := types.NewMsgLiquify(from)
			if err := msg.ValidateBasic(); err != nil {
				return sdkerrors.Wrap(err, "invalid --from value provided")
			}

			// Send it
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
