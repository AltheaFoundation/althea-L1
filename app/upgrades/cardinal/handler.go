package cardinal

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
	erc20types "github.com/AltheaFoundation/althea-L1/x/erc20/types"
	gasfreekeeper "github.com/AltheaFoundation/althea-L1/x/gasfree/keeper"
	nativedexkeeper "github.com/AltheaFoundation/althea-L1/x/nativedex/keeper"
	nativedextypes "github.com/AltheaFoundation/althea-L1/x/nativedex/types"
)

// TethysToCardinalPlanName is the on-chain upgrade plan name this handler is written for.
// It should match the name supplied in the governance upgrade proposal.
var TethysToCardinalPlanName = "cardinal"

// GetCardinalUpgradeHandler returns the upgrade handler for the Cardinal upgrade. It:
//  1. Ensures the NativeDex module account exists (needed for nativdex proposals to execute).
//  2. Fixes the iFi DEX stablecoin pair pool template (36000) to prevent future issues on new stablecoin pair pools.
//  3. Fixes the stablecoin pair pools (USDC-USDS, USDS-sUSDS, USDS-USDT) to match the updated template.
//  4. Updates gasfree module parameters to include additional message types.
func GetCardinalUpgradeHandler(
	mm *module.Manager, configurator *module.Configurator, crisisKeeper *crisiskeeper.Keeper, distrKeeper *distrkeeper.Keeper,
	accountKeeper authkeeper.AccountKeeper, nativedexKeeper nativedexkeeper.Keeper, gasfreeKeeper gasfreekeeper.Keeper,
) func(
	ctx sdk.Context, plan upgradetypes.Plan, vmap module.VersionMap,
) (module.VersionMap, error) {
	if mm == nil {
		panic("Nil argument to GetCardinalUpgradeHandler")
	}
	return func(ctx sdk.Context, plan upgradetypes.Plan, vmap module.VersionMap) (module.VersionMap, error) {
		ctx.Logger().Info("Module Consensus Version Map", "vmap", vmap)

		ctx.Logger().Info("Cardinal Upgrade: Running any configured module migrations")
		out, outErr := mm.RunMigrations(ctx, *configurator, vmap)

		initModuleAccount(ctx, accountKeeper)

		fixDexTemplate(ctx, nativedexKeeper)
		usdc := common.HexToAddress("0x80b5a32E4F032B2a058b4F29EC95EEfEEB87aDcd")
		sUsds := common.HexToAddress("0x5FD55A1B9FC24967C4dB09C513C3BA0DFa7FF687")
		usds := common.HexToAddress("0xd567B3d7B8FE3C79a1AD8dA978812cfC4Fa05e75")
		usdt := common.HexToAddress("0xecEEEfCEE421D8062EF8d6b4D814efe4dc898265")
		fixDexPool(ctx, nativedexKeeper, usdc, usds, 36000)
		fixDexPool(ctx, nativedexKeeper, sUsds, usds, 36000)
		fixDexPool(ctx, nativedexKeeper, usds, usdt, 36000)

		updateGasfreeParams(ctx, gasfreeKeeper, []common.Address{usdc, usds, sUsds, usdt})

		ctx.Logger().Info("Asserting invariants after upgrade")
		crisisKeeper.AssertInvariants(ctx)

		ctx.Logger().Info("Cardinal Upgrade Successful")
		return out, outErr
	}
}

// initModuleAccount creates the nativedex module account, which is needed for
// governance proposals to be able to execute successfully.
func initModuleAccount(ctx sdk.Context, accountKeeper authkeeper.AccountKeeper) {
	modAcc := accountKeeper.GetModuleAccount(ctx, nativedextypes.ModuleName)
	if modAcc.GetName() != nativedextypes.ModuleName {
		panic("Created account for nativedex module does not have the right module name!")
	}
}

// fixDexTemplate updates the 36000 pool template (used for stablecoin pairs) to ensure
// concentrated position creation works on new pools using this template.
func fixDexTemplate(ctx sdk.Context, nativedexKeeper nativedexkeeper.Keeper) {
	callpath := uint16(3) // ColdPath callpath
	setTemplateCode := uint8(110)
	templateIndexU64 := uint64(36000)
	templateIndex := big.NewInt(0).SetUint64(templateIndexU64)
	feeRate := uint16(5000)
	tickSize := uint16(1)
	jitThresh := uint8(6)
	knockout := uint8(182)
	oracleFlags := uint8(0)

	// Encode arguments for the setTemplate command.
	argumentTypeNames := []string{"uint8", "uint256", "uint16", "uint16", "uint8", "uint8", "uint8"}
	argumentValues := []interface{}{setTemplateCode, templateIndex, feeRate, tickSize, jitThresh, knockout, oracleFlags}
	cmdArgs, err := contracts.EncodeTypes(argumentTypeNames, argumentValues)
	if err != nil {
		panic("Could not encode cmd args for fixing dex template: " + err.Error())
	}

	// Dispatch via opsResolution -> ColdPath.protocolCmd -> setTemplate
	_, err = nativedexKeeper.EVMKeeper.CallEVM(
		ctx,
		contracts.CrocPolicyContract.ABI,
		nativedextypes.ModuleEVMAddress,
		nativedexKeeper.GetVerifiedCrocPolicyAddress(ctx),
		true, // static? privileged path
		"opsResolution",
		nativedexKeeper.GetNativeDexAddress(ctx),
		callpath,
		cmdArgs,
	)
	if err != nil {
		panic("Could not call opsResolution to fix dex template: " + err.Error())
	}
	ctx.Logger().Info("Successfully fixed dex template")
}

// fixDexPool revises parameters for an existing pool (base, quote, poolIdx) to the
// same values used in fixDexTemplate's template settings. This fixes the concentrated position
// creation issue for existing pools using the 36000 template.
func fixDexPool(ctx sdk.Context, nativedexKeeper nativedexkeeper.Keeper, base common.Address, quote common.Address, poolIdx uint64) {
	callpath := uint16(3) // ColdPath callpath
	revisePoolCode := uint8(111)

	// Desired parameter values (mirroring template)
	feeRate := uint16(5000)
	tickSize := uint16(1)
	jitThresh := uint8(6)
	knockout := uint8(182)

	// Encode arguments matching revisePool decoding layout.
	poolIdxBig := big.NewInt(0).SetUint64(poolIdx)
	argTypes := []string{"uint8", "address", "address", "uint256", "uint16", "uint16", "uint8", "uint8"}
	argValues := []interface{}{revisePoolCode, base, quote, poolIdxBig, feeRate, tickSize, jitThresh, knockout}
	cmdArgs, err := contracts.EncodeTypes(argTypes, argValues)
	if err != nil {
		panic("Could not encode cmd args for fixing dex pool: " + err.Error())
	}

	_, err = nativedexKeeper.EVMKeeper.CallEVM(
		ctx,
		contracts.CrocPolicyContract.ABI,
		nativedextypes.ModuleEVMAddress,
		nativedexKeeper.GetVerifiedCrocPolicyAddress(ctx),
		true,
		"opsResolution",
		nativedexKeeper.GetNativeDexAddress(ctx),
		callpath,
		cmdArgs,
	)
	if err != nil {
		panic("Could not call opsResolution to revise pool: " + err.Error())
	}
	ctx.Logger().Info("Successfully revised dex pool", "base", base.Hex(), "quote", quote.Hex(), "poolIdx", poolIdx)
}

// updateGasfreeParams adds new messages to the gasfree module params so that machine accounts are able to
// move stablecoins out of the EVM layer without needing to hold the native token for gas.
// The messages themselves will handle fee deduction instead.
func updateGasfreeParams(ctx sdk.Context, gasfreeKeeper gasfreekeeper.Keeper, gasfreeErc20InteropTokens []common.Address) {
	gasfreeFeeBasisPoints := uint64(100)
	gasfreeKeeper.SetGasfreeErc20InteropFeeBasisPoints(ctx, gasfreeFeeBasisPoints)
	gasfreeMessages := gasfreeKeeper.GetGasFreeMessageTypes(ctx)
	gasfreeMessages = append(
		gasfreeMessages,
		// nolint: exhaustruct
		sdk.MsgTypeURL(&erc20types.MsgSendCoinToEVM{}),
		// nolint: exhaustruct
		sdk.MsgTypeURL(&erc20types.MsgSendERC20ToCosmos{}),
		// nolint: exhaustruct
		sdk.MsgTypeURL(&erc20types.MsgSendERC20ToCosmosAndIBCTransfer{}),
	)
	gasfreeKeeper.SetGasFreeMessageTypes(ctx, gasfreeMessages)
	ctx.Logger().Info("Successfully updated gasfree message types", "gasfreeMessages", gasfreeMessages)

	stringAddresses := make([]string, len(gasfreeErc20InteropTokens))
	for i, addr := range gasfreeErc20InteropTokens {
		stringAddresses[i] = addr.Hex()
	}
	gasfreeKeeper.SetGasfreeErc20InteropTokens(ctx, stringAddresses)
	ctx.Logger().Info("Successfully updated gasfree ERC20 interop tokens", "gasfreeErc20InteropTokens", stringAddresses)
}
