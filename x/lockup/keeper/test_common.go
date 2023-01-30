package keeper

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	ccodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/mint"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramsproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/althea-net/althea-chain/x/lockup/types"
)

// TestInput stores the various keepers required to test lockup
type TestInput struct {
	ParamsKeeper paramskeeper.Keeper
	LockupKeeper Keeper
	Context      sdk.Context
	Codec        codec.Codec
	LegacyAmino  *codec.LegacyAmino
}

var (
	// ModuleBasics is a mock module basic manager for testing
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.AppModuleBasic{},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distribution.AppModuleBasic{},
		gov.NewAppModuleBasic(paramsclient.ProposalHandler),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		vesting.AppModuleBasic{},
	)

	// TestingStakeParams is a set of staking params for testing
	TestingStakeParams = stakingtypes.Params{
		UnbondingTime:     100,
		MaxValidators:     10,
		MaxEntries:        10,
		HistoricalEntries: 10000,
		BondDenom:         "stake",
	}
)

// CreateTestEnv creates the keeper testing environment for lockup
func CreateTestEnv(t *testing.T) TestInput {
	t.Helper()

	// Initialize store keys
	lockupKey := sdk.NewKVStoreKey(types.StoreKey)
	keyAcc := sdk.NewKVStoreKey(authtypes.StoreKey)
	keyStaking := sdk.NewKVStoreKey(stakingtypes.StoreKey)
	keyBank := sdk.NewKVStoreKey(banktypes.StoreKey)
	keyDistro := sdk.NewKVStoreKey(distrtypes.StoreKey)
	keyParams := sdk.NewKVStoreKey(paramstypes.StoreKey)
	tkeyParams := sdk.NewTransientStoreKey(paramstypes.TStoreKey)
	keyGov := sdk.NewKVStoreKey(govtypes.StoreKey)

	// Initialize memory database and mount stores on it
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStaking, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyBank, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyDistro, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyGov, sdk.StoreTypeIAVL, db)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)

	// Create sdk.Context
	// nolint: exhaustruct
	ctx := sdk.NewContext(ms, tmproto.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, false, log.TestingLogger())

	legacyAmino := MakeTestCodec()
	protoCodec := MakeTestMarshaler()

	paramsKeeper := paramskeeper.NewKeeper(protoCodec, legacyAmino, keyParams, tkeyParams)
	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(types.ModuleName)

	// this is also used to initialize module accounts for all the map keys
	maccPerms := map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		types.ModuleName:               {authtypes.Minter, authtypes.Burner},
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		protoCodec,
		keyAcc, // target store
		getSubspace(paramsKeeper, authtypes.ModuleName),
		authtypes.ProtoBaseAccount, // prototype
		maccPerms,
	)

	blockedAddr := make(map[string]bool, len(maccPerms))
	for acc := range maccPerms {
		blockedAddr[authtypes.NewModuleAddress(acc).String()] = true
	}
	bankKeeper := bankkeeper.NewBaseKeeper(
		protoCodec,
		keyBank,
		accountKeeper,
		getSubspace(paramsKeeper, banktypes.ModuleName),
		blockedAddr,
	)

	// nolint: exhaustruct
	bankKeeper.SetParams(ctx, banktypes.Params{DefaultSendEnabled: true})

	stakingKeeper := stakingkeeper.NewKeeper(protoCodec, keyStaking, accountKeeper, bankKeeper, getSubspace(paramsKeeper, stakingtypes.ModuleName))
	stakingKeeper.SetParams(ctx, TestingStakeParams)

	distKeeper := distrkeeper.NewKeeper(protoCodec, keyDistro, getSubspace(paramsKeeper, distrtypes.ModuleName), accountKeeper, bankKeeper, stakingKeeper, authtypes.FeeCollectorName, nil)
	distKeeper.SetParams(ctx, distrtypes.DefaultParams())

	// set genesis items required for distribution
	distKeeper.SetFeePool(ctx, distrtypes.InitialFeePool())

	// set up initial accounts
	for name, perms := range maccPerms {
		mod := authtypes.NewEmptyModuleAccount(name, perms...)
		if name == distrtypes.ModuleName {
			// some big pot to pay out
			amt := sdk.NewCoins(sdk.NewInt64Coin("ualtg", 500000))
			err = bankKeeper.MintCoins(ctx, types.ModuleName, amt)
			require.NoError(t, err)
			err = bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, mod.Name, amt)

			// distribution module balance must be outstanding rewards + community pool in order to pass
			// invariants checks, therefore we must add any amount we add to the module balance to the fee pool
			feePool := distKeeper.GetFeePool(ctx)
			newCoins := feePool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(amt...)...)
			feePool.CommunityPool = newCoins
			distKeeper.SetFeePool(ctx, feePool)

			require.NoError(t, err)
		}
		accountKeeper.SetModuleAccount(ctx, mod)
	}

	stakeAddr := authtypes.NewModuleAddress(stakingtypes.BondedPoolName)
	moduleAcct := accountKeeper.GetAccount(ctx, stakeAddr)
	require.NotNil(t, moduleAcct)

	router := baseapp.NewRouter()
	// nolint: exhaustruct
	router.AddRoute(bank.AppModule{}.Route())
	// nolint: exhaustruct
	router.AddRoute(staking.AppModule{}.Route())
	// nolint: exhaustruct
	router.AddRoute(distribution.AppModule{}.Route())

	govRouter := govtypes.NewRouter().
		AddRoute(paramsproposal.RouterKey, params.NewParamChangeProposalHandler(paramsKeeper)).
		AddRoute(govtypes.RouterKey, govtypes.ProposalHandler)

	govKeeper := govkeeper.NewKeeper(
		protoCodec, keyGov, getSubspace(paramsKeeper, govtypes.ModuleName).WithKeyTable(govtypes.ParamKeyTable()), accountKeeper, bankKeeper, stakingKeeper, govRouter,
	)

	govKeeper.SetProposalID(ctx, govtypes.DefaultStartingProposalID)
	govKeeper.SetDepositParams(ctx, govtypes.DefaultDepositParams())
	govKeeper.SetVotingParams(ctx, govtypes.DefaultVotingParams())
	govKeeper.SetTallyParams(ctx, govtypes.DefaultTallyParams())
	k := NewKeeper(protoCodec, lockupKey, getSubspace(paramsKeeper, types.ModuleName))

	InitGenesis(ctx, k, *types.DefaultGenesisState())

	return TestInput{
		ParamsKeeper: paramsKeeper,
		LockupKeeper: k,
		Context:      ctx,
		Codec:        protoCodec,
		LegacyAmino:  legacyAmino,
	}
}

// TODO: Remove this from althea-chain and cosmos-gravity-bridge as well
// MakeTestCodec creates a legacy amino codec for testing
func MakeTestCodec() *codec.LegacyAmino {
	var cdc = codec.NewLegacyAmino()
	auth.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	bank.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	sdk.RegisterLegacyAminoCodec(cdc)
	ccodec.RegisterCrypto(cdc)
	params.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	return cdc
}

// MakeTestMarshaler creates a proto codec for use in testing
func MakeTestMarshaler() *codec.ProtoCodec {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	return codec.NewProtoCodec(interfaceRegistry)
}

// getSubspace returns a param subspace for a given module name.
func getSubspace(k paramskeeper.Keeper, moduleName string) paramstypes.Subspace {
	subspace, _ := k.GetSubspace(moduleName)
	return subspace
}
