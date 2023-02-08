package keyring

import (
	sdkhd "github.com/cosmos/cosmos-sdk/crypto/hd"
	sdkkeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"

	etherminthd "github.com/evmos/ethermint/crypto/hd"
)

// Configuration
var (
	// SupportedAlgorithms defines the list of signing algorithms used on Althea Chain:
	//  - Secp256k1: Cosmos' implementation of the SECP256K1 signing algorithm, similar to but incompatibile with Ethereum's
	//  - EthSecp256k1: Ethereum's implementation of the SECP256K1 signing algorithm
	SupportedAlgorithms = sdkkeyring.SigningAlgoList{sdkhd.Secp256k1, etherminthd.EthSecp256k1}

	// SupportedAlgorithmsLedger defines the list of signing algorithms used on Althea Chain for the Ledger device:
	//  - eth_secp256k1 (Ethereum)
	// TODO: Add a custom keyring which delegates signing to the correct application, either "Cosmos" or "Ethereum"
	// 		depending on the key type
	SupportedAlgorithmsLedger = sdkkeyring.SigningAlgoList{sdkhd.Secp256k1, etherminthd.EthSecp256k1}
)

// Values used elsewhere, derived from the above config
func Option() sdkkeyring.Option {
	return func(options *sdkkeyring.Options) {
		options.SupportedAlgos = SupportedAlgorithms
		options.SupportedAlgosLedger = SupportedAlgorithmsLedger
	}
}
