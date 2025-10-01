package althea

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	// Tendermint
	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	dbm "github.com/tendermint/tm-db"

	// Cosmos SDK
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/store/streaming"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/posthandler"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrclient "github.com/cosmos/cosmos-sdk/x/distribution/client"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/group"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramsproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	// Cosmos IBC-Go
	ica "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts"
	icahost "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	transfer "github.com/cosmos/ibc-go/v6/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v6/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v6/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v6/modules/core/02-client"
	ibcclientclient "github.com/cosmos/ibc-go/v6/modules/core/02-client/client"
	ibcclienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v6/modules/core/05-port/types"
	ibchost "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v6/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v6/testing/types"

	// EVM
	ethante "github.com/evmos/ethermint/app/ante"
	"github.com/evmos/ethermint/ethereum/eip712"
	ethermintsrvflags "github.com/evmos/ethermint/server/flags"
	ethtypes "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"github.com/evmos/ethermint/x/evm/vm/geth"
	"github.com/evmos/ethermint/x/feemarket"
	feemarketkeeper "github.com/evmos/ethermint/x/feemarket/keeper"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"

	// unnamed import of statik for swagger UI support
	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"

	"github.com/AltheaFoundation/althea-L1/app/ante"
	altheaappparams "github.com/AltheaFoundation/althea-L1/app/params"
	"github.com/AltheaFoundation/althea-L1/app/upgrades"
	"github.com/AltheaFoundation/althea-L1/app/upgrades/tethys"
	altheacfg "github.com/AltheaFoundation/althea-L1/config"
	"github.com/AltheaFoundation/althea-L1/x/erc20"
	erc20client "github.com/AltheaFoundation/althea-L1/x/erc20/client"
	erc20keeper "github.com/AltheaFoundation/althea-L1/x/erc20/keeper"
	erc20types "github.com/AltheaFoundation/althea-L1/x/erc20/types"
	"github.com/AltheaFoundation/althea-L1/x/gasfree"
	gasfreekeeper "github.com/AltheaFoundation/althea-L1/x/gasfree/keeper"
	gasfreetypes "github.com/AltheaFoundation/althea-L1/x/gasfree/types"
	lockup "github.com/AltheaFoundation/althea-L1/x/lockup"
	lockupkeeper "github.com/AltheaFoundation/althea-L1/x/lockup/keeper"
	lockuptypes "github.com/AltheaFoundation/althea-L1/x/lockup/types"
	"github.com/AltheaFoundation/althea-L1/x/microtx"
	microtxkeeper "github.com/AltheaFoundation/althea-L1/x/microtx/keeper"
	microtxtypes "github.com/AltheaFoundation/althea-L1/x/microtx/types"
	"github.com/AltheaFoundation/althea-L1/x/nativedex"
	nativedexkeeper "github.com/AltheaFoundation/althea-L1/x/nativedex/keeper"
	nativedextypes "github.com/AltheaFoundation/althea-L1/x/nativedex/types"
	"github.com/AltheaFoundation/althea-L1/x/onboarding"
	onboardingkeeper "github.com/AltheaFoundation/althea-L1/x/onboarding/keeper"
	onboardingtypes "github.com/AltheaFoundation/althea-L1/x/onboarding/types"
)

func init() {
	// Set DefaultNodeHome before the chain starts
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	DefaultNodeHome = filepath.Join(userHomeDir, ".althea")

	// DefaultPowerReduction is used to translate full 6/8/18 decimal token value -> whole token representation for
	// computing validator power. By importing Canto's app package the DefaultPowerReduction is set for their
	// staking token, we manually adjust here to 18 decimals for aalthea here for peace of mind.
	sdk.DefaultPowerReduction = sdk.NewIntFromUint64(1_000_000_000_000_000_000)

	// TODO: Determine a sensible MinGasPrice for the EVM
	feemarkettypes.DefaultMinGasPrice = sdk.NewDec(altheacfg.DefaultMinGasPrice())
	feemarkettypes.DefaultMinGasMultiplier = sdk.NewDecWithPrec(1, 1)
}

const Name = "althea"

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		genutil.AppModuleBasic{},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
				distrclient.ProposalHandler,
				upgradeclient.LegacyProposalHandler,
				upgradeclient.LegacyCancelProposalHandler,
				ibcclientclient.UpdateClientProposalHandler,
				ibcclientclient.UpgradeProposalHandler,
				erc20client.RegisterCoinProposalHandler,
				erc20client.RegisterERC20ProposalHandler,
				erc20client.ToggleTokenConversionProposalHandler,
			},
		),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		ibc.AppModuleBasic{},
		// TODO: Add feegrant
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		vesting.AppModuleBasic{},
		lockup.AppModuleBasic{},
		microtx.AppModuleBasic{},
		onboarding.AppModuleBasic{},
		nativedex.AppModuleBasic{},
		evm.AppModuleBasic{},
		erc20.AppModuleBasic{},
		feemarket.AppModuleBasic{},
		ica.AppModuleBasic{},
		groupmodule.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		evmtypes.ModuleName:            {authtypes.Minter, authtypes.Burner}, // used for secure addition and subtraction of balance using module account
		evmtypes.FeeBurner:             {authtypes.Burner},
		minttypes.ModuleName:           {authtypes.Minter},
		erc20types.ModuleName:          {authtypes.Minter, authtypes.Burner},
		lockuptypes.ModuleName:         nil,
		microtxtypes.ModuleName:        nil,
		gasfreetypes.ModuleName:        nil,
		onboardingtypes.ModuleName:     nil,
		nativedextypes.ModuleName:      nil,
		feemarkettypes.ModuleName:      nil,
		icatypes.ModuleName:            nil,
	}

	// module accounts that are allowed to receive tokens
	allowedReceivingModAcc = map[string]bool{
		distrtypes.ModuleName: true,
		evmtypes.FeeBurner:    true,
	}

	// enable checks that run on the first BeginBlocker execution after an upgrade/genesis init/node restart
	firstBlock sync.Once
)

var (
	_ simapp.App              = (*AltheaApp)(nil)
	_ servertypes.Application = (*AltheaApp)(nil)
)

// AltheaApp extends an ABCI application
type AltheaApp struct { // nolint: golint
	*baseapp.BaseApp
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tKeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	// NOTE: If you add anything to this struct, add a nil check to ValidateMembers below!
	AccountKeeper     *authkeeper.AccountKeeper
	AuthzKeeper       *authzkeeper.Keeper
	BankKeeper        *bankkeeper.BaseKeeper
	CapabilityKeeper  *capabilitykeeper.Keeper
	StakingKeeper     *stakingkeeper.Keeper
	SlashingKeeper    *slashingkeeper.Keeper
	MintKeeper        *mintkeeper.Keeper
	DistrKeeper       *distrkeeper.Keeper
	GovKeeper         *govkeeper.Keeper
	CrisisKeeper      *crisiskeeper.Keeper
	UpgradeKeeper     *upgradekeeper.Keeper
	ParamsKeeper      *paramskeeper.Keeper
	FeegrantKeeper    *feegrantkeeper.Keeper
	IbcKeeper         *ibckeeper.Keeper
	EvidenceKeeper    *evidencekeeper.Keeper
	IbcTransferKeeper *ibctransferkeeper.Keeper
	EvmKeeper         *evmkeeper.Keeper
	Erc20Keeper       *erc20keeper.Keeper
	FeemarketKeeper   *feemarketkeeper.Keeper
	IcaHostKeeper     *icahostkeeper.Keeper
	GroupKeeper       *groupkeeper.Keeper

	LockupKeeper     *lockupkeeper.Keeper
	MicrotxKeeper    *microtxkeeper.Keeper
	GasfreeKeeper    *gasfreekeeper.Keeper
	OnboardingKeeper *onboardingkeeper.Keeper
	NativedexKeeper  *nativedexkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      *capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper *capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper  *capabilitykeeper.ScopedKeeper

	// the module manager
	MM *module.Manager

	// simulation manager
	sm *module.SimulationManager

	// Configurator
	Configurator *module.Configurator

	// amino and proto encoding
	EncodingConfig altheaappparams.EncodingConfig
}

// ValidateMembers checks for unexpected values, typically nil values needed for the chain to operate
// nolint: gocyclo
func (app AltheaApp) ValidateMembers() {
	if app.legacyAmino == nil {
		panic("Nil legacyAmino!")
	}

	// keepers
	if app.AccountKeeper == nil {
		panic("Nil AccountKeeper!")
	}
	if app.AuthzKeeper == nil {
		panic("Nil AuthzKeeper!")
	}
	if app.BankKeeper == nil {
		panic("Nil BankKeeper!")
	}
	if app.CapabilityKeeper == nil {
		panic("Nil CapabilityKeeper!")
	}
	if app.StakingKeeper == nil {
		panic("Nil StakingKeeper!")
	}
	if app.SlashingKeeper == nil {
		panic("Nil SlashingKeeper!")
	}
	if app.MintKeeper == nil {
		panic("Nil MintKeeper!")
	}
	if app.DistrKeeper == nil {
		panic("Nil DistrKeeper!")
	}
	if app.GovKeeper == nil {
		panic("Nil GovKeeper!")
	}
	if app.CrisisKeeper == nil {
		panic("Nil CrisisKeeper!")
	}
	if app.UpgradeKeeper == nil {
		panic("Nil UpgradeKeeper!")
	}
	if app.ParamsKeeper == nil {
		panic("Nil ParamsKeeper!")
	}
	if app.FeegrantKeeper == nil {
		panic("Nil FeegrantKeeper!")
	}
	if app.IbcKeeper == nil {
		panic("Nil IbcKeeper!")
	}
	if app.EvidenceKeeper == nil {
		panic("Nil EvidenceKeeper!")
	}
	if app.IbcTransferKeeper == nil {
		panic("Nil IbcTransferKeeper!")
	}
	if app.EvmKeeper == nil {
		panic("Nil EvmKeeper!")
	}
	if app.Erc20Keeper == nil {
		panic("Nil Erc20Keeper!")
	}
	if app.FeemarketKeeper == nil {
		panic("Nil FeemarketKeeper!")
	}
	if app.IcaHostKeeper == nil {
		panic("Nil IcaHostKeeper!")
	}
	if app.GroupKeeper == nil {
		panic("Nil GroupKeeper!")
	}

	if app.LockupKeeper == nil {
		panic("Nil LockupKeeper!")
	}
	if app.MicrotxKeeper == nil {
		panic("Nil MicrotxKeeper!")
	}
	if app.GasfreeKeeper == nil {
		panic("Nil GasfreeKeeper!")
	}
	if app.OnboardingKeeper == nil {
		panic("Nil OnboardingKeeper")
	}
	if app.NativedexKeeper == nil {
		panic("Nil NativedexKeeper")
	}

	// scoped keepers
	if app.ScopedIBCKeeper == nil {
		panic("Nil ScopedIBCKeeper!")
	}
	if app.ScopedTransferKeeper == nil {
		panic("Nil ScopedTransferKeeper!")
	}
	if app.ScopedICAHostKeeper == nil {
		panic("Nil ScopedICAHostKeeper!")
	}

	// managers
	if app.MM == nil {
		panic("Nil ModuleManager!")
	}
	if app.sm == nil {
		panic("Nil ModuleManager!")
	}
}

// NewAltheaApp returns a reference to an initialized Althea chain
// To avoid implicit duplication of critical values (thanks, Go) and buggy behavior, we declare nearly every used value
// locally and provide references to/duplicates of those local vars to every related constructor after initialization
func NewAltheaApp(
	logger log.Logger, db dbm.DB, traceStore io.Writer, loadLatest bool, skipUpgradeHeights map[int64]bool,
	homePath string, invCheckPeriod uint, encodingConfig altheaappparams.EncodingConfig, appOpts servertypes.AppOptions, baseAppOptions ...func(*baseapp.BaseApp),
) *AltheaApp {
	// --------------------------------------------------------------------------
	// -------------------------- Base Intitialization --------------------------
	// --------------------------------------------------------------------------

	// Core de/serialization types for Amino (legacy, needed for ledger) and Protobuf (new, recommended) formats
	appCodec := encodingConfig.Codec
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	eip712.SetEncodingConfig(simappparams.EncodingConfig(encodingConfig))

	// Baseapp initialization, provides correct implementation of ABCI layer, I/O services, state storage, and more
	bApp := *baseapp.NewBaseApp(Name, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	// Store keys for the typical persisted key-value store, one must be provided per module, all must be unique
	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey, authzkeeper.StoreKey, banktypes.StoreKey,
		stakingtypes.StoreKey, minttypes.StoreKey, distrtypes.StoreKey,
		slashingtypes.StoreKey, govtypes.StoreKey, paramstypes.StoreKey,
		ibchost.StoreKey, upgradetypes.StoreKey, evidencetypes.StoreKey,
		ibctransfertypes.StoreKey, capabilitytypes.StoreKey,
		erc20types.StoreKey, evmtypes.StoreKey, feemarkettypes.StoreKey,
		icahosttypes.StoreKey, group.StoreKey, feegrant.StoreKey,

		lockuptypes.StoreKey, microtxtypes.StoreKey, gasfreetypes.StoreKey,
		onboardingtypes.StoreKey, nativedextypes.StoreKey,
	)
	// Transient keys which only last for a block before being wiped
	// Params uses thsi to track whether some parameter changed this block or not
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey, evmtypes.TransientKey, feemarkettypes.TransientKey)
	// In-Memory keys which provide efficient lookup and caching, avoid store bloat, but need to be populated on startup,
	// including node restarts, new nodes, chain panics. Capability uses this to hide the actual capabilities and to store
	// bidirectional capability references for efficient lookup, but the KV store only contains a one-way mapping
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)

	// load state streaming if enabled
	if _, _, err := streaming.LoadStreamingServices(&bApp, appOpts, appCodec, keys); err != nil {
		fmt.Printf("failed to load state streaming: %s", err)
		os.Exit(1)
	}

	// nolint: exhaustruct
	app := &AltheaApp{
		BaseApp:           &bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
		keys:              keys,
		tKeys:             tkeys,
		memKeys:           memKeys,
		EncodingConfig:    encodingConfig,
	}

	// --------------------------------------------------------------------------
	// ------------------------- Keeper Intitialization -------------------------
	// --------------------------------------------------------------------------

	// Create all keepers and register inter-module relationships

	paramsKeeper := initParamsKeeper(appCodec, legacyAmino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])
	app.ParamsKeeper = &paramsKeeper

	bApp.SetParamStore(paramsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable()))

	// Capability keeper has the function to create "Scoped Keepers" which partition the capabilities a module is aware of
	// for security. Create all Scoped Keepers between here and the capabilityKeeper.Seal() call below.
	capabilityKeeper := *capabilitykeeper.NewKeeper(
		appCodec,
		keys[capabilitytypes.StoreKey],
		memKeys[capabilitytypes.MemStoreKey],
	)
	app.CapabilityKeeper = &capabilityKeeper

	scopedIBCKeeper := capabilityKeeper.ScopeToModule(ibchost.ModuleName)
	app.ScopedIBCKeeper = &scopedIBCKeeper

	scopedTransferKeeper := capabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	app.ScopedTransferKeeper = &scopedTransferKeeper

	scopedICAHostKeeper := capabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	app.ScopedICAHostKeeper = &scopedICAHostKeeper

	// No more scoped keepers from here on!
	capabilityKeeper.Seal()

	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		keys[authtypes.StoreKey],
		app.GetSubspace(authtypes.ModuleName),
		authtypes.ProtoBaseAccount,
		maccPerms,
		altheacfg.Bech32PrefixAccAddr,
	)
	app.AccountKeeper = &accountKeeper

	authzKeeper := authzkeeper.NewKeeper(
		keys[authzkeeper.StoreKey],
		appCodec,
		bApp.MsgServiceRouter(),
		accountKeeper,
	)
	app.AuthzKeeper = &authzKeeper

	bankKeeper := bankkeeper.NewBaseKeeper(
		appCodec,
		keys[banktypes.StoreKey],
		accountKeeper,
		app.GetSubspace(banktypes.ModuleName),
		app.BlockedAddrs(),
	)
	app.BankKeeper = &bankKeeper

	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec,
		keys[stakingtypes.StoreKey],
		accountKeeper,
		bankKeeper,
		app.GetSubspace(stakingtypes.ModuleName),
	)
	app.StakingKeeper = &stakingKeeper

	distrKeeper := distrkeeper.NewKeeper(
		appCodec,
		keys[distrtypes.StoreKey],
		app.GetSubspace(distrtypes.ModuleName),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		authtypes.FeeCollectorName,
	)
	app.DistrKeeper = &distrKeeper

	slashingKeeper := slashingkeeper.NewKeeper(
		appCodec,
		keys[slashingtypes.StoreKey],
		&stakingKeeper,
		app.GetSubspace(slashingtypes.ModuleName),
	)
	app.SlashingKeeper = &slashingKeeper

	// Connect the inter-module staking hooks together, these are the only modules allowed to interact with how staking
	// works, including inflationary staking rewards and punishing bad actors (excluding genutil which works at genesis to
	// seed the set of validators from the genesis txs set)
	stakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			distrKeeper.Hooks(),
			slashingKeeper.Hooks(),
		),
	)

	upgradeKeeper := upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		keys[upgradetypes.StoreKey],
		appCodec,
		homePath,
		&bApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.UpgradeKeeper = &upgradeKeeper

	ibcKeeper := *ibckeeper.NewKeeper(
		appCodec,
		keys[ibchost.StoreKey],
		app.GetSubspace(ibchost.ModuleName),
		stakingKeeper,
		upgradeKeeper,
		scopedIBCKeeper,
	)
	app.IbcKeeper = &ibcKeeper

	// EVM keepers
	tracer := cast.ToString(appOpts.Get(ethermintsrvflags.EVMTracer))

	// Feemarket implements EIP-1559 (https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1559.md) on Cosmos
	fmSs := app.GetSubspace(feemarkettypes.ModuleName)
	feemarketKeeper := feemarketkeeper.NewKeeper(
		appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		keys[feemarkettypes.StoreKey], tkeys[feemarkettypes.TransientKey],
		fmSs,
	)
	app.FeemarketKeeper = &feemarketKeeper

	// EVM calls the go-ethereum source code within ABCI to implement an EVM within Althea Chain
	evmSs := app.GetSubspace(evmtypes.ModuleName)
	evmKeeper := *evmkeeper.NewKeeper(
		appCodec,
		keys[evmtypes.StoreKey],
		tkeys[evmtypes.TransientKey],
		authtypes.NewModuleAddress(govtypes.ModuleName),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		feemarketKeeper,
		nil, geth.NewEVM,
		tracer,
		evmSs,
		ethtypes.ProtoAccountWithAddress,
	)

	// ERC20 provides translation between Cosmos-style tokens and Ethereum ERC20 contracts so that things like IBC work
	erc20Keeper := erc20keeper.NewKeeper(
		keys[erc20types.StoreKey],
		appCodec,
		app.GetSubspace(erc20types.ModuleName),
		accountKeeper,
		bankKeeper,
		&evmKeeper,
	)
	app.Erc20Keeper = &erc20Keeper

	// Connect the inter-module EVM hooks together, these are the only modules allowed to interact with how contracts are
	// executed, including ERC20's  Cosmos Coin <-> EVM ERC20 Token translation functions via magic contract address
	evmKeeper = *evmKeeper.SetHooks(evmkeeper.NewMultiEvmHooks(erc20Keeper.Hooks()))
	app.EvmKeeper = &evmKeeper

	// Note: onboarding keeper must have transfer keeper and channel keeper and the ics4 wrapper set
	onboardingKeeper := *onboardingkeeper.NewKeeper(
		app.GetSubspace(onboardingtypes.ModuleName), accountKeeper, bankKeeper, erc20Keeper, ibcKeeper.ChannelKeeper,
	)
	app.OnboardingKeeper = &onboardingKeeper
	app.OnboardingKeeper.Validate()

	ibcTransferKeeper := ibctransferkeeper.NewKeeper(
		appCodec, keys[ibctransfertypes.StoreKey], app.GetSubspace(ibctransfertypes.ModuleName),
		onboardingKeeper, ibcKeeper.ChannelKeeper, &ibcKeeper.PortKeeper,
		accountKeeper, bankKeeper, scopedTransferKeeper,
	)
	app.IbcTransferKeeper = &ibcTransferKeeper
	ibcTransferAppModule := transfer.NewAppModule(ibcTransferKeeper)

	icaHostKeeper := icahostkeeper.NewKeeper(
		appCodec, keys[icahosttypes.StoreKey], app.GetSubspace(icahosttypes.SubModuleName),
		ibcKeeper.ChannelKeeper, ibcKeeper.ChannelKeeper, &ibcKeeper.PortKeeper,
		accountKeeper, scopedICAHostKeeper, app.MsgServiceRouter(),
	)
	app.IcaHostKeeper = &icaHostKeeper

	// Construct the IBC Transfer "stack", a chain of IBC modules which enable arbitrary loosely-coupled Msg processing per module
	var transferStack porttypes.IBCModule = transfer.NewIBCModule(ibcTransferKeeper)
	transferStack = onboarding.NewIBCMiddleware(onboardingKeeper, transferStack)

	// Construct the ICA Host stack, which for the host module is very simple but the controller module could require
	// a more complex stack to handle the responses
	icaAppModule := ica.NewAppModule(nil, &icaHostKeeper)
	var icaHostStack porttypes.IBCModule = icahost.NewIBCModule(icaHostKeeper)

	// The stacks get wrapped into the router, which delegates applicable Msgs to the configured stack
	// i.e. only IBC Transfer Msgs will be routed to the transferStack, and only ICA Host Msgs will be routed to icaHostStack
	ibcRouter := porttypes.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)
	ibcRouter.AddRoute(icahosttypes.SubModuleName, icaHostStack)
	ibcKeeper.SetRouter(ibcRouter)

	mintKeeper := mintkeeper.NewKeeper(
		appCodec,
		keys[minttypes.StoreKey],
		app.GetSubspace(minttypes.ModuleName),
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
	)
	app.MintKeeper = &mintKeeper

	crisisKeeper := crisiskeeper.NewKeeper(
		app.GetSubspace(crisistypes.ModuleName),
		invCheckPeriod,
		bankKeeper,
		authtypes.FeeCollectorName,
	)
	app.CrisisKeeper = &crisisKeeper

	feegrantKeeper := feegrantkeeper.NewKeeper(appCodec, keys[feegrant.StoreKey], app.AccountKeeper)
	app.FeegrantKeeper = &feegrantKeeper

	// Nativedex allows management of the native DEX instance from the Cosmos side
	nativedexKeeper := nativedexkeeper.NewKeeper(
		keys[nativedextypes.StoreKey], appCodec, app.GetSubspace(nativedextypes.ModuleName),
		app.Erc20Keeper,
	)
	app.NativedexKeeper = &nativedexKeeper

	// Register custom governance proposal logic via router keys and handler functions
	govRouter := govv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramsproposal.RouterKey, params.NewParamChangeProposalHandler(paramsKeeper)).
		AddRoute(distrtypes.RouterKey, distr.NewCommunityPoolSpendProposalHandler(distrKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(upgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(ibcKeeper.ClientKeeper)).
		AddRoute(erc20types.RouterKey, erc20.NewErc20ProposalHandler(&erc20Keeper)).
		AddRoute(nativedextypes.RouterKey, nativedex.NewNativeDexProposalHandler(&nativedexKeeper))

	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		keys[govtypes.StoreKey],
		app.GetSubspace(govtypes.ModuleName),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		govRouter,
		app.MsgServiceRouter(),
		govConfig,
	)
	govKeeper = *govKeeper.SetHooks(govtypes.NewMultiGovHooks(
	// Register any governance hooks here
	))

	app.GovKeeper = &govKeeper

	evidenceKeeper := *evidencekeeper.NewKeeper(
		appCodec,
		keys[evidencetypes.StoreKey],
		&stakingKeeper,
		slashingKeeper,
	)
	app.EvidenceKeeper = &evidenceKeeper

	groupConfig := group.DefaultConfig()
	groupKeeper := groupkeeper.NewKeeper(keys[group.StoreKey], appCodec, app.MsgServiceRouter(), app.AccountKeeper, groupConfig)
	app.GroupKeeper = &groupKeeper

	// Althea custom modules

	// Lockup locks the chain at genesis to prevent native token transfers before the chain is sufficiently decentralized
	lockupKeeper := lockupkeeper.NewKeeper(
		appCodec, keys[lockuptypes.StoreKey], app.GetSubspace(lockuptypes.ModuleName),
	)
	app.LockupKeeper = &lockupKeeper

	// Gasfree allows for gasless transactions by bypassing the gas charging ante handlers for specific txs consisting of
	// governance controlled message types. These txs are charged fees out-of-band in a separate ante handler
	gasfreeKeeper := gasfreekeeper.NewKeeper(appCodec, keys[gasfreetypes.StoreKey], app.GetSubspace(gasfreetypes.ModuleName))
	app.GasfreeKeeper = &gasfreeKeeper

	// Microtx enables peer-to-peer automated microtransactions to form the payment layer for Althea-based networks
	microtxKeeper := microtxkeeper.NewKeeper(
		keys[microtxtypes.StoreKey], app.GetSubspace(microtxtypes.ModuleName), appCodec,
		&bankKeeper, &accountKeeper, &evmKeeper, &erc20Keeper, &gasfreeKeeper,
	)
	app.MicrotxKeeper = &microtxKeeper

	// --------------------------------------------------------------------------
	// ----------------------- AppModule Intitialization ------------------------
	// --------------------------------------------------------------------------

	var skipGenesisInvariants = cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	mm := *module.NewManager(
		genutil.NewAppModule(
			accountKeeper,
			stakingKeeper,
			bApp.DeliverTx,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(
			appCodec,
			accountKeeper,
			nil,
		),
		vesting.NewAppModule(accountKeeper, bankKeeper),
		authzmodule.NewAppModule(
			appCodec,
			authzKeeper,
			accountKeeper,
			bankKeeper,
			app.InterfaceRegistry(),
		),
		bank.NewAppModule(
			appCodec,
			bankKeeper,
			accountKeeper,
		),
		capability.NewAppModule(
			appCodec,
			capabilityKeeper,
		),
		crisis.NewAppModule(
			&crisisKeeper,
			skipGenesisInvariants,
		),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, *app.FeegrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(
			appCodec,
			govKeeper,
			accountKeeper,
			bankKeeper,
		),
		mint.NewAppModule(
			appCodec,
			mintKeeper,
			accountKeeper,
			nil,
		),
		slashing.NewAppModule(
			appCodec,
			slashingKeeper,
			accountKeeper,
			bankKeeper,
			stakingKeeper,
		),
		distr.NewAppModule(
			appCodec,
			distrKeeper,
			accountKeeper,
			bankKeeper,
			stakingKeeper,
		),
		staking.NewAppModule(
			appCodec,
			stakingKeeper,
			accountKeeper,
			bankKeeper,
		),
		upgrade.NewAppModule(upgradeKeeper),
		evidence.NewAppModule(evidenceKeeper),
		ibc.NewAppModule(&ibcKeeper),
		params.NewAppModule(paramsKeeper),
		ibcTransferAppModule,
		feemarket.NewAppModule(feemarketKeeper, fmSs),
		evm.NewAppModule(&evmKeeper, accountKeeper, evmSs),
		erc20.NewAppModule(erc20Keeper, accountKeeper),
		icaAppModule,
		gasfree.NewAppModule(gasfreeKeeper),
		lockup.NewAppModule(lockupKeeper, bankKeeper),
		microtx.NewAppModule(microtxKeeper, accountKeeper),
		onboarding.NewAppModule(onboardingKeeper),
		nativedex.NewAppModule(nativedexKeeper, accountKeeper),
		groupmodule.NewAppModule(appCodec, groupKeeper, accountKeeper, bankKeeper, interfaceRegistry),
	)
	app.MM = &mm

	// --------------------------------------------------------------------------
	// ---------------------------- ABCI Ordering -------------------------------
	// --------------------------------------------------------------------------

	// Determine the order in which modules' BeginBlock() functions are called each block

	// NOTE: capability module's BeginBlocker must come before any modules using capabilities (e.g. IBC)
	mm.SetOrderBeginBlockers(
		upgradetypes.ModuleName,
		capabilitytypes.ModuleName,
		feemarkettypes.ModuleName,
		evmtypes.ModuleName,
		minttypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		ibchost.ModuleName,
		ibctransfertypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		govtypes.ModuleName,
		crisistypes.ModuleName,
		genutiltypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		vestingtypes.ModuleName,
		gasfreetypes.ModuleName,
		lockuptypes.ModuleName,
		microtxtypes.ModuleName,
		erc20types.ModuleName,
		onboardingtypes.ModuleName,
		nativedextypes.ModuleName,
		icatypes.ModuleName,
		group.ModuleName,
	)

	// Determine the order in which modules' EndBlock() functions are called each block

	mm.SetOrderEndBlockers(
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		feemarkettypes.ModuleName,
		evmtypes.ModuleName,
		erc20types.ModuleName,
		ibchost.ModuleName,
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		minttypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		onboardingtypes.ModuleName,
		gasfreetypes.ModuleName,
		lockuptypes.ModuleName,
		microtxtypes.ModuleName,
		nativedextypes.ModuleName,
		group.ModuleName,
	)

	// Determine the order in which modules' InitGenesis() functions are called at chain genesis

	mm.SetOrderInitGenesis(
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		ibchost.ModuleName,
		evmtypes.ModuleName,
		feemarkettypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		gasfreetypes.ModuleName,
		lockuptypes.ModuleName,
		microtxtypes.ModuleName,
		erc20types.ModuleName,
		onboardingtypes.ModuleName,
		nativedextypes.ModuleName,
		group.ModuleName,
		crisistypes.ModuleName,
	)

	// --------------------------------------------------------------------------
	// ---------------------------- Miscellaneous -------------------------------
	// --------------------------------------------------------------------------

	mm.RegisterInvariants(&crisisKeeper)
	mm.RegisterRoutes(app.Router(), app.QueryRouter(), encodingConfig.Amino)
	configurator := module.NewConfigurator(appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.Configurator = &configurator
	mm.RegisterServices(*app.Configurator)

	// Simapp provides fuzz testing capabilities
	sm := *module.NewSimulationManager(
		auth.NewAppModule(appCodec, accountKeeper, authsims.RandomGenesisAccounts),
		bank.NewAppModule(appCodec, bankKeeper, accountKeeper),
		capability.NewAppModule(appCodec, capabilityKeeper),
		gov.NewAppModule(appCodec, govKeeper, accountKeeper, bankKeeper),
		mint.NewAppModule(appCodec, mintKeeper, accountKeeper, nil),
		staking.NewAppModule(appCodec, stakingKeeper, accountKeeper, bankKeeper),
		distr.NewAppModule(appCodec, distrKeeper, accountKeeper, bankKeeper, stakingKeeper),
		slashing.NewAppModule(appCodec, slashingKeeper, accountKeeper, bankKeeper, stakingKeeper),
		params.NewAppModule(paramsKeeper),
		evidence.NewAppModule(evidenceKeeper),
		ibc.NewAppModule(&ibcKeeper),
		ibcTransferAppModule,
		evm.NewAppModule(&evmKeeper, accountKeeper, evmSs),
		feemarket.NewAppModule(feemarketKeeper, fmSs),
	)
	app.sm = &sm

	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	// Create the chain of mempool Tx filter functions, aka the AnteHandler

	app.setAnteHandler(appOpts)
	app.setPostHandler()

	// Register the configured upgrades for the upgrade module
	app.registerUpgradeHandlers()
	app.registerStoreLoaders()

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}

	// Check for any obvious errors in initialization
	app.ValidateMembers()

	// Hand execution back to whichever cmd created this app
	return app
}

// Name returns the name of the App
func (app *AltheaApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker delegates the ABCI BeginBlock execution to the ModuleManager
func (app *AltheaApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	out := app.MM.BeginBlock(ctx, req)
	firstBlock.Do(func() { // Run the startup firstBeginBlocker assertions only once
		app.firstBeginBlocker(ctx, req)
	})

	return out
}

// firstBeginBlocker runs once at the end of the first BeginBlocker to check static assertions as a validator starts up
func (app *AltheaApp) firstBeginBlocker(ctx sdk.Context, _ abci.RequestBeginBlock) {
	app.assertBaseDenomMatchesConfig(ctx)
}

// EndBlocker delegates the ABCI EndBlock execution to the ModuleManager
func (app *AltheaApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.MM.EndBlock(ctx, req)
}

// InitChainer deserializes the given chain genesis state, registers in-place upgrade migrations, and delegates
// the ABCI InitGenesis execution to the ModuleManager
func (app *AltheaApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState simapp.GenesisState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.MM.GetVersionMap())

	return app.MM.InitGenesis(ctx, app.appCodec, genesisState)
}

func (app *AltheaApp) setAnteHandler(appOpts servertypes.AppOptions) {
	options := app.NewAnteHandlerOptions(appOpts)
	if err := options.Validate(); err != nil {
		panic(fmt.Errorf("invalid antehandler options: %v", err))
	}
	ah := ante.NewAnteHandler(options)

	// Create the lockup AnteHandler, to ensure sufficient decentralization before funds may be transferred
	lockupAnteHandler := lockup.NewWrappedLockupAnteHandler(ah, *app.LockupKeeper, app.AppCodec())
	app.SetAnteHandler(lockupAnteHandler)
}

func (app *AltheaApp) setPostHandler() {
	postHandler, err := posthandler.NewPostHandler(
		posthandler.HandlerOptions{},
	)
	if err != nil {
		panic(err)
	}

	app.SetPostHandler(postHandler)
}

// LoadHeight loads the blockchain a particular height
func (app *AltheaApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *AltheaApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// BlockedAddrs returns all the app's module account addresses that are not
// allowed to receive external tokens.
func (app *AltheaApp) BlockedAddrs() map[string]bool {
	blockedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = !allowedReceivingModAcc[acc]
	}

	return blockedAddrs
}

// LegacyAmino returns SimApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *AltheaApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns Althea Chain's codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *AltheaApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns Althea Chain's InterfaceRegistry which knows all of the Protobuf-defined interfaces
// and implementing types the chain is aware of
func (app *AltheaApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *AltheaApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *AltheaApp) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tKeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *AltheaApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetBaseApp returns the baseapp, used for testing
func (app *AltheaApp) GetBaseApp() *baseapp.BaseApp { return app.BaseApp }

// GetStakingKeeper returns the staking Keeper, used for testing
func (app *AltheaApp) GetStakingKeeper() ibctesting.StakingKeeper { return *app.StakingKeeper }

// GetIBCKeeper returns the IBC Keeper, used for testing
func (app *AltheaApp) GetIBCKeeper() *ibckeeper.Keeper { return app.IbcKeeper }

// GetScopedIBCKeeper returns the Scoped IBC Keeper, used for testing
func (app *AltheaApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper { return *app.ScopedIBCKeeper }

// GetTxConfig returns the Encoding config's tx config, used for testing
func (app *AltheaApp) GetTxConfig() client.TxConfig { return app.EncodingConfig.TxConfig }

// GetSubspace returns a param subspace for a given module name.
// Reading params is fine, but they should be updated by governace proposal
func (app *AltheaApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *AltheaApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *AltheaApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// GRPC endpoints under /cosmos.base.tendermint.v1beta1.Service
	// including GetNodeInfo, GetSyncing, GetLatestBlock, GetBlockByHeight, GetLatestValidatorSet, GetValidatorSetByHeight
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// GRPC endpoints under /cosmos.tx.v1beta1.Service
	// including Simulate, GetTx, BroadcastTx, GetTxsEvent, GetBlockWithTxs
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register all GRPC routes declared by modules in the ModuleBasics
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// TODO: build the custom swagger files and add here?
	if apiConfig.Swagger {
		RegisterSwaggerAPI(clientCtx, apiSvr.Router)
	}
}

// RegisterTxService registers all Protobuf-based Tx receiving gRPC services based on what is registered in the
// interface registry. These are stapled on to the baseapp's gRPC Query router
func (app *AltheaApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService registers the /cosmos.base.tendermint.v1beta1.Service query endpoints on the baseapp's
// gRPC query router
func (app *AltheaApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(clientCtx, app.BaseApp.GRPCQueryRouter(), app.interfaceRegistry, app.Query)
}

func (app *AltheaApp) RegisterNodeService(clientCtx client.Context) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter())
}

// RegisterSwaggerAPI registers swagger route with API Server
// TODO: build the custom swagger files and add here?
func RegisterSwaggerAPI(ctx client.Context, rtr *mux.Router) {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}
	return dupMaccPerms
}

// initParamsKeeper constructs params' keeper and all module param subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govv1.ParamKeyTable())
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibchost.ModuleName)
	paramsKeeper.Subspace(lockuptypes.ModuleName)
	paramsKeeper.Subspace(microtxtypes.ModuleName)
	paramsKeeper.Subspace(gasfreetypes.ModuleName)
	// Ethermint marks evmtypes.ParamKeyTable() as deprecated but does not have a documented replacement. For now, disable the linting error.
	paramsKeeper.Subspace(evmtypes.ModuleName).WithKeyTable(evmtypes.ParamKeyTable()) //nolint: staticcheck
	paramsKeeper.Subspace(erc20types.ModuleName)
	paramsKeeper.Subspace(feemarkettypes.ModuleName).WithKeyTable(feemarkettypes.ParamKeyTable())
	paramsKeeper.Subspace(icahosttypes.SubModuleName)
	paramsKeeper.Subspace(onboardingtypes.ModuleName)
	paramsKeeper.Subspace(nativedextypes.ModuleName)

	return paramsKeeper
}

// registerUpgradeHandlers registers in-place upgrades, which are faster and easier than genesis-based upgrades
func (app *AltheaApp) registerUpgradeHandlers() {
	upgrades.RegisterUpgradeHandlers(
		app.MM, app.Configurator, app.UpgradeKeeper, app.CrisisKeeper, app.DistrKeeper,
	)
}

// registerStoreLoaders handles special upgrades where module stores are added, removed, or renamed
func (app *AltheaApp) registerStoreLoaders() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}
	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	// STORE LOADER CONFIGURATION:
	// Added: []string{"newmodule"}, // We are adding these modules
	// Renamed: []storetypes.StoreRename{{"foo", "bar"}}, example foo to bar rename
	// Deleted: []string{"bazmodule"}, example deleted bazmodule

	// <name> Group module store loader setup
	if upgradeInfo.Name == tethys.PlanName {
		// Register the Group and Feegrant modules as new modules that need new stores allocated
		storeUpgrades := storetypes.StoreUpgrades{
			Added:   []string{group.StoreKey, feegrant.StoreKey},
			Renamed: nil,
			Deleted: nil,
		}

		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}

func (app *AltheaApp) NewAnteHandlerOptions(appOpts servertypes.AppOptions) ante.HandlerOptions {
	maxGasWanted := cast.ToUint64(appOpts.Get(ethermintsrvflags.EVMMaxTxGasWanted))
	return ante.HandlerOptions{
		AccountKeeper:          app.AccountKeeper,
		BankKeeper:             app.BankKeeper,
		SignModeHandler:        app.EncodingConfig.TxConfig.SignModeHandler(),
		FeegrantKeeper:         app.FeegrantKeeper,
		SigGasConsumer:         SigVerificationGasConsumer,
		IBCKeeper:              app.IbcKeeper,
		EvmKeeper:              app.EvmKeeper,
		FeeMarketKeeper:        *app.FeemarketKeeper,
		MaxTxGasWanted:         maxGasWanted,
		ExtensionOptionChecker: ante.CosmosExtensionOptionChecker,
		TxFeeChecker:           ethante.NewDynamicFeeChecker(app.EvmKeeper),
		DisabledAuthzMsgs: []string{
			//nolint: exhaustruct
			sdk.MsgTypeURL((&evmtypes.MsgEthereumTx{})),
			//nolint: exhaustruct
			sdk.MsgTypeURL((&vestingtypes.MsgCreateVestingAccount{})),
		},
		Cdc:           app.AppCodec(),
		GasfreeKeeper: app.GasfreeKeeper,
		MicrotxKeeper: app.MicrotxKeeper,
	}
}
