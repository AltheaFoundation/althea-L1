package main

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/spf13/cobra"
	tmtypes "github.com/tendermint/tendermint/types"

	evmtypes "github.com/evmos/ethermint/x/evm/types"

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
				return fmt.Errorf("error performing Althea-L1 additional validations on genesis file %s: %s", genesis, err.Error())
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

	bankGenDoc := genesis[banktypes.ModuleName]
	evmGenDoc := genesis[evmtypes.ModuleName]

	if err := ValidateCoinDeclarations(cdc, ctx, bankGenDoc); err != nil {
		return err
	}
	if err := ValidateEVMDenom(cdc, ctx, bankGenDoc, evmGenDoc); err != nil {
		return err
	}

	return nil
}

// ValidateMintGenesis will assert that the config's BaseDenom constant and the mint MintDenom match before the chain
// starts up. We have an assertion at chain runtime to ensure this, but a genesis check leads to easier chain launch.
func ValidateMintGenesis(cdc codec.JSONCodec, ctx client.Context, genesis json.RawMessage) error {
	var data minttypes.GenesisState
	if err := cdc.UnmarshalJSON(genesis, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", minttypes.ModuleName, err)
	}

	if data.Params.MintDenom != altheacfg.BaseDenom {
		return fmt.Errorf("the BaseDenom set in althea-chain/config (%v) is not the mint module's MintDenom (%v)", altheacfg.BaseDenom, data.Params.MintDenom)
	}

	return nil
}

// ValidateCoinDeclarations will assert that the denoms set up in the bank module genesis are sensible
func ValidateCoinDeclarations(cdc codec.JSONCodec, ctx client.Context, genesis json.RawMessage) error {
	var data banktypes.GenesisState
	if err := cdc.UnmarshalJSON(genesis, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", banktypes.ModuleName, err)
	}

	for _, metadata := range data.DenomMetadata {
		minDecimals := ^uint32(0) // This is how you get max of a uint32 in Go, don't ask me why
		minDenom := ""            // The name of the smallest (base) unit
		maxDecimals := 0
		maxDenom := "" // The name of the biggest unit (the name everyone calls it by)

		// Locate the names and sizes of the biggest and smallest units
		for _, unit := range metadata.DenomUnits {
			if unit.Exponent > uint32(maxDecimals) {
				maxDecimals = int(unit.Exponent)
				maxDenom = unit.Denom
			}
			if unit.Exponent < minDecimals {
				minDecimals = unit.Exponent
				minDenom = unit.Denom
			}
		}

		if metadata.Base != minDenom {
			return fmt.Errorf(
				"Invalid base denom: Expected base (%v) to equal the smallest unit (%v with exponent %v)",
				metadata.Base,
				minDenom,
				minDecimals,
			)
		}

		var expectedMinDenomPrefix string
		switch maxDecimals {
		case 6:
			expectedMinDenomPrefix = "u"
		case 9:
			expectedMinDenomPrefix = "n"
		case 18:
			expectedMinDenomPrefix = "a"
		}

		// Expecting to have a unit like "aalthea" (18 decimals) or "ugraviton" (6 decimals) or "ntoken" (9 decimals)
		if minDenom != expectedMinDenomPrefix+maxDenom {
			return fmt.Errorf(
				"Invalid DenomUnits: expecting the smallest denomination (%v with exponent %v) to begin with %v to adhere to token denom conventions",
				minDenom,
				minDecimals,
				expectedMinDenomPrefix,
			)
		}
		if minDenom != metadata.Base {
			return fmt.Errorf("Invalid DenomUnits: expecting the Base denom (%v) to be the smallest unit (%v with exponent %v)",
				metadata.Base,
				minDenom,
				minDecimals,
			)
		}
	}

	return nil
}

// ValidateEVMDenom will assert that the config's EVM denom actually exists, and has a supply
func ValidateEVMDenom(cdc codec.JSONCodec, ctx client.Context, bankGenesis json.RawMessage, evmGenesis json.RawMessage) error {
	var bankData banktypes.GenesisState
	if err := cdc.UnmarshalJSON(bankGenesis, &bankData); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", banktypes.ModuleName, err)
	}
	var evmData evmtypes.GenesisState
	if err := cdc.UnmarshalJSON(evmGenesis, &evmData); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", evmtypes.ModuleName, err)
	}

	bankMetadatas := bankData.DenomMetadata
	bankMetadataMap := MetadatasToMap(bankMetadatas)
	bankSupplies := bankData.Supply
	bankSupplyMap := CoinsToMap(bankSupplies)
	evmDenom := evmData.Params.EvmDenom
	foundEvmDenom := false

	for _, metadata := range bankMetadatas {
		supply, ok := bankSupplyMap[metadata.Base]
		if !ok {
			return fmt.Errorf("Could not find supply for token %v", metadata.Base)
		}
		if !supply.Amount.IsPositive() {
			return fmt.Errorf("Found invalid supply for token %v", metadata.Base)
		}
		if metadata.Base == evmDenom {
			foundEvmDenom = true
		}
	}

	for k := range bankSupplyMap {
		_, found := bankMetadataMap[k]
		if !found {
			return fmt.Errorf("Did not find metadata for token %v", k)
		}
	}

	if !foundEvmDenom {
		return fmt.Errorf("The EVM denom (%v) was not found in the bank module genesis, or it had nonpositive supply. Did you forget to set it?", evmDenom)
	}

	return nil
}

// Turns hard-to-search coins into a Denom -> Coin map
func CoinsToMap(coins sdk.Coins) map[string]sdk.Coin {
	ret := make(map[string]sdk.Coin)
	for _, c := range coins {
		ret[c.Denom] = c
	}

	return ret
}

// Turns hard-to-search coins into a Denom -> Coin map
func MetadatasToMap(metadata []banktypes.Metadata) map[string]banktypes.Metadata {
	ret := make(map[string]banktypes.Metadata)
	for _, c := range metadata {
		ret[c.Base] = c
	}

	return ret
}

// TODO: Validate all the genesis transactions so we can set up a convenient CI job for chain launch
