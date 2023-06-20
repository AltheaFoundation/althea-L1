// Simply calls the NewRootCmd() command setup func and executes the returned
// command. See ./cmd/root.go for the important details.
package main

import (
	"os"

	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"

	althea "github.com/althea-net/althea-chain/app"
	altheacfg "github.com/althea-net/althea-chain/config"
)

func main() {
	setupConfig()
	altheacfg.RegisterDenoms() // Register aalthea and althea as token denoms

	rootCmd, _ := NewRootCmd()
	if err := Execute(rootCmd, althea.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)
		default:
			os.Exit(1)
		}
	}
}

// Applies modifications to the sdk Config:
// Bech32 prefixes
// HD path
func setupConfig() {
	config := sdk.GetConfig()
	altheacfg.SetBech32Prefixes(config)
	altheacfg.SetBip44CoinType(config)
	config.Seal()
}
