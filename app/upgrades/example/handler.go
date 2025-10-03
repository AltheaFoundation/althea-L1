package example

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/AltheaFoundation/althea-L1/contracts"
	nativedexkeeper "github.com/AltheaFoundation/althea-L1/x/nativedex/keeper"
	nativedextypes "github.com/AltheaFoundation/althea-L1/x/nativedex/types"
)

var PlanName = "example"

func GetExampleUpgradeHandler(
	mm *module.Manager, configurator *module.Configurator, crisisKeeper *crisiskeeper.Keeper, distrKeeper *distrkeeper.Keeper,
	accountKeeper authkeeper.AccountKeeper, nativedexKeeper nativedexkeeper.Keeper,
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

		// TODO: Make sure the store exists - if the params are set somewhere this should be true but not quite sure here

		initModuleAccount(ctx, accountKeeper)

		fixDexTemplate(ctx, nativedexKeeper)
		usdc := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
		usds := common.HexToAddress("0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
		sUsds := common.HexToAddress("0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
		usdt := common.HexToAddress("0xdAC17F958A7d8bD7cFfC4fE88D12eFbA6C172B0")
		fixDexPool(ctx, nativedexKeeper, usdc, usds, 36000)
		fixDexPool(ctx, nativedexKeeper, usds, sUsds, 36000)
		fixDexPool(ctx, nativedexKeeper, usds, usdt, 36000)

		ctx.Logger().Info("Asserting invariants after upgrade")
		crisisKeeper.AssertInvariants(ctx)

		ctx.Logger().Info("Example Upgrade Successful")
		return out, outErr
	}
}

func initModuleAccount(ctx sdk.Context, accountKeeper authkeeper.AccountKeeper) {
	modAcc := accountKeeper.GetModuleAccount(ctx, nativedextypes.ModuleName)
	if modAcc.GetName() != nativedextypes.ModuleName {
		panic("Created account for nativedex module does not have the right module name!")
	}
}

func fixDexTemplate(ctx sdk.Context, nativedexKeeper nativedexkeeper.Keeper) {
	callpath := 3 // ColdPath contract
	setTemplateCode := uint8(110)
	templateIndexU64 := uint64(36000)
	templateIndex := big.NewInt(0).SetUint64(templateIndexU64)
	feeRate := uint16(5000)
	tickSize := uint16(1)
	jitThresh := uint8(6)
	knockout := uint8(182)
	oracleFlags := uint8(0)
	// Encode the cmd argument for CrocPolicy's opsResolution function, which will eventually be used in ColdPath's setTemplate function.
	// The arguments are (uint8 code, uint256 poolIdx, uint16 feeRate, uint16 tickSize, uint8 jitThresh, uint8 knockout, uint8 oracleFlags)
	argumentTypeNames := []string{"uint8", "uint256", "uint16", "uint16", "uint8", "uint8", "uint8"}
	argumentValues := []interface{}{setTemplateCode, templateIndex, feeRate, tickSize, jitThresh, knockout, oracleFlags}
	cmdArgs, err := contracts.EncodeTypes(argumentTypeNames, argumentValues)
	if err != nil {
		panic("Could not encode cmd args for fixing dex template: " + err.Error())
	}
	_, err = nativedexKeeper.EVMKeeper.CallEVM(ctx, contracts.CrocPolicyContract.ABI, nativedextypes.ModuleEVMAddress, nativedexKeeper.GetVerifiedCrocPolicyAddress(ctx), true, "opsResolution", nativedexKeeper.GetNativeDexAddress(ctx), callpath, cmdArgs)
	if err != nil {
		panic("Could not call opsResolution to fix dex template: " + err.Error())
	}
	ctx.Logger().Info("Successfully fixed dex template")
}

// fixDexPool revises parameters for an existing pool (base, quote, poolIdx) to the
// same values used in fixDexTemplate's template settings. This constructs a protocol
// command targeting ColdPath.revisePool via ProtocolCmd.POOL_REVISE_CODE (111) and
// dispatches it through CrocPolicy.opsResolution.
//
// Solidity signature of revisePool decoding path in ColdPath:
//
//	(uint8 code, address base, address quote, uint256 poolIdx, uint16 feeRate, uint16 tickSize, uint8 jitThresh, uint8 knockout)
//
// Go side encoding must mirror that layout exactly.
func fixDexPool(ctx sdk.Context, nativedexKeeper nativedexkeeper.Keeper, base common.Address, quote common.Address, poolIdx uint64) {
	callpath := uint16(3)        // ColdPath contract
	revisePoolCode := uint8(111) // ProtocolCmd.POOL_REVISE_CODE

	// Reuse the same parameter values as template
	feeRate := uint16(5000)
	tickSize := uint16(1)
	jitThresh := uint8(6)
	knockout := uint8(182)

	// Encode arguments for (uint8, address, address, uint256, uint16, uint16, uint8, uint8)
	// Note: uint256 poolIdx must be a *big.Int
	poolIdxBig := big.NewInt(0).SetUint64(poolIdx)
	argTypes := []string{"uint8", "address", "address", "uint256", "uint16", "uint16", "uint8", "uint8"}
	argValues := []interface{}{revisePoolCode, base, quote, poolIdxBig, feeRate, tickSize, jitThresh, knockout}
	cmdArgs, err := contracts.EncodeTypes(argTypes, argValues)
	if err != nil {
		panic("Could not encode cmd args for fixing dex pool: " + err.Error())
	}

	_, err = nativedexKeeper.EVMKeeper.CallEVM(ctx, contracts.CrocPolicyContract.ABI, nativedextypes.ModuleEVMAddress, nativedexKeeper.GetVerifiedCrocPolicyAddress(ctx), true, "opsResolution", nativedexKeeper.GetNativeDexAddress(ctx), callpath, cmdArgs)
	if err != nil {
		panic("Could not call opsResolution to revise pool: " + err.Error())
	}
	ctx.Logger().Info("Successfully revised dex pool", "base", base.Hex(), "quote", quote.Hex(), "poolIdx", poolIdx)
}

// avoid unused warning if not yet wired into upgrade logic
var _ = fixDexPool
