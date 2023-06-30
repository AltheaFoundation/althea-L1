package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/althea-net/althea-chain/x/microtx/types"
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
		CmdTokenizeAccount(),
	}...)

	return microtxTxCmd
}

// CmdMicrotx crafts and submits a MsgMicrotx to the chain
func CmdMicrotx() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "microtx [sender] [receiver] [amount1] [[amount2] [amount3] ...]",
		Short: "microtx sends all provided amounts from sender to receiver",
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

			var amounts sdk.Coins
			for i := 2; i < len(args); i++ {
				amount := args[i]
				coin, err := sdk.ParseCoinNormalized(amount)
				if err != nil {
					return sdkerrors.Wrapf(err, "invalid amount provided: %v", amount)
				}
				amounts = amounts.Add(coin)
			}

			// Make the message
			msg := types.NewMsgMicrotx(sender.String(), receiver.String(), amounts)
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

// CmdTokenizeAccount crafts and submits a MsgTokenizeAccount to the chain
func CmdTokenizeAccount() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "tokenize-account --from <account>",
		Short: "tokenize-account will convert the account to Liquid Infrastructure",
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
			msg := types.NewMsgTokenizeAccount(from)
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
