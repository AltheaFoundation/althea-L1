package microtx

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/AltheaFoundation/althea-L1/x/microtx/keeper"
)

// EndBlocker is called at the end of every block
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
}
