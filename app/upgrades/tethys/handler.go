package tethys

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

var PlanName = "tethys"

func GetTethysUpgradeHandler(
	mm *module.Manager, configurator *module.Configurator, crisisKeeper *crisiskeeper.Keeper, distrKeeper *distrkeeper.Keeper,
) func(
	ctx sdk.Context, plan upgradetypes.Plan, vmap module.VersionMap,
) (module.VersionMap, error) {
	if mm == nil {
		panic("Nil argument to GetTethysUpgradeHandler")
	}
	return func(ctx sdk.Context, plan upgradetypes.Plan, vmap module.VersionMap) (module.VersionMap, error) {
		ctx.Logger().Info("Module Consensus Version Map", "vmap", vmap)

		ctx.Logger().Info("Tethys Upgrade: Running any configured module migrations")
		out, outErr := mm.RunMigrations(ctx, *configurator, vmap)

		updateDistributionParams(ctx, distrKeeper)

		ctx.Logger().Info("Asserting invariants after upgrade")
		crisisKeeper.AssertInvariants(ctx)

		ctx.Logger().Info("Tethys Upgrade Successful")
		return out, outErr
	}
}

func updateDistributionParams(ctx sdk.Context, distrKeeper *distrkeeper.Keeper) {
	ctx.Logger().Info("Tethys Upgrade: Updating Distribution module Params")
	distrParams := distrKeeper.GetParams(ctx)
	updatedBaseReward := sdk.NewDecWithPrec(50, 2)  // 50%
	updatedBonusReward := sdk.NewDecWithPrec(04, 2) // 04%
	distrParams.BaseProposerReward = updatedBaseReward
	distrParams.BonusProposerReward = updatedBonusReward
	ctx.Logger().Info("Updating Distribution module Params from ", distrKeeper.GetParams(ctx).String(), " to ", distrParams.String())
	distrKeeper.SetParams(ctx, distrParams)
}
