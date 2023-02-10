package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/spf13/cobra"
	tmtypes "github.com/tendermint/tendermint/types"

	altheacfg "github.com/althea-net/althea-chain/config"
)

// This file is mostly a copy of the genutil ValidateGenesisCmd and helpers over at https://github.com/cosmos/cosmos-sdk
// in the x/genutil/client/cli/validate_genesis.go file, with a specific function used to perform validations specific
// to the Althea Chain.

const chainUpgradeGuide = "https://docs.cosmos.network/master/migrations/chain-upgrade-guide-040.html"

// ValidateGenesisCmd takes a genesis file, and makes sure that it is valid.
func ValidateGenesisCmd(mbm module.BasicManager) *cobra.Command {
	return &cobra.Command{
		Use:   "validate-genesis [file]",
		Args:  cobra.RangeArgs(0, 1),
		Short: "validates the genesis file at the default location or at the location passed as an arg",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx := client.GetClientContextFromCmd(cmd)

			cdc := clientCtx.Codec

			// Load default if passed no args, otherwise load passed file
			var genesis string
			if len(args) == 0 {
				genesis = serverCtx.Config.GenesisFile()
			} else {
				genesis = args[0]
			}

			genDoc, err := validateGenDoc(genesis)
			if err != nil {
				return err
			}

			var genState map[string]json.RawMessage
			if err = json.Unmarshal(genDoc.AppState, &genState); err != nil {
				return fmt.Errorf("error unmarshalling genesis doc %s: %s", genesis, err.Error())
			}

			if err = mbm.ValidateGenesis(cdc, clientCtx.TxConfig, genState); err != nil {
				return fmt.Errorf("error validating genesis file %s: %s", genesis, err.Error())
			}

			if err = altheaValidateGenesis(cdc, clientCtx, genState); err != nil {
				return fmt.Errorf("error performing Althea-Chain-specific validations on genesis file %s: %s", genesis, err.Error())
			}

			fmt.Printf("File at %s is a valid genesis file\n", genesis)
			return nil
		},
	}
}

// validateGenDoc reads a genesis file and validates that it is a correct
// Tendermint GenesisDoc. This function does not do any cosmos-related
// validation.
func validateGenDoc(importGenesisFile string) (*tmtypes.GenesisDoc, error) {
	genDoc, err := tmtypes.GenesisDocFromFile(importGenesisFile)
	if err != nil {
		return nil, fmt.Errorf("%s. Make sure that"+
			" you have correctly migrated all Tendermint consensus params, please see the"+
			" chain migration guide at %s for more info",
			err.Error(), chainUpgradeGuide,
		)
	}

	return genDoc, nil
}

// AltheaValidateGenesis performs the Althea Chain validations, and is the whole reason the above code was copied
// from the cosmos-sdk source
func altheaValidateGenesis(cdc codec.JSONCodec, ctx client.Context, genesis map[string]json.RawMessage) error {
	mintGenDoc := genesis[minttypes.ModuleName]
	if err := ValidateMintGenesis(cdc, ctx, mintGenDoc); err != nil {
		return err
	}

	return nil
}

// ValidateMintGenesis will assert that the config's NativeToken constant and the mint MintDenom match before the chain
// starts up. We have an assertion at chain runtime to ensure this, but a genesis check leads to easier chain launch.
func ValidateMintGenesis(cdc codec.JSONCodec, ctx client.Context, genesis json.RawMessage) error {
	var data minttypes.GenesisState
	if err := cdc.UnmarshalJSON(genesis, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", minttypes.ModuleName, err)
	}

	if data.Params.MintDenom != altheacfg.NativeToken {
		return fmt.Errorf("the NativeToken set in althea-chain/config (%v) is not the mint module's MintDenom (%v)", altheacfg.NativeToken, data.Params.MintDenom)
	}

	return nil
}

// TODO: Validate all the genesis transactions so we can set up a convenient CI job for chain launch
