package keeper

import (
	"github.com/althea-net/althea-chain/x/microtx/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// nolint: exhaustruct
var _ types.QueryServer = Keeper{
	storeKey: nil,
	// nolint: exhaustruct
	paramSpace: paramstypes.Subspace{},
	cdc:        nil,
	// nolint: exhaustruct
	bankKeeper: &bankkeeper.BaseKeeper{},
}
