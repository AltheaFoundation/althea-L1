package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	v2 "github.com/AltheaFoundation/althea-L1/x/erc20/migrations/v2"
)

// nolint: exhaustruct
var _ module.MigrationHandler = Migrator{}.Migrate1to2

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{
		keeper: keeper,
	}
}

// Migrate1to2 migrates from consensus version 1 to 2.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.UpdateParams(ctx, &m.keeper.paramstore)
}
