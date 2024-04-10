package contracts

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

var (
	//go:embed compiled/CrocPolicy.json
	CrocPolicyJSON []byte // nolint: golint

	// CrocPolicyContract is the compiled contract
	CrocPolicyContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(CrocPolicyJSON, &CrocPolicyContract)
	if err != nil {
		panic(err)
	}

	if len(CrocPolicyContract.Bin) == 0 {
		panic("load contract failed")
	}
}
