package contracts

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

var (
	//go:embed compiled/LiquidInfrastructureNFT.json
	LiquidInfrastructureNFTJSON []byte // nolint: golint

	// LiquidInfrastructureNFTContract is the compiled erc20 contract
	LiquidInfrastructureNFTContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(LiquidInfrastructureNFTJSON, &LiquidInfrastructureNFTContract)
	if err != nil {
		panic(err)
	}

	if len(LiquidInfrastructureNFTContract.Bin) == 0 {
		panic("load contract failed")
	}
}
