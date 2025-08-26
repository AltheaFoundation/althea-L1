package althea

import (
	"github.com/cosmos/cosmos-sdk/simapp"
)

func NewDefaultGenesisState() simapp.GenesisState {
	encCfg := MakeEncodingConfig()
	return ModuleBasicManager.DefaultGenesis(encCfg.Codec)
}
