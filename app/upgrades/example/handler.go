package example

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

var PlanName = "example"

func GetExampleUpgradeHandler(
	mm *module.Manager, configurator *module.Configurator, crisisKeeper *crisiskeeper.Keeper, distrKeeper *distrkeeper.Keeper,
) func(
	ctx sdk.Context, plan upgradetypes.Plan, vmap module.VersionMap,
) (module.VersionMap, error) {
	if mm == nil {
		panic("Nil argument to GetExampleUpgradeHandler")
	}
	return func(ctx sdk.Context, plan upgradetypes.Plan, vmap module.VersionMap) (module.VersionMap, error) {
		ctx.Logger().Info("Module Consensus Version Map", "vmap", vmap)

		ctx.Logger().Info("Example Upgrade: Running any configured module migrations")
		out, outErr := mm.RunMigrations(ctx, *configurator, vmap)

		ctx.Logger().Info("Asserting invariants after upgrade")
		crisisKeeper.AssertInvariants(ctx)

		ctx.Logger().Info("Example Upgrade Successful")
		return out, outErr
	}
}
