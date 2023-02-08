package params

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
)

// MakeBaseEncodingConfig creates a basic EncodingConfig with no registrations
// the result is a blank slate with which all types must be registered
func MakeBaseEncodingConfig() EncodingConfig {
	amino := codec.NewLegacyAmino()
	interfaceRegistry := types.NewInterfaceRegistry()
	codec := codec.NewProtoCodec(interfaceRegistry)
	txCfg := tx.NewTxConfig(codec, tx.DefaultSignModes)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             codec,
		TxConfig:          txCfg,
		Amino:             amino,
	}
}
