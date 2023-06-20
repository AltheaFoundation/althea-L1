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

	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	dbm "github.com/tendermint/tm-db"

	// Cosmos SDK
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
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
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
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
	transfer "github.com/cosmos/ibc-go/v3/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v3/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v3/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v3/modules/core/02-client"
	ibcclientclient "github.com/cosmos/ibc-go/v3/modules/core/02-client/client"
	ibcclienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v3/modules/core/05-port/types"
	ibchost "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v3/modules/core/keeper"

	// EVM + ERC20

	cantoante "github.com/Canto-Network/Canto/v5/app/ante"
	"github.com/Canto-Network/Canto/v5/x/erc20"
	erc20client "github.com/Canto-Network/Canto/v5/x/erc20/client"
	erc20keeper "github.com/Canto-Network/Canto/v5/x/erc20/keeper"
	erc20types "github.com/Canto-Network/Canto/v5/x/erc20/types"
	"github.com/Canto-Network/Canto/v5/x/vesting"
	ethermintsrvflags "github.com/evmos/ethermint/server/flags"
	ethermint "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm"
	evmrest "github.com/evmos/ethermint/x/evm/client/rest"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"github.com/evmos/ethermint/x/feemarket"
	feemarketkeeper "github.com/evmos/ethermint/x/feemarket/keeper"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"

	// unnamed import of statik for swagger UI support
	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"

	altheaappparams "github.com/althea-net/althea-chain/app/params"
	altheacfg "github.com/althea-net/althea-chain/config"
	lockup "github.com/althea-net/althea-chain/x/lockup"
	lockupkeeper "github.com/althea-net/althea-chain/x/lockup/keeper"
	lockuptypes "github.com/althea-net/althea-chain/x/lockup/types"
	"github.com/althea-net/althea-chain/x/microtx"
	microtxkeeper "github.com/althea-net/althea-chain/x/microtx/keeper"
	microtxtypes "github.com/althea-net/althea-chain/x/microtx/types"
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
		gov.NewAppModuleBasic(
			paramsclient.ProposalHandler,
			distrclient.ProposalHandler,
			upgradeclient.ProposalHandler,
			upgradeclient.CancelProposalHandler,
			ibcclientclient.UpdateClientProposalHandler,
			ibcclientclient.UpgradeProposalHandler,
			erc20client.RegisterCoinProposalHandler,
			erc20client.RegisterERC20ProposalHandler,
			erc20client.ToggleTokenConversionProposalHandler,
		),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		ibc.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		vesting.AppModuleBasic{},
		lockup.AppModuleBasic{},
		microtx.AppModuleBasic{},
		evm.AppModuleBasic{},
		erc20.AppModuleBasic{},
		feemarket.AppModuleBasic{},
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
		minttypes.ModuleName:           {authtypes.Minter},
		erc20types.ModuleName:          {authtypes.Minter, authtypes.Burner},
		lockuptypes.ModuleName:         nil,
		microtxtypes.ModuleName:        nil,
		feemarkettypes.ModuleName:      nil,
	}

	// module accounts that are allowed to receive tokens
	allowedReceivingModAcc = map[string]bool{
		distrtypes.ModuleName: true,
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
	keys    map[string]*sdk.KVStoreKey
	tKeys   map[string]*sdk.TransientStoreKey
	memKeys map[string]*sdk.MemoryStoreKey

	// keepers
	// NOTE: If you add anything to this struct, add a nil check to ValidateMembers below!
	accountKeeper     *authkeeper.AccountKeeper
	authzKeeper       *authzkeeper.Keeper
	bankKeeper        *bankkeeper.BaseKeeper
	capabilityKeeper  *capabilitykeeper.Keeper
	stakingKeeper     *stakingkeeper.Keeper
	slashingKeeper    *slashingkeeper.Keeper
	mintKeeper        *mintkeeper.Keeper
	distrKeeper       *distrkeeper.Keeper
	govKeeper         *govkeeper.Keeper
	crisisKeeper      *crisiskeeper.Keeper
	upgradeKeeper     *upgradekeeper.Keeper
	paramsKeeper      *paramskeeper.Keeper
	ibcKeeper         *ibckeeper.Keeper
	evidenceKeeper    *evidencekeeper.Keeper
	ibcTransferKeeper *ibctransferkeeper.Keeper
	lockupKeeper      *lockupkeeper.Keeper
	microtxKeeper     *microtxkeeper.Keeper
	evmKeeper         *evmkeeper.Keeper
	erc20Keeper       *erc20keeper.Keeper
	feemarketKeeper   *feemarketkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      *capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper *capabilitykeeper.ScopedKeeper

	// the module manager
	mm *module.Manager

	// simulation manager
	sm *module.SimulationManager

	// configurator
	configurator *module.Configurator
}

// ValidateMembers checks for unexpected values, typically nil values needed for the chain to operate
func (app AltheaApp) ValidateMembers() {
	if app.legacyAmino == nil {
		panic("Nil legacyAmino!")
	}

	// keepers
	if app.accountKeeper == nil {
		panic("Nil accountKeeper!")
	}
	if app.authzKeeper == nil {
		panic("Nil authzKeeper!")
	}
	if app.bankKeeper == nil {
		panic("Nil bankKeeper!")
	}
	if app.capabilityKeeper == nil {
		panic("Nil capabilityKeeper!")
	}
	if app.stakingKeeper == nil {
		panic("Nil stakingKeeper!")
	}
	if app.slashingKeeper == nil {
		panic("Nil slashingKeeper!")
	}
	if app.mintKeeper == nil {
		panic("Nil mintKeeper!")
	}
	if app.distrKeeper == nil {
		panic("Nil distrKeeper!")
	}
	if app.govKeeper == nil {
		panic("Nil govKeeper!")
	}
	if app.crisisKeeper == nil {
		panic("Nil crisisKeeper!")
	}
	if app.upgradeKeeper == nil {
		panic("Nil upgradeKeeper!")
	}
	if app.paramsKeeper == nil {
		panic("Nil paramsKeeper!")
	}
	if app.ibcKeeper == nil {
		panic("Nil ibcKeeper!")
	}
	if app.evidenceKeeper == nil {
		panic("Nil evidenceKeeper!")
	}
	if app.ibcTransferKeeper == nil {
		panic("Nil ibcTransferKeeper!")
	}
	if app.lockupKeeper == nil {
		panic("Nil lockupKeeper!")
	}
	if app.microtxKeeper == nil {
		panic("Nil microtxKeeper!")
	}
	if app.evmKeeper == nil {
		panic("Nil evmKeeper!")
	}
	if app.erc20Keeper == nil {
		panic("Nil erc20Keeper!")
	}
	if app.feemarketKeeper == nil {
		panic("Nil feemarketKeeper!")
	}

	// scoped keepers
	if app.ScopedIBCKeeper == nil {
		panic("Nil ScopedIBCKeeper!")
	}
	if app.ScopedTransferKeeper == nil {
		panic("Nil ScopedTransferKeeper!")
	}

	// managers
	if app.mm == nil {
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
		ibctransfertypes.StoreKey, capabilitytypes.StoreKey, lockuptypes.StoreKey,
		erc20types.StoreKey, evmtypes.StoreKey, feemarkettypes.StoreKey,
	)
	// Transient keys which only last for a block before being wiped
	// Params uses thsi to track whether some parameter changed this block or not
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey, evmtypes.TransientKey, feemarkettypes.TransientKey)
	// In-Memory keys which provide efficient lookup and caching, avoid store bloat, but need to be populated on startup,
	// including node restarts, new nodes, chain panics. Capability uses this to hide the actual capabilities and to store
	// bidirectional capability references for efficient lookup, but the KV store only contains a one-way mapping
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)

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
	}

	// --------------------------------------------------------------------------
	// ------------------------- Keeper Intitialization -------------------------
	// --------------------------------------------------------------------------

	// Create all keepers and register inter-module relationships

	paramsKeeper := initParamsKeeper(appCodec, legacyAmino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])
	app.paramsKeeper = &paramsKeeper

	bApp.SetParamStore(paramsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramskeeper.ConsensusParamsKeyTable()))

	// Capability keeper has the function to create "Scoped Keepers" which partition the capabilities a module is aware of
	// for security. Create all Scoped Keepers between here and the capabilityKeeper.Seal() call below.
	capabilityKeeper := *capabilitykeeper.NewKeeper(
		appCodec,
		keys[capabilitytypes.StoreKey],
		memKeys[capabilitytypes.MemStoreKey],
	)
	app.capabilityKeeper = &capabilityKeeper

	scopedIBCKeeper := capabilityKeeper.ScopeToModule(ibchost.ModuleName)
	app.ScopedIBCKeeper = &scopedIBCKeeper

	scopedTransferKeeper := capabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	app.ScopedTransferKeeper = &scopedTransferKeeper

	// No more scoped keepers from here on!
	capabilityKeeper.Seal()

	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		keys[authtypes.StoreKey],
		app.GetSubspace(authtypes.ModuleName),
		ethermint.ProtoAccount,
		maccPerms,
	)
	app.accountKeeper = &accountKeeper

	authzKeeper := authzkeeper.NewKeeper(
		keys[authzkeeper.StoreKey],
		appCodec,
		bApp.MsgServiceRouter(),
	)
	app.authzKeeper = &authzKeeper

	bankKeeper := bankkeeper.NewBaseKeeper(
		appCodec,
		keys[banktypes.StoreKey],
		accountKeeper,
		app.GetSubspace(banktypes.ModuleName),
		app.BlockedAddrs(),
	)
	app.bankKeeper = &bankKeeper

	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec,
		keys[stakingtypes.StoreKey],
		accountKeeper,
		bankKeeper,
		app.GetSubspace(stakingtypes.ModuleName),
	)
	app.stakingKeeper = &stakingKeeper

	distrKeeper := distrkeeper.NewKeeper(
		appCodec,
		keys[distrtypes.StoreKey],
		app.GetSubspace(distrtypes.ModuleName),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		authtypes.FeeCollectorName,
		app.ModuleAccountAddrs(),
	)
	app.distrKeeper = &distrKeeper

	slashingKeeper := slashingkeeper.NewKeeper(
		appCodec,
		keys[slashingtypes.StoreKey],
		&stakingKeeper,
		app.GetSubspace(slashingtypes.ModuleName),
	)
	app.slashingKeeper = &slashingKeeper

	upgradeKeeper := upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		keys[upgradetypes.StoreKey],
		appCodec,
		homePath,
		&bApp,
	)
	app.upgradeKeeper = &upgradeKeeper

	ibcKeeper := *ibckeeper.NewKeeper(
		appCodec,
		keys[ibchost.StoreKey],
		app.GetSubspace(ibchost.ModuleName),
		stakingKeeper,
		upgradeKeeper,
		scopedIBCKeeper,
	)
	app.ibcKeeper = &ibcKeeper

	ibcTransferKeeper := ibctransferkeeper.NewKeeper(
		appCodec, keys[ibctransfertypes.StoreKey], app.GetSubspace(ibctransfertypes.ModuleName),
		ibcKeeper.ChannelKeeper, ibcKeeper.ChannelKeeper, &ibcKeeper.PortKeeper,
		accountKeeper, bankKeeper, scopedTransferKeeper,
	)
	app.ibcTransferKeeper = &ibcTransferKeeper

	// Connect the inter-module staking hooks together, these are the only modules allowed to interact with how staking
	// works, including inflationary staking rewards and punishing bad actors (excluding genutil which works at genesis to
	// seed the set of validators from the genesis txs set)
	stakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			distrKeeper.Hooks(),
			slashingKeeper.Hooks(),
		),
	)

	mintKeeper := mintkeeper.NewKeeper(
		appCodec,
		keys[minttypes.StoreKey],
		app.GetSubspace(minttypes.ModuleName),
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
	)
	app.mintKeeper = &mintKeeper

	crisisKeeper := crisiskeeper.NewKeeper(
		app.GetSubspace(crisistypes.ModuleName),
		invCheckPeriod,
		bankKeeper,
		authtypes.FeeCollectorName,
	)
	app.crisisKeeper = &crisisKeeper

	// EVM keepers
	tracer := cast.ToString(appOpts.Get(ethermintsrvflags.EVMTracer))

	// Feemarket implements EIP-1559 (https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1559.md) on Cosmos
	feemarketKeeper := feemarketkeeper.NewKeeper(
		appCodec, app.GetSubspace(feemarkettypes.ModuleName),
		keys[feemarkettypes.StoreKey], tkeys[feemarkettypes.TransientKey],
	)
	app.feemarketKeeper = &feemarketKeeper

	// EVM calls the go-ethereum source code within ABCI to implement an EVM within Althea Chain
	evmKeeper := evmkeeper.NewKeeper(
		appCodec,
		keys[evmtypes.StoreKey],
		tkeys[evmtypes.TransientKey],
		app.GetSubspace(evmtypes.ModuleName),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		feemarketKeeper,
		tracer,
	)

	// ERC20 provides translation between Cosmos-style tokens and Ethereum ERC20 contracts so that things like IBC work
	erc20Keeper := erc20keeper.NewKeeper(
		keys[erc20types.StoreKey],
		appCodec,
		app.GetSubspace(erc20types.ModuleName),
		accountKeeper,
		bankKeeper,
		evmKeeper,
	)
	app.erc20Keeper = &erc20Keeper

	// Connect the inter-module EVM hooks together, these are the only modules allowed to interact with how contracts are
	// executed, including ERC20's  Cosmos Coin <-> EVM ERC20 Token translation functions via magic contract address
	evmKeeper = evmKeeper.SetHooks(evmkeeper.NewMultiEvmHooks(erc20Keeper.Hooks()))
	app.evmKeeper = evmKeeper

	// Register custom governance proposal logic via router keys and handler functions
	govRouter := govtypes.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govtypes.ProposalHandler).
		AddRoute(paramsproposal.RouterKey, params.NewParamChangeProposalHandler(paramsKeeper)).
		AddRoute(distrtypes.RouterKey, distr.NewCommunityPoolSpendProposalHandler(distrKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(upgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(ibcKeeper.ClientKeeper)).
		AddRoute(erc20types.RouterKey, erc20.NewErc20ProposalHandler(&erc20Keeper))

	govKeeper := govkeeper.NewKeeper(
		appCodec,
		keys[govtypes.StoreKey],
		app.GetSubspace(govtypes.ModuleName),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		govRouter,
	)
	app.govKeeper = &govKeeper

	// Construct the IBC "stack", a chain of IBC modules which enable arbitrary loosely-coupled Msg processing per module
	// This is the simplest sort of stack, with just the core and a single IBC app for token transfers
	// e.g. advanced Interchain Accounts implementations need a point of execution for this chain to modify state once
	// updates on a foreign chain have been observed
	ibcTransferAppModule := transfer.NewAppModule(ibcTransferKeeper)
	ibcTransferIBCModule := transfer.NewIBCModule(ibcTransferKeeper)

	ibcRouter := porttypes.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, ibcTransferIBCModule)
	ibcKeeper.SetRouter(ibcRouter)

	evidenceKeeper := *evidencekeeper.NewKeeper(
		appCodec,
		keys[evidencetypes.StoreKey],
		&stakingKeeper,
		slashingKeeper,
	)
	app.evidenceKeeper = &evidenceKeeper

	// Althea custom modules

	// Lockup locks the chain at genesis to prevent native token transfers before the chain is sufficiently decentralized
	lockupKeeper := lockupkeeper.NewKeeper(
		appCodec, keys[lockuptypes.StoreKey], app.GetSubspace(lockuptypes.ModuleName),
	)
	app.lockupKeeper = &lockupKeeper

	// Microtx enables peer-to-peer automated microtransactions to form the payment layer for Althea-based networks
	microtxKeeper := microtxkeeper.NewKeeper(
		keys[microtxtypes.StoreKey], app.GetSubspace(microtxtypes.ModuleName), appCodec, &bankKeeper, &accountKeeper,
	)
	app.microtxKeeper = &microtxKeeper

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
		staking.NewAppModule(appCodec,
			stakingKeeper,
			accountKeeper,
			bankKeeper,
		),
		upgrade.NewAppModule(upgradeKeeper),
		evidence.NewAppModule(evidenceKeeper),
		ibc.NewAppModule(&ibcKeeper),
		params.NewAppModule(paramsKeeper),
		ibcTransferAppModule,
		lockup.NewAppModule(lockupKeeper, bankKeeper),
		microtx.NewAppModule(microtxKeeper, bankKeeper),
		evm.NewAppModule(evmKeeper, accountKeeper),
		erc20.NewAppModule(erc20Keeper, accountKeeper),
		feemarket.NewAppModule(feemarketKeeper),
	)
	app.mm = &mm

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
		banktypes.ModuleName,
		crisistypes.ModuleName,
		authtypes.ModuleName,
		ibctransfertypes.ModuleName,
		genutiltypes.ModuleName,
		authz.ModuleName,
		govtypes.ModuleName,
		paramstypes.ModuleName,
		lockuptypes.ModuleName,
		microtxtypes.ModuleName,
		erc20types.ModuleName,
	)

	// Determine the order in which modules' EndBlock() functions are called each block

	mm.SetOrderEndBlockers(
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		evmtypes.ModuleName,
		feemarkettypes.ModuleName,
		upgradetypes.ModuleName,
		capabilitytypes.ModuleName,
		minttypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		ibchost.ModuleName,
		banktypes.ModuleName,
		authtypes.ModuleName,
		ibctransfertypes.ModuleName,
		genutiltypes.ModuleName,
		authz.ModuleName,
		paramstypes.ModuleName,
		lockuptypes.ModuleName,
		microtxtypes.ModuleName,
		erc20types.ModuleName,
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
		upgradetypes.ModuleName,
		ibchost.ModuleName,
		evmtypes.ModuleName,
		feemarkettypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		ibctransfertypes.ModuleName,
		authz.ModuleName,
		paramstypes.ModuleName,
		lockuptypes.ModuleName,
		microtxtypes.ModuleName,
		erc20types.ModuleName,
		crisistypes.ModuleName,
	)

	// --------------------------------------------------------------------------
	// ---------------------------- Miscellaneous -------------------------------
	// --------------------------------------------------------------------------

	mm.RegisterInvariants(&crisisKeeper)
	mm.RegisterRoutes(app.Router(), app.QueryRouter(), encodingConfig.Amino)
	configurator := module.NewConfigurator(appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.configurator = &configurator
	mm.RegisterServices(*app.configurator)

	// Simapp provides fuzz testing capabilities
	sm := *module.NewSimulationManager(
		auth.NewAppModule(appCodec, accountKeeper, authsims.RandomGenesisAccounts),
		bank.NewAppModule(appCodec, bankKeeper, accountKeeper),
		capability.NewAppModule(appCodec, capabilityKeeper),
		gov.NewAppModule(appCodec, govKeeper, accountKeeper, bankKeeper),
		mint.NewAppModule(appCodec, mintKeeper, accountKeeper),
		staking.NewAppModule(appCodec, stakingKeeper, accountKeeper, bankKeeper),
		distr.NewAppModule(appCodec, distrKeeper, accountKeeper, bankKeeper, stakingKeeper),
		slashing.NewAppModule(appCodec, slashingKeeper, accountKeeper, bankKeeper, stakingKeeper),
		params.NewAppModule(paramsKeeper),
		evidence.NewAppModule(evidenceKeeper),
		ibc.NewAppModule(&ibcKeeper),
		ibcTransferAppModule,
		evm.NewAppModule(evmKeeper, accountKeeper),
		feemarket.NewAppModule(feemarketKeeper),
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
	maxGasWanted := cast.ToUint64(appOpts.Get(ethermintsrvflags.EVMMaxTxGasWanted))
	options := cantoante.HandlerOptions{
		AccountKeeper:   accountKeeper,
		BankKeeper:      bankKeeper,
		IBCKeeper:       &ibcKeeper,
		FeeMarketKeeper: feemarketKeeper,
		StakingKeeper:   stakingKeeper,
		EvmKeeper:       evmKeeper,
		FeegrantKeeper:  nil,
		SignModeHandler: encodingConfig.TxConfig.SignModeHandler(),
		SigGasConsumer:  SigVerificationGasConsumer,
		Cdc:             appCodec,
		MaxTxGasWanted:  maxGasWanted,
	}
	if err := options.Validate(); err != nil {
		panic(fmt.Errorf("invalid antehandler options: %v", err))
	}
	ah := cantoante.NewAnteHandler(options)

	// Create the lockup AnteHandler, to ensure sufficient decentralization before funds may be transferred
	lockupAnteHandler := lockup.NewWrappedLockupAnteHandler(ah, lockupKeeper, appCodec)
	app.SetAnteHandler(lockupAnteHandler)

	// Register the configured upgrades for the upgrade module
	app.registerUpgradeHandlers()

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
	out := app.mm.BeginBlock(ctx, req)
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
	return app.mm.EndBlock(ctx, req)
}

// InitChainer deserializes the given chain genesis state, registers in-place upgrade migrations, and delegates
// the ABCI InitGenesis execution to the ModuleManager
func (app *AltheaApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	app.upgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())

	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
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
func (app *AltheaApp) GetKey(storeKey string) *sdk.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *AltheaApp) GetTKey(storeKey string) *sdk.TransientStoreKey {
	return app.tKeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *AltheaApp) GetMemKey(storeKey string) *sdk.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
// Reading params is fine, but they should be updated by governace proposal
func (app *AltheaApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.paramsKeeper.GetSubspace(moduleName)
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
	// SDK /node_info, /syncing, /blocks, and /validatorsets REST endpoints
	rpc.RegisterRoutes(clientCtx, apiSvr.Router)

	// Note: Delegates requests to the EVM if given a hash variable with a leading "0x"
	evmrest.RegisterTxRoutes(clientCtx, apiSvr.Router) // Cosmos and EVM /txs REST endpoints

	// Note: The Cosmos REST registration has been replaced by evmrest's
	// authrest.RegisterTxRoutes(clientCtx, apiSvr.Router) // Cosmos /txs REST endpoints

	// GRPC endpoints under /cosmos.base.tendermint.v1beta1.Service
	// including GetNodeInfo, GetSyncing, GetLatestBlock, GetBlockByHeight, GetLatestValidatorSet, GetValidatorSetByHeight
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// GRPC endpoints under /cosmos.tx.v1beta1.Service
	// including Simulate, GetTx, BroadcastTx, GetTxsEvent, GetBlockWithTxs
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register all REST routes declared by modules in the ModuleBasics
	ModuleBasics.RegisterRESTRoutes(clientCtx, apiSvr.Router)
	// Register all GRPC routes declared by modules in the ModuleBasics
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// TODO: build the custom swagger files and add here?
	if apiConfig.Swagger {
		RegisterSwaggerAPI(clientCtx, apiSvr.Router)
	}
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

// RegisterTxService registers all Protobuf-based Tx receiving gRPC services based on what is registered in the
// interface registry. These are stapled on to the baseapp's gRPC Query router
func (app *AltheaApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService registers the /cosmos.base.tendermint.v1beta1.Service query endpoints on the baseapp's
// gRPC query router
func (app *AltheaApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.interfaceRegistry)
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
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey sdk.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govtypes.ParamKeyTable())
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibchost.ModuleName)
	paramsKeeper.Subspace(lockuptypes.ModuleName)
	paramsKeeper.Subspace(microtxtypes.ModuleName)
	paramsKeeper.Subspace(evmtypes.ModuleName)
	paramsKeeper.Subspace(erc20types.ModuleName)
	paramsKeeper.Subspace(feemarkettypes.ModuleName)

	return paramsKeeper
}

// registerUpgradeHandlers registers in-place upgrades, which are faster and easier than genesis-based upgrades
func (app *AltheaApp) registerUpgradeHandlers() {
	// No op
}
