package althea

import (
	"github.com/AltheaFoundation/althea-L1/app/params"

	ethermintenccdc "github.com/evmos/ethermint/encoding/codec"
)

// MakeEncodingConfig creates a ready-to-use EncodingConfig, loaded with all the
// interfaces needed during chain operation
// Note that this function registers both the legacy Amino format (for Ledger use)
// and also the new and recommended Protobuf format
func MakeEncodingConfig() params.EncodingConfig {
	encodingConfig := params.MakeBaseEncodingConfig()
	// ethermint/encoding/codec's RegisterLegacyAminoCodec and RegisterInterfaces
	// do purely more registration than std does:

	// ethermint/crypto/codec: RegisterLegacyAminoCodec
	// Calls sdk.RegisterLegacyAminoCodec() for Msg and Tx
	// Calls ethermint/crypto/codec RegisterCrypto() for ethsecp256k1 Pub- and/Priv-Keys,
	// 		a deprecated LegacyInfo interface, BIP44 HD path type BIP44Params,
	// 		LegacyInfo types for public Local, Ledger, Offline, and Multisig key descriptions
	// Calls sdk/types RegisterCrypto() for [sr25519, ed25519, secp256k1] pubkeys, legacy amino multisig pubkeys,
	// 		PrivKey, [sr25519, ed25519, secp256k1] privkeys
	// Calls sdk/codec RegisterEvidences() for Evidence, and DuplicateVoteEvidence
	// overwrites the sdk/codec/legacy global Cdc var and sdk/client/keys global KeysCdc var to be the one created
	ethermintenccdc.RegisterLegacyAminoCodec(encodingConfig.Amino)

	// ethermint/crypto/codec: RegisterInterfaces
	// Calls std.RegisterInterfaces for Msg, Tx, PubKey, PrivKey, [ed25519, secp256k1] pub and privkeys, legacy amino multisig pubkeys, secp256r1 pubkey
	// Calls ethermint/crypto/codec.RegisterInterfaces() for ethsecp256k1 Pub- and/Priv-Keys
	// Calls ethermint.RegisterInterfaces for EthAccount as an implementation for AccountI + GenesisAccount,
	//		and Web3Tx + DynamicFeeTx as TxExtensionOption implementations
	ethermintenccdc.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	// Register all Amino and Protobuf Interfaces + Types for the modules Althea-Chain uses
	ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	return encodingConfig
}
