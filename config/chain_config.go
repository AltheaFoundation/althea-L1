package config

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	ethermint "github.com/evmos/ethermint/types"
)

const (
	// BaseDenom is the native staking token used by the chain and MUST MATCH the mint module's MintDenom parameter
	// This value is used by the lockup module to set the default locked token denom
	BaseDenom = "aalthea"
	// The whole token denom, aka 1 * 10^18 aalthea tokens
	DisplayDenom = "althea"

	// Chain ID configuration
	// 7357 = TEST, 417834 = ALTHEA mainnet
	ChainIdPrefix  = "althea_7357-"
	ChainIdVersion = "1"
)

var (
	DefaultChainID     = func() string { return ChainIdPrefix + ChainIdVersion }
	DefaultMinGasPrice = func() int64 { return 1 }
)

const (
	// Bech32PrefixAccAddr defines the Bech32 prefix of an account's address
	Bech32PrefixAccAddr = "althea"
	// Bech32PrefixAccPub defines the Bech32 prefix of an account's public key
	Bech32PrefixAccPub = "altheapub"
	// Bech32PrefixValAddr defines the Bech32 prefix of a validator's operator address
	Bech32PrefixValAddr = "altheavaloper"
	// Bech32PrefixValPub defines the Bech32 prefix of a validator's operator public key
	Bech32PrefixValPub = "altheavaloperpub"
	// Bech32PrefixConsAddr defines the Bech32 prefix of a consensus node address
	Bech32PrefixConsAddr = "altheavalcons"
	// Bech32PrefixConsPub defines the Bech32 prefix of a consensus node public key
	Bech32PrefixConsPub = "altheavalconspub"
)

func SetBech32Prefixes(config *sdk.Config) {
	config.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(Bech32PrefixValAddr, Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(Bech32PrefixConsAddr, Bech32PrefixConsPub)
}

// SetBip44CoinType sets the global coin type to be used in hierarchical deterministic wallets.
// Sets up the default HD path to be m/44'/60'/0'/0/0 (m/purpose/coin_type/account/change/address_index)
func SetBip44CoinType(config *sdk.Config) {
	config.SetPurpose(sdk.Purpose) // Shared
	config.SetCoinType(ethermint.Bip44CoinType)
	config.SetFullFundraiserPath(ethermint.BIP44HDPath) // nolint: staticcheck
}

// RegisterDenoms registers the base and display denominations to the SDK.
func RegisterDenoms() {
	if err := sdk.RegisterDenom(DisplayDenom, sdk.OneDec()); err != nil {
		panic(err)
	}

	if err := sdk.RegisterDenom(BaseDenom, sdk.NewDecWithPrec(1, ethermint.BaseDenomUnit)); err != nil {
		panic(err)
	}
}
