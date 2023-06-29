package types

import (
	"crypto/md5"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName is the name of the module
	ModuleName = "microtx"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey is the module name router key
	RouterKey = ModuleName

	// QuerierRoute to be used for querierer msgs
	QuerierRoute = ModuleName
)

var (
	// The Microtx Module's bech32 address
	ModuleAddress = authtypes.NewModuleAddress(ModuleName)
	// The Microtx Module's EVM address
	ModuleEVMAddress = common.BytesToAddress(ModuleAddress.Bytes())

	// TokenizedAccountsKey is the index for all TokenizedAccounts, whose keys contain
	// a bech32 x/auth account address and values are EVM TokenizedAccountNFT contract addresses
	TokenizedAccountsKey = HashString("TokenizedAccounts")
)

// GetTokenizedAccountKey returns the TokenizedAccount key for the given bech32 address,
// the key's format is [ TokenizedAccountsKey | bech32 address ]
func GetTokenizedAccountKey(address sdk.AccAddress) []byte {
	return AppendBytes(TokenizedAccountsKey, []byte(address.String()))
}

func GetAccountFromTokenizedAccountKey(key []byte) sdk.AccAddress {
	accountBz := key[len(TokenizedAccountsKey):]
	return sdk.AccAddress(accountBz)
}

// Hashing string using cryptographic MD5 function
// returns 128bit(16byte) value
func HashString(input string) []byte {
	md5 := md5.New()
	md5.Write([]byte(input))
	return md5.Sum(nil)
}

func AppendBytes(args ...[]byte) []byte {
	length := 0
	for _, v := range args {
		length += len(v)
	}

	res := make([]byte, length)

	length = 0
	for _, v := range args {
		copy(res[length:length+len(v)], v)
		length += len(v)
	}

	return res
}
