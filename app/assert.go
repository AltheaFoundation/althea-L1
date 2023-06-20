package althea

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/althea-net/althea-chain/config"
)

// Assert that the mint keeper and the Althea-Chain config match on their definitions of the BaseDenom
// Note: This should only be called a single time on chain startup, a great place to call this is in the BeginBlocker
// (since the chain is fully initialized then) guarded by a special sync.RunOnce variable
func (app *AltheaApp) assertBaseDenomMatchesConfig(ctx sdk.Context) {
	expectedBaseDenom := config.BaseDenom
	if app == nil || expectedBaseDenom == "" || app.mintKeeper == nil {
		panic("Unable to assert first BeginBlock configuration, some values are nil when they should be initialized!")
	}

	mintDenom := app.mintKeeper.GetParams(ctx).MintDenom
	if mintDenom == "" {
		panic("The mint keeper does not have a valid MintDenom set!")
	}

	if expectedBaseDenom != mintDenom {
		panic(fmt.Sprintf(
			"Mismatched mint module native token (%v) and expected native token (%v) - make sure that the genesis file matches "+
				"the value in app/config! The lockup module will be unable to lock the correct token if this is not corrected!",
			mintDenom, expectedBaseDenom,
		))
	}
}
