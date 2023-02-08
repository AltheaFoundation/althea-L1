// Simply calls the NewRootCmd() command setup func and executes the returned
// command. See ./cmd/root.go for the important details.
package main

import (
	"os"

	"github.com/althea-net/althea-chain/cmd/althea/cmd"
	"github.com/cosmos/cosmos-sdk/server"
)

func main() {
	rootCmd, _ := cmd.NewRootCmd()
	if err := cmd.Execute(rootCmd); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)
		default:
			os.Exit(1)
		}
	}
}
