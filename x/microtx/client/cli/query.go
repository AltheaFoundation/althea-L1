package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/spf13/cobra"

	"github.com/althea-net/althea-chain/x/microtx/types"
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
		CmdQueryTokenizedAccount(),
		CmdQueryTokenizedAccounts(),
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
		Use:   "Microtx-fee amount",
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
				return sdkerrors.Wrap(err, "invalid amount, expecting a nonnegative integer")
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

// CmdQueryTokenizedAccount fetches any Tokenized Account matching the request
func CmdQueryTokenizedAccount() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "tokenized-account [--owner owner-bech32] [--account account-bech32] [--nft 0xNFTADDRESS]",
		Args:  cobra.ExactArgs(0),
		Short: "Query for any matching tokenized accounts",
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

			req := types.QueryTokenizedAccountRequest{
				Owner:            owner,
				TokenizedAccount: account,
				NftAddress:       nft,
			}

			res, err := queryClient.TokenizedAccount(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().String(FlagOwner, "", "the bech32 address (althea1abc...) of the owner of the Tokenized Account")
	cmd.Flags().String(FlagAccount, "", "the bech32 address (althea1abc...) of the Tokenized Account")
	cmd.Flags().String(FlagNFT, "", "the EIP-55 (0xD3ADB33F...) address of the TokenizedAccountNFT contract")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryTokenizedAccounts fetches all known Tokenized Accounts
func CmdQueryTokenizedAccounts() *cobra.Command {
	// nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "tokenized-accounts",
		Args:  cobra.ExactArgs(0),
		Short: "Query for any matching tokenized accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := types.QueryTokenizedAccountsRequest{}

			res, err := queryClient.TokenizedAccounts(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
