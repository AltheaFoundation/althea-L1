package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/althea-net/althea-L1/x/nativedex/types"
)

// nolint: exhaustruct
func TestGenesisState_Validate(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.ValidateBasic()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
