package althea

import (
	"github.com/cosmos/cosmos-sdk/simapp"
)

func NewDefaultGenesisState() simapp.GenesisState {
	encCfg := MakeEncodingConfig()
	return ModuleBasics.DefaultGenesis(encCfg.Codec)
}
