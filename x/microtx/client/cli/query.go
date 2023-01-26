package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/spf13/cobra"

	"github.com/althea-net/althea-chain/x/microtx/types"
)

// GetQueryCmd bundles all the query subcmds together so they appear under the `query` or `q` subcommand
func GetQueryCmd() *cobra.Command {
	// nolint: exhaustruct
	microtxQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the microtx module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	microtxQueryCmd.AddCommand([]*cobra.Command{
		CmdQueryParams(),
		CmdQueryXferFee(),
	}...)

	return microtxQueryCmd
}

// CmdQueryParams fetches the current microtx params
func CmdQueryParams() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "params",
		Args:  cobra.NoArgs,
		Short: "Query microtx params",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&res.Params)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryXferFee fetches the fee needed to Xfer a certain amount
func CmdQueryXferFee() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "xfer-fee amount",
		Args:  cobra.ExactArgs(1),
		Short: "Query the fee needed to Xfer amount to another wallet",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			amount, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return sdkerrors.Wrap(err, "invalid amount, expecting a nonnegative integer")
			}

			req := types.QueryXferFeeRequest{
				Amount: amount,
			}

			res, err := queryClient.XferFee(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
