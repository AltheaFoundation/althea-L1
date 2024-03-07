package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidation(t *testing.T) {
	defaultGenesis := DefaultGenesisState()
	err := defaultGenesis.ValidateBasic()
	assert.Nil(t, err, "error produced from default genesis ValidateBasic %v", err)
	err = ValidateGasFreeMessageTypes(defaultGenesis.Params.GasFreeMessageTypes)
	assert.Nil(t, err, "error produced from default gasFreeMessageTypes validation")

	// It is not easy to construct an invalid GenesisState, as the only check is for
	// the GasFreeMessageTypes param, which only performs a type check
	// so here we pass nil as the best attempt

	// nolint: exhaustruct
	badGenesis := GenesisState{Params: nil}
	err = ValidateGasFreeMessageTypes(badGenesis)
	assert.NotNil(t, err, "badGenesis gasFreeMessageTypes did not produce an error in validation fn")
	err = badGenesis.ValidateBasic()
	assert.NotNil(t, err, "badGenesis did not produce an error in validation fn")
}
