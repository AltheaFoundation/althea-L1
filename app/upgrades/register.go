package upgrades

import (
	"github.com/cosmos/cosmos-sdk/types/module"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"

	"github.com/AltheaFoundation/althea-L1/app/upgrades/neutrino"
)

// RegisterUpgradeHandlers registers handlers for all upgrades
// Note: This method has crazy parameters because of circular import issues, I didn't want to make an AltheaApp struct
// along with an interface
func RegisterUpgradeHandlers(
	mm *module.Manager, configurator *module.Configurator, upgradeKeeper *upgradekeeper.Keeper,
	crisisKeeper *crisiskeeper.Keeper,
) {
	if mm == nil || configurator == nil || crisisKeeper == nil || upgradeKeeper == nil {
		panic("Nil argument to RegisterUpgradeHandlers()!")
	}
	// Neutrino upgrade
	upgradeKeeper.SetUpgradeHandler(
		neutrino.PlanName,
		neutrino.GetNeutrinoUpgradeHandler(mm, configurator, crisisKeeper),
	)
}