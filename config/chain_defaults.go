package config

const (
	// NativeToken is the native staking token used by the chain and MUST MATCH the mint module's MintDenom parameter
	// This value is used by the lockup module to set the default locked token denom
	NativeToken = "aalthea"

	// Chain ID configuration
	// 7357 = TEST, 417834 = ALTHEA mainnet
	ChainIdPrefix  = "althea_7357-"
	ChainIdVersion = "1"

	OneGwei = int64(1_000_000_000)
)

var (
	DefaultChainID     = func() string { return ChainIdPrefix + ChainIdVersion }
	DefaultMinGasPrice = func() int64 { return 1_000 * OneGwei }
)
