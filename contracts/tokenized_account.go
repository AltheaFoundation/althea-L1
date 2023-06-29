package contracts

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

var (
	//go:embed compiled/TokenizedAccountNFT.json
	TokenizedAccountNFTJSON []byte // nolint: golint

	// TokenizedAccountNFTContract is the compiled erc20 contract
	TokenizedAccountNFTContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(TokenizedAccountNFTJSON, &TokenizedAccountNFTContract)
	if err != nil {
		panic(err)
	}

	if len(TokenizedAccountNFTContract.Bin) == 0 {
		panic("load contract failed")
	}
}
