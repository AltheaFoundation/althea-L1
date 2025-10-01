// The root command contains everything under `$ althea`, notably the `tx` and
// `q` commands, the `start` command for validators, and all defaults are set here
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cosmos/go-bip39"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	cfg "github.com/tendermint/tendermint/config"
	tmcli "github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmtypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"

	// EVM

	ethermintclient "github.com/evmos/ethermint/client"
	"github.com/evmos/ethermint/ethereum/eip712"
	ethermintserver "github.com/evmos/ethermint/server"
	ethermintserverconfig "github.com/evmos/ethermint/server/config"
	ethermintserverflags "github.com/evmos/ethermint/server/flags"

	// Althea
	althea "github.com/AltheaFoundation/althea-L1/app"
	"github.com/AltheaFoundation/althea-L1/app/params"
	altheacfg "github.com/AltheaFoundation/althea-L1/config"
	"github.com/AltheaFoundation/althea-L1/crypto/keyring"
)

const EnvPrefix = "althea"

type printInfo struct {
	Moniker    string          `json:"moniker" yaml:"moniker"`
	ChainID    string          `json:"chain_id" yaml:"chain_id"`
	NodeID     string          `json:"node_id" yaml:"node_id"`
	GenTxsDir  string          `json:"gentxs_dir" yaml:"gentxs_dir"`
	AppMessage json.RawMessage `json:"app_message" yaml:"app_message"`
}

func newPrintInfo(moniker, chainID, nodeID, genTxsDir string, appMessage json.RawMessage) printInfo {
	return printInfo{
		Moniker:    moniker,
		ChainID:    chainID,
		NodeID:     nodeID,
		GenTxsDir:  genTxsDir,
		AppMessage: appMessage,
	}
}

func displayInfo(info printInfo) error {
	out, err := json.MarshalIndent(info, "", " ")
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(os.Stderr, "%s\n", string(sdk.MustSortJSON(out))); err != nil {
		return err
	}

	return nil
}

// NewRootCmd creates a new root command for althea. It is called once in the
// main function. The name of the binary is controlled by the Makefile in the
// project root. Most everything else is controlled here via cobra.
// The module subcommands are automatically created by passing the ModuleBasics
// (from app.go) in initRootCmd().
// The module subcommands should be in x/{module}/client/cli
func NewRootCmd() (*cobra.Command, params.EncodingConfig) {
	encodingConfig := althea.MakeEncodingConfig()
	initClientCtx := client.Context{}.
		// Encoding and interfaces
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastBlock).
		WithHomeDir(althea.DefaultNodeHome).
		WithKeyringOptions(keyring.Option()).
		WithViper(EnvPrefix)

	eip712.SetEncodingConfig(simappparams.EncodingConfig(encodingConfig))

	rootCmd := &cobra.Command{
		Use:   althea.Name,
		Short: "Althea L1: Submit transactions or run a validator",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return fmt.Errorf("unable to update context with client.toml config: %v", err)
			}

			// Calls ReadPersistentCmdFlags and SetCmdClientContext
			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			altheaAppTemplate, altheaAppConfig := initAppConfig()
			altheaTMConfig := initTendermintConfig()

			// Takes all these configurations and applies them, additionally configuring Tendermint
			// to adhere to these desires
			return server.InterceptConfigsPreRunHandler(cmd, altheaAppTemplate, altheaAppConfig, altheaTMConfig)
		},
	}

	initRootCmd(rootCmd, &encodingConfig)

	return rootCmd, encodingConfig
}

// initAppConfig defines the default configuration. These defaults can be overridden via an
// app.toml file or with flags provided on the command line
func initAppConfig() (string, interface{}) {
	// DEFAULT SERVER CONFIGURATIONS
	appTempl, appCfg := ethermintserverconfig.AppConfig(altheacfg.BaseDenom)

	return appTempl, appCfg
}

// initTendermintConfig helps to override default Tendermint Config values.
// return tmcfg.DefaultConfig if no custom configuration is required for the application.
func initTendermintConfig() *cfg.Config {
	cfg := cfg.DefaultConfig()

	// these values put a higher strain on node memory
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	return cfg
}

// Execute executes the root command.
func Execute(rootCmd *cobra.Command, defaultHome string) error {
	// Create and set a client.Context on the command's Context. During the pre-run
	// of the root command, a default initialized client.Context is provided to
	// seed child command execution with values such as AccountRetriver, Keyring,
	// and a Tendermint RPC. This requires the use of a pointer reference when
	// getting and setting the client.Context. Ideally, we utilize
	// https://github.com/spf13/cobra/pull/1118.
	srvCtx := server.NewDefaultContext()
	ctx := context.Background()
	ctx = context.WithValue(ctx, client.ClientContextKey, &client.Context{})
	ctx = context.WithValue(ctx, server.ServerContextKey, srvCtx)

	rootCmd.PersistentFlags().String("log_level", "info", "The logging level in the format of <module>:<level>,...")
	rootCmd.PersistentFlags().String(flags.FlagLogFormat, cfg.LogFormatPlain, "The logging format (json|plain)")

	executor := tmcli.PrepareBaseCmd(rootCmd, "", defaultHome)
	return executor.ExecuteContext(ctx)
}

// Setup all of the subcommands for the root command
func initRootCmd(rootCmd *cobra.Command, encodingConfig *params.EncodingConfig) {
	rootCmd.AddCommand(
		// ValidateChainID will make sure the configured chain id adheres to strings like althea_1234-1
		ethermintclient.ValidateChainID(InitCmd(althea.ModuleBasics, althea.DefaultNodeHome)),
		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{}, althea.DefaultNodeHome),
		genutilcli.GenTxCmd(althea.ModuleBasics, encodingConfig.TxConfig, banktypes.GenesisBalancesIterator{}, althea.DefaultNodeHome),
		ValidateGenesisCmd(althea.ModuleBasics),
		AddGenesisAccountCmd(althea.DefaultNodeHome),
		tmcli.NewCompletionCmd(rootCmd, true),
		testnetCmd(althea.ModuleBasics, banktypes.GenesisBalancesIterator{}),
		debug.Cmd(),  // Output useful info about keys
		config.Cmd(), // Set config options one by one
	)

	ac := appCreator{encodingConfig}
	// The ethermint server commands perform a lot of modifications on top of the base ones, notably setting up the
	// EVM JSONRPC server, tx indexer, and some various improvements like closing the DB automatically
	ethermintserver.AddCommands(rootCmd, ethermintserver.NewDefaultStartOptions(ac.newApp, althea.DefaultNodeHome), ac.createSimappAndExport, addModuleInitFlags)

	rootCmd.AddCommand(ethermintserver.NewIndexTxCmd())

	// add keybase, auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		rpc.StatusCommand(),
		// queryCommand registers the query subcommands by looking at ModuleBasics
		queryCommand(),
		// txCommand registers the tx subcommands by looking at ModuleBasics
		txCommand(),
		// Adds the same commands as sdk keys.Commands(), but enables dry run and sets the default keytype to eth_secp256k1
		// keys.Commands(althea.DefaultNodeHome),
		ethermintclient.KeyCommands(althea.DefaultNodeHome),
	)

	rootCmd, err := ethermintserverflags.AddTxFlags(rootCmd)
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(server.RosettaCommand(encodingConfig.InterfaceRegistry, encodingConfig.Codec))
}

// InitCmd returns a command that initializes all files needed for Tendermint
// and the respective application.
// Note that this is mostly a copy of the default InitCmd found in genutil, however we need to overwrite the default
// chain id to one which will not cause a panic in ethermint/x/evm/genesis.go
func InitCmd(mbm module.BasicManager, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize private validator, p2p, genesis, and application configuration files",
		Long:  `Initialize validators's and node's configuration files.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			cdc := clientCtx.Codec

			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)

			chainID, _ := cmd.Flags().GetString(flags.FlagChainID)
			if chainID == "" {
				chainID = altheacfg.DefaultChainID()
			}

			// Get bip39 mnemonic
			var mnemonic string
			recover, _ := cmd.Flags().GetBool(genutilcli.FlagRecover)
			if recover {
				inBuf := bufio.NewReader(cmd.InOrStdin())
				value, err := input.GetString("Enter your bip39 mnemonic", inBuf)
				if err != nil {
					return err
				}

				mnemonic = value
				if !bip39.IsMnemonicValid(mnemonic) {
					return errors.New("invalid mnemonic")
				}
			}

			nodeID, _, err := genutil.InitializeNodeValidatorFilesFromMnemonic(config, mnemonic)
			if err != nil {
				return err
			}

			config.Moniker = args[0]

			genFile := config.GenesisFile()
			overwrite, _ := cmd.Flags().GetBool(genutilcli.FlagOverwrite)

			if !overwrite && tmos.FileExists(genFile) {
				return fmt.Errorf("genesis.json file already exists: %v", genFile)
			}

			appState, err := json.MarshalIndent(mbm.DefaultGenesis(cdc), "", " ")
			if err != nil {
				return errors.Wrap(err, "Failed to marshall default genesis state")
			}

			genDoc := &tmtypes.GenesisDoc{}
			if _, err := os.Stat(genFile); err != nil {
				if !os.IsNotExist(err) {
					return err
				}
			} else {
				genDoc, err = tmtypes.GenesisDocFromFile(genFile)
				if err != nil {
					return errors.Wrap(err, "Failed to read genesis doc from file")
				}
			}

			genDoc.ChainID = chainID
			genDoc.Validators = nil
			genDoc.AppState = appState

			if err = genutil.ExportGenesisFile(genDoc, genFile); err != nil {
				return errors.Wrap(err, "Failed to export gensis file")
			}

			toPrint := newPrintInfo(config.Moniker, chainID, nodeID, "", appState)

			cfg.WriteConfigFile(filepath.Join(config.RootDir, "config", "config.toml"), config)
			return displayInfo(toPrint)
		},
	}

	cmd.Flags().String(tmcli.HomeFlag, defaultNodeHome, "node's home directory")
	cmd.Flags().BoolP(genutilcli.FlagOverwrite, "o", false, "overwrite the genesis.json file")
	cmd.Flags().Bool(genutilcli.FlagRecover, false, "provide seed phrase to recover existing key instead of creating")
	cmd.Flags().String(flags.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")

	return cmd
}

// Add the --x-crisis-skip-assert-invariants flag, perhaps more in the future
func addModuleInitFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)
}

// Generate the query subcommands for each module in ModuleBasics and other manually
// registered commands
func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetAccountCmd(),
		rpc.ValidatorCommand(),
		rpc.BlockCommand(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),
	)

	althea.ModuleBasics.AddQueryCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

// Generate the tx subcommands for each module in ModuleBasics and other manually
// registered commands
func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		flags.LineBreak,
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetAuxToFeeCommand(),
	)

	althea.ModuleBasics.AddTxCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

// Convenience type to make parameter passing simpler + remove duplication of encoding config
type appCreator struct {
	encCfg *params.EncodingConfig
}

// newApp is an AppCreator used for the start command, anything which must be passed to NewAltheaApp (in app.go)
// can be fetched and added here
func (a appCreator) newApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	baseappOptions := server.DefaultBaseappOptions(appOpts)

	skipUpgradeHeights := make(map[int64]bool)
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	return althea.NewAltheaApp(
		logger, db, traceStore, true, skipUpgradeHeights,
		cast.ToString(appOpts.Get(flags.FlagHome)),
		cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod)),
		*a.encCfg,
		appOpts,
		baseappOptions...,
	)
}

// Creates an app which will not run, instead used for state exports
// Pass -1 to export the current state, any other positive value to export that state (if it is available)
func (a appCreator) createSimappAndExport(
	logger log.Logger, db dbm.DB, traceStore io.Writer, height int64, forZeroHeight bool, jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
) (servertypes.ExportedApp, error) {
	var altheaApp *althea.AltheaApp
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home not set")
	}

	if height != -1 {
		altheaApp = althea.NewAltheaApp(logger, db, traceStore, false, map[int64]bool{}, "", uint(1), *a.encCfg, appOpts)

		if err := altheaApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		altheaApp = althea.NewAltheaApp(logger, db, traceStore, true, map[int64]bool{}, "", uint(1), *a.encCfg, appOpts)
	}

	return altheaApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs)
}
