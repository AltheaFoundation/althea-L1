package cli

import (
	"strconv"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/AltheaFoundation/althea-L1/x/microtx/types"
)

const (
	FlagOwner   = "owner"
	FlagAccount = "account"
	FlagNFT     = "nft"
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
		CmdQueryMicrotxFee(),
		CmdQueryLiquidAccount(),
		CmdQueryLiquidAccounts(),
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

// CmdQueryMicrotxFee fetches the fee needed to Microtx a certain amount
func CmdQueryMicrotxFee() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "microtx-fee amount",
		Args:  cobra.ExactArgs(1),
		Short: "Query the fee needed to Microtx amount to another wallet",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			amount, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return errorsmod.Wrap(err, "invalid amount, expecting a nonnegative integer")
			}

			req := types.QueryMicrotxFeeRequest{
				Amount: amount,
			}

			res, err := queryClient.MicrotxFee(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryLiquidAccount fetches any Liquid Infrastructure Account matching the request
func CmdQueryLiquidAccount() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "liquid-account [--owner owner-bech32] [--account account-bech32] [--nft 0xNFTADDRESS]",
		Args:  cobra.ExactArgs(0),
		Short: "Query for any matching liquid accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			owner, err := cmd.Flags().GetString(FlagOwner)
			if err != nil {
				return err
			}

			account, err := cmd.Flags().GetString(FlagAccount)
			if err != nil {
				return err
			}

			nft, err := cmd.Flags().GetString(FlagNFT)
			if err != nil {
				return err
			}

			req := types.QueryLiquidAccountRequest{
				Owner:   owner,
				Account: account,
				Nft:     nft,
			}

			res, err := queryClient.LiquidAccount(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().String(FlagOwner, "", "the bech32 address (althea1abc...) OR the EIP-55 address (0xD34DB33F...) of the owner of the Liquid Infrastructure Account")
	cmd.Flags().String(FlagAccount, "", "the bech32 address (althea1abc...) of the Liquid Infrastructure Account")
	cmd.Flags().String(FlagNFT, "", "the EIP-55 (0xD3ADB33F...) address of the LiquidInfrastructureNFT contract")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryLiquidAccounts fetches all known Liquid Infrastructure Accounts
func CmdQueryLiquidAccounts() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "liquid-accounts",
		Args:  cobra.ExactArgs(0),
		Short: "Query for any matching liquid accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := types.QueryLiquidAccountsRequest{}

			res, err := queryClient.LiquidAccounts(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
