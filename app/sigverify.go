package althea

import (
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	canto "github.com/Canto-Network/Canto/v6/app"
)

const (
	secp256k1VerifyCost uint64 = 21000
)

// SigVerificationGasConsumer is the Althea-Chain implementation of SignatureVerificationGasConsumer,
// based on Canto's version at https://github.com/Canto-Network/Canto in the app/sigverify.go file.
// It consumes gas for signature verification based upon the public key type.
// The cost is fetched from the given params and is matched by the concrete type.
// The types of keys supported are:
//
// - secp256k1 (Cosmos keys)
//
// HANDLED BY CANTO'S IMPLEMENTATION:
//
// - ethsecp256k1 (Ethereum keys)
// - ed25519 (Validators)
// - multisig (Cosmos SDK multisigs)
func SigVerificationGasConsumer(
	meter sdk.GasMeter, sig signing.SignatureV2, params authtypes.Params,
) error {
	pubkey := sig.PubKey
	switch pubkey.(type) {

	case *secp256k1.PubKey:
		// Cosmos keys (since it's the same algorithm as the Ethereum keys, charge the same amount)
		meter.ConsumeGas(secp256k1VerifyCost, "ante verify: secp256k1")
		return nil

	default:
		return canto.SigVerificationGasConsumer(meter, sig, params)
	}
}
