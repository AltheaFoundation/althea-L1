package types

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName defines the module name
	ModuleName = "nativedex"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName
)

var (
	// The Microtx Module's bech32 address
	ModuleAddress = authtypes.NewModuleAddress(ModuleName)
	// The Microtx Module's EVM address
	ModuleEVMAddress = common.BytesToAddress(ModuleAddress.Bytes())
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
