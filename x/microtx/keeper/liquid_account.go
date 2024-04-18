package keeper

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"

	erc20types "github.com/Canto-Network/Canto/v5/x/erc20/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/AltheaFoundation/althea-L1/x/microtx/types"
)

// Default gas limit for eth txs from the module account
var DefaultGasLimit uint64 = 30000000

// The ID for the Account token, the only token controlled by a LiquidInfrastructureNFT
// Note when using this value as an argument, the EVM requires it to be a *big.Int, the non-pointer
// type will cause execution failure
var AccountId *big.Int = big.NewInt(1)

var CurrentNFTVersion *big.Int = big.NewInt(1)

// DoLiquify will deploy a LiquidInfrastructureNFT smart contract for the given account.
// The token will then be transferred to the given account and live under its control.
// Transfer to another owner requires interacting with the EVM
func (k Keeper) DoLiquify(
	ctx sdk.Context,
	account sdk.AccAddress,
) (common.Address, error) {
	nftAddr, err := k.deployLiquidInfrastructureNFTContract(ctx, account)
	if err != nil {
		return common.Address{}, sdkerrors.Wrapf(types.ErrContractDeployment,
			"EVM::Liquify error deploying LiquidInfrastructureNFT: %s", err.Error())
	}
	if err := k.addLiquidInfrastructureEntry(ctx, account, nftAddr); err != nil {
		return common.Address{}, sdkerrors.Wrapf(err, "unable to map bech32 -> NFT address")
	}

	ctx.EventManager().EmitEvent(types.NewEventLiquify(account.String(), nftAddr))
	k.Logger(ctx).Info("Account Liquified", "account", account.String(), "owner", account.String(), "nft", nftAddr.Hex())

	return nftAddr, nil
}

// deployLiquidInfrastructureNFTContract deploys an NFT contract for the given `account` and then transfers ownership of the
// underlying NFT to the given `account`
func (k Keeper) deployLiquidInfrastructureNFTContract(ctx sdk.Context, account sdk.AccAddress) (common.Address, error) {
	contract, err := k.DeployContract(ctx, account, types.LiquidInfrastructureNFT, account.String())
	if err != nil {
		return common.Address{}, sdkerrors.Wrap(err, "liquid infrastructure account contract deployment failed")
	}

	version, err := k.queryLiquidInfrastructureContractVersion(ctx, contract)
	if err != nil {
		return common.Address{}, sdkerrors.Wrap(err, "could not query NFT version")
	}
	if version.Cmp(CurrentNFTVersion) != 0 {
		return common.Address{}, sdkerrors.Wrapf(err, "expected contract with version %v, got %v", CurrentNFTVersion, version)
	}

	return contract, nil
}

// addLiquidInfrastructureEntry Sets a new Liquid Infrastructure Account entry in the bech32 -> EVM NFT address mapping
// accAddress - The account to Liquify
// nftAddress - The deployed LiquidInfrastructureNFT contract address
func (k Keeper) addLiquidInfrastructureEntry(ctx sdk.Context, accAddress sdk.AccAddress, nftAddress common.Address) error {
	store := ctx.KVStore(k.storeKey)
	key := types.GetLiquidAccountKey(accAddress)

	if store.Has(key) {
		return sdkerrors.Wrapf(types.ErrContractDeployment, "account %v already liquified", accAddress.String())
	}

	store.Set(key, nftAddress.Bytes())
	return nil
}

// queryLiquidInfrastructureOwner is used by the module to provide a convenient query interface, it calls the ERC721 ownerOf() function
// with the only token used by LiquidInfrastructureNFTs (0x1) and returns the owner's eth address
func (k Keeper) queryLiquidInfrastructureOwner(ctx sdk.Context, nftAddress common.Address) (*common.Address, error) {
	// ABI: ownerOf(uint256 tokenId) public view virtual override returns (address)
	var args = ToMethodArgs(&AccountId)
	res, err := k.QueryEVM(ctx, "ownerOf", types.LiquidInfrastructureNFT, types.ModuleEVMAddress, &nftAddress, args...)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "unable to call ownerOf with arg0=1")
	}
	owner := common.BytesToAddress(res.Ret)
	return &owner, nil
}

// queryLiquidInfrastructureContractVersion fetches the `Version()` of the LiquidInfrastructureNFT deployed at `nftAddress`
func (k Keeper) queryLiquidInfrastructureContractVersion(ctx sdk.Context, nftAddress common.Address) (*big.Int, error) {
	// ABI: uint256 public constant Version
	res, err := k.QueryEVM(ctx, "Version", types.LiquidInfrastructureNFT, types.ModuleEVMAddress, &nftAddress)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "unable to call Versionw with no args")
	}
	version := big.NewInt(0).SetBytes(res.Ret)
	return version, nil
}

// queryLiquidInfrastructureThresholds is used by the module to control liquid infrastructure account balances, it calls the
// LiquidInfrastructureNFT getThresholds() function and formats the output into a useable type
func (k Keeper) queryLiquidInfrastructureThresholds(ctx sdk.Context, nftAddress common.Address) ([]types.LiquidAccountThreshold, error) {
	// ABI: getThresholds() public virtual view returns (address[] memory, uint256[] memory)
	res, err := k.QueryEVM(ctx, "getThresholds", types.LiquidInfrastructureNFT, types.ModuleEVMAddress, &nftAddress)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "unable to call getThresholds with no arguments")
	}

	// Use the ABI to unpack values. Expecting ([addr, ...], [uint256, ...])
	values, err := types.LiquidInfrastructureNFT.ABI.Unpack("getThresholds", res.Ret)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "Unable to unpack the getThresholds() response")
	}
	if len(values) != 2 {
		return nil, fmt.Errorf("expected to get a 2 tuple response, instead got %v values", len(values))
	}
	addresses, ok := values[0].([]common.Address)
	if !ok {
		return nil, fmt.Errorf("go-ethereum ABI decoder returned invalid response in position 0, expected []common.Address, but found %T", values[0])
	}

	amounts, ok := values[1].([]*big.Int)
	if !ok {
		return nil, fmt.Errorf("go-ethereum ABI decoder returned invalid response in position 1, expected []*big.Int, but found %T", values[1])
	}

	if len(addresses) != len(amounts) {
		return nil, fmt.Errorf("go-ethereum ABI decoder read incorrect number of values (%d addresses vs %d amounts)", len(addresses), len(amounts))
	}

	var output []types.LiquidAccountThreshold
	for i := 0; i < len(addresses); i++ {
		amount := amounts[i]
		if amount == nil {
			return nil, fmt.Errorf("discovered invalid amount at %d -th location in thresholds result", i)
		}
		output = append(output, types.NewLiquidAccountThreshold(addresses[i], *amount))
	}

	return output, nil
}

// GetLiquidAccountEntry fetches the LiquidInfrastructureNFT contract address for the given `accAddress`
// returns nil, ErrNoLiquidAccount if `accAddress` has not been liquified (no record found)
func (k Keeper) GetLiquidAccountEntry(ctx sdk.Context, accAddress sdk.AccAddress) (*common.Address, error) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetLiquidAccountKey(accAddress)

	contractAddressBz := store.Get(key)
	if len(contractAddressBz) == 0 {
		return nil, types.ErrNoLiquidAccount
	}

	contractAddress := common.BytesToAddress(contractAddressBz)
	return &contractAddress, nil
}

// GetLiquidAccount fetches info about a Liquid Infrastructure Account
// returns nil, ErrNoLiquidAccount if `accAddress` has not been liquified (no record found)
func (k Keeper) GetLiquidAccount(ctx sdk.Context, accAddress sdk.AccAddress) (*types.LiquidInfrastructureAccount, error) {
	contractAddress, err := k.GetLiquidAccountEntry(ctx, accAddress)
	if err != nil {
		return nil, err
	}

	owner, err := k.queryLiquidInfrastructureOwner(ctx, *contractAddress)
	if err != nil {
		return nil, err
	}

	return &types.LiquidInfrastructureAccount{
		Owner:      EVMToSDKAddress(*owner).String(),
		Account:    accAddress.String(),
		NftAddress: contractAddress.Hex(),
	}, nil
}

// IsLiquidAccount checks if the input account is a Liquid Infrastructure Account
func (k Keeper) IsLiquidAccount(ctx sdk.Context, account sdk.AccAddress) bool {
	isLiquidAccount, _ := k.IsLiquidAccountWithValue(ctx, account)
	return isLiquidAccount
}

// IsLiquidAccountWithValue checks if the input account is a Liquid Infrastructure Account and returns the account's nft contract
func (k Keeper) IsLiquidAccountWithValue(ctx sdk.Context, account sdk.AccAddress) (bool, *common.Address) {
	taEntry, err := k.GetLiquidAccountEntry(ctx, account)
	k.Logger(ctx).Debug("IsLiquidAccount", "taEntry", taEntry)
	return err == nil && taEntry != nil, taEntry
}

// GetLiquidAccountByNFTAddress fetches info about a LiquidAccount given the address of the LiquidInfrastructureNFT in the EVM
// returns nil, ErrNoLiquidAccount if `nftAddress` is not a record for any Liquid Infrastructure Account
func (k Keeper) GetLiquidAccountByNFTAddress(ctx sdk.Context, nftAddress common.Address) (*types.LiquidInfrastructureAccount, error) {
	var liquidAccount *types.LiquidInfrastructureAccount = nil

	k.IterateLiquidAccounts(
		ctx,
		func(_ []byte, accAddress sdk.AccAddress, owner common.Address, nftAddr common.Address) (stop bool) {
			if nftAddr == nftAddress {
				liquidAccount = &types.LiquidInfrastructureAccount{
					Owner:      EVMToSDKAddress(owner).String(),
					Account:    accAddress.String(),
					NftAddress: nftAddr.Hex(),
				}
				return true
			}
			return false
		},
	)

	return liquidAccount, nil
}

// GetLiquidAccountsByCosmosOwner fetches info about a Liquid Infrastructure Account given the bech32 address of the LiquidInfrastructureNFT holder
// returns nil, ErrNoLiquidAccount if `ownerAddress` has no LiquidInfrastructureNFTs (no record found)
func (k Keeper) GetLiquidAccountsByCosmosOwner(ctx sdk.Context, ownerAddress sdk.AccAddress) ([]*types.LiquidInfrastructureAccount, error) {
	owner := SDKToEVMAddress(ownerAddress)
	return k.GetLiquidAccountsByEVMOwner(ctx, owner)
}

// GetLiquidAccountsByEVMOwner fetches info about a Liquid Infrastructure Account given the EVM address of the LiquidInfrastructureNFT holder
// returns nil, ErrNoLiquidAccount if `ownerAddress` has no LiquidInfrastructureNFTs (no record found)
func (k Keeper) GetLiquidAccountsByEVMOwner(ctx sdk.Context, ownerAddress common.Address) ([]*types.LiquidInfrastructureAccount, error) {
	var liquidAccounts []*types.LiquidInfrastructureAccount

	k.IterateLiquidAccounts(
		ctx,
		func(_ []byte, accAddress sdk.AccAddress, owner common.Address, nftAddr common.Address) (stop bool) {
			if owner == ownerAddress {
				liquidAccounts = append(liquidAccounts, &types.LiquidInfrastructureAccount{
					Owner:      EVMToSDKAddress(owner).String(),
					Account:    accAddress.String(),
					NftAddress: nftAddr.Hex(),
				})
			}
			return true
		},
	)

	return liquidAccounts, nil
}

// GetLiquidAccountByNFTAddress fetches info about a Liquid Infrastructure Account given the address of the LiquidInfrastructureNFT in the EVM
func (k Keeper) CollectLiquidAccounts(ctx sdk.Context) ([]*types.LiquidInfrastructureAccount, error) {
	var liquidAccounts []*types.LiquidInfrastructureAccount

	k.IterateLiquidAccounts(
		ctx,
		func(_ []byte, accAddress sdk.AccAddress, owner common.Address, nftAddr common.Address) (stop bool) {
			liquidAccounts = append(liquidAccounts, &types.LiquidInfrastructureAccount{
				Owner:      EVMToSDKAddress(owner).String(),
				Account:    accAddress.String(),
				NftAddress: nftAddr.Hex(),
			})
			return false
		},
	)

	return liquidAccounts, nil
}

// IterateLiquidAccounts calls the provided callback `cb` on every discovered Liquid Infrastructure Account entry. Return stop=true to end iteration early.
func (k Keeper) IterateLiquidAccounts(ctx sdk.Context, cb func(key []byte, accAddress sdk.AccAddress, owner common.Address, nftAddress common.Address) (stop bool)) {
	pStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.LiquidAccountKey)
	iterator := pStore.Iterator(nil, nil) // Iterate through all entries
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		accAddressBz := key[len(types.LiquidAccountKey):]
		accAddress := sdk.AccAddress(accAddressBz)
		nftAddressBz := iterator.Value()
		nftAddress := common.BytesToAddress(nftAddressBz)
		owner, err := k.queryLiquidInfrastructureOwner(ctx, nftAddress)
		if err != nil {
			break
		}

		if cb(key, accAddress, *owner, nftAddress) {
			break
		}
	}
}

// RedirectLiquidAccountExcessBalance will check if this account is a Liquid Infrastructure Account,
// then may funnel any excess balance to the registered LiquidInfrastructureNFT depending on the set thresholds
// If no threshold is set for `changedErc20`, its balance WILL NOT be sent to the NFT
func (k Keeper) RedirectLiquidAccountExcessBalance(ctx sdk.Context, account sdk.AccAddress, changedErc20 common.Address) error {
	logger := k.Logger(ctx)
	logger.Debug("Enter RedirectLiquidAccountExcessBalances", "receiver", account.String(), "changedBalance", changedErc20)
	isLiquidAccount, nft := k.IsLiquidAccountWithValue(ctx, account)
	if !isLiquidAccount {
		logger.Debug("receiver not liquified")
		return nil // Do nothing for non-Liquid Infrastructure Accounts
	}
	// If the account IS liquified, it MUST have its NFT registered
	if nft == nil {
		panic("discovered nil nft address in liquid account entry")
	}
	logger.Debug("Discovered Liquid Infrastructure Account->NFT entry", "account", account.String(), "nft", nft.Hex())

	thresholds, err := k.queryLiquidInfrastructureThresholds(ctx, *nft)
	if err != nil {
		panic(fmt.Errorf("failed to query evm for liquid account thresholds: %s", err.Error()))
	}

	var redirectedAmount sdk.Coin
	threshold := types.FindThresholdForERC20(thresholds, changedErc20)
	logger.Debug("Found threshold for modified balance", "threshold", threshold, "changedBalance", changedErc20)

	pair, found := k.erc20Keeper.GetTokenPair(ctx, k.erc20Keeper.GetTokenPairID(ctx, threshold.Token.Hex()))
	if !found || !pair.Enabled {
		// This should've been checked earlier in msg_server.go
		panic("threshold token pair does not exist or is inactive, should have been caught earlier")
	}
	logger.Debug("Found pair for threshold token", "token", threshold.Token.Hex())
	balance := k.bankKeeper.GetBalance(ctx, account, pair.Denom)
	logger.Debug("Found new balance of token", "balance", balance.String())

	balanceInExcess := balance.Amount.BigInt().Cmp(&threshold.Amount) > 0
	logger.Debug("Checking if balance is in excess of threshold", "balance", balance.String(), "threshold", threshold.Amount.String(), "exceeded", balanceInExcess)
	if balanceInExcess {
		logger.Debug("Redirecting balance to nft", "account", account.String(), "nft", nft.Hex(), "exceeded", balanceInExcess)
		redirected, err := k.RedirectBalanceToToken(ctx, account, *nft, balance, threshold.Amount)
		if err != nil {
			return err
		}
		logger.Debug("Redirected to nft", "amount", redirected.String())
		redirectedAmount = *redirected
	}

	logger.Debug("Emitting balance redirect event to log")
	// Emit an event for the block's event log
	ctx.EventManager().EmitEvent(
		types.NewEventBalanceRedirect(account.String(), redirectedAmount),
	)

	return nil
}

// RedirectBalanceToToken will funnel all excess amounts of `currBalance` (based on `thresholdAmount`) to `nft`
// it creates a MsgConvertCoin and uses the erc20Keeper to execute it
func (k Keeper) RedirectBalanceToToken(
	ctx sdk.Context,
	account sdk.AccAddress,
	nft common.Address,
	currBalance sdk.Coin,
	thresholdAmount big.Int,
) (*sdk.Coin, error) {
	logger := k.Logger(ctx)
	logger.Debug("Enter RedirectBalanceToToken", "account", account.String(), "nft", nft.Hex(), "currBalance", currBalance.String(), "thresholdAmount", thresholdAmount.String())
	excessBalance := currBalance.Amount.Sub(sdk.NewIntFromBigInt(&thresholdAmount))
	if excessBalance.IsNegative() {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, "attempted to funnel insufficient balance")
	}
	funnelAmount := sdk.NewCoin(currBalance.Denom, excessBalance)
	context := sdk.WrapSDKContext(ctx)
	msgConvertCoin := erc20types.NewMsgConvertCoin(funnelAmount, nft, account)
	logger.Debug("About to redirect balance via convert coin", "msg", msgConvertCoin.String())
	if err := msgConvertCoin.ValidateBasic(); err != nil {
		return nil, sdkerrors.Wrap(err, "generated invalid convert coin msg")
	}
	logger.Debug("Calling convert coin")
	_, err := k.erc20Keeper.ConvertCoin(context, msgConvertCoin)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "unable to funnel excess balance via x/erc20 convert coin")
	}
	logger.Debug("Redirected balanace via convert coin")
	return &funnelAmount, nil
}

// TODO: Move all this to a shared EVM utils package at the project root. It makes the most sense in ethermint but it's not there for some reason.
// -----------------------------------------------------------------------------------------------------------------------------------------------

// SDKToEVMAddress converts `addr` to its EVM equivalent
func SDKToEVMAddress(addr sdk.AccAddress) common.Address {
	return common.BytesToAddress(addr.Bytes())
}

func EVMToSDKAddress(addr common.Address) sdk.AccAddress {
	return sdk.AccAddress(addr.Bytes())
}

// IsEthermintAccount indicates the given account has a registered Ethermint public key
func IsEthermintAccount(account authtypes.AccountI) bool {
	pk := account.GetPubKey()
	return pk != nil && pk.Type() == ethsecp256k1.KeyType
}

// ToMethodArgs conveniently converts EVM method call args to an array
func ToMethodArgs(args ...interface{}) []interface{} {
	return args
}

// DeployContract will deploy an arbitrary smart-contract. It takes the compiled contract object as
// well as an arbitrary number of arguments which will be supplied to the contructor. All contracts deployed
// are deployed by the deployer account.
func (k Keeper) DeployContract(
	ctx sdk.Context,
	deployer sdk.AccAddress,
	contract evmtypes.CompiledContract,
	args ...interface{},
) (common.Address, error) {
	deployerEVM := SDKToEVMAddress(deployer)
	// pack constructor arguments according to compiled contract's abi
	// method name is nil in this case, we are calling the constructor
	ctorArgs, err := contract.ABI.Pack("", args...)
	if err != nil {
		return common.Address{}, sdkerrors.Wrapf(types.ErrContractDeployment,
			"EVM::DeployContract error packing data: %s", err.Error())
	}

	// pack method data into byte string, enough for bin and constructor arguments
	data := make([]byte, len(contract.Bin)+len(ctorArgs))

	// copy bin into data, and append to data the constructor arguments
	copy(data[:len(contract.Bin)], contract.Bin)

	// copy constructor args last
	copy(data[len(contract.Bin):], ctorArgs)

	// retrieve sequence number first to derive address if not by CREATE2
	nonce, err := k.accountKeeper.GetSequence(ctx, deployer)
	if err != nil {
		return common.Address{},
			sdkerrors.Wrapf(types.ErrContractDeployment,
				"EVM::DeployContract error retrieving nonce: %s", err.Error())
	}

	amount := big.NewInt(0)
	_, err = k.CallEVM(ctx, deployerEVM, nil, amount, data, true)
	if err != nil {
		return common.Address{},
			sdkerrors.Wrapf(types.ErrContractDeployment,
				"EVM::DeployContract error deploying contract: %s", err.Error())
	}

	// Derive the newly created module smart contract using the module address and nonce
	return crypto.CreateAddress(deployerEVM, nonce), nil
}

// CallMethod is a function to interact with a contract once it is deployed. It inputs the method name on the
// smart contract, the compiled smart contract, the address from which the tx will be made, the contract address,
// the amount to be supplied in msg.value, and finally an arbitrary number of arguments that should be supplied to the
// function method.
func (k Keeper) CallMethod(
	ctx sdk.Context,
	method string, // E.g. transfer when calling transfer(from address, to address, amount uint256)
	contract evmtypes.CompiledContract,
	from common.Address,
	contractAddr *common.Address,
	amount *big.Int,
	args ...interface{}, // E.g. [common.Address, common.Address, *big.Int]... for the above transfer(...) example
) (*evmtypes.MsgEthereumTxResponse, error) {
	// pack method args
	data, err := contract.ABI.Pack(method, args...)
	if err != nil {
		return nil, sdkerrors.Wrapf(types.ErrContractCall, "EVM::CallMethod there was an issue packing the arguments into the method signature: %s", err.Error())
	}

	// call method
	resp, err := k.CallEVM(ctx, from, contractAddr, amount, data, true)
	if err != nil {
		return nil, sdkerrors.Wrapf(types.ErrContractCall, "EVM::CallMethod error applying message: %s", err.Error())
	}

	return resp, nil
}

// CallEVM performs a EVM transaction given the from address, the to address, amount to be sent, data and
// whether to commit the tx in the EVM keeper.
func (k Keeper) CallEVM(
	ctx sdk.Context,
	from common.Address,
	to *common.Address,
	amount *big.Int,
	data []byte,
	commit bool,
) (*evmtypes.MsgEthereumTxResponse, error) {
	nonce, err := k.accountKeeper.GetSequence(ctx, from.Bytes())
	if err != nil {
		return nil, err
	}

	cosmosLimit := ctx.GasMeter().Limit()
	gasUsed := ctx.GasMeter().GasConsumed()
	if cosmosLimit <= gasUsed {
		return nil, sdkerrors.Wrap(sdkerrors.ErrOutOfGas, "insufficient gas")
	}
	gasLimit := cosmosLimit - gasUsed

	// Create the EVM msg
	msg := ethtypes.NewMessage(
		from,
		to,
		nonce,
		amount,        // amount
		gasLimit,      // gasLimit
		big.NewInt(0), // gasPrice
		big.NewInt(0), // gasFeeCap
		big.NewInt(0), // gasTipCap
		data,
		ethtypes.AccessList{}, // AccessList
		!commit,
	)

	// Apply the msg to the EVM keeper
	res, err := k.evmKeeper.ApplyMessage(ctx, msg, evmtypes.NewNoOpTracer(), commit)
	if err != nil {
		return nil, err
	}

	// ApplyMessage does not consume gas, so we need to consume the gas used within the EVM
	// This is completely different than the ethermint gas consumption, where an AnteHandler consumes all the gas
	// the EVM Tx is allowed to use, and then refunds remaining gas after execution.
	ctx.GasMeter().ConsumeGas(res.GasUsed, "EVM call")

	if res.Failed() {
		return nil, sdkerrors.Wrap(evmtypes.ErrVMExecution, res.VmError)
	}
	return res, nil
}

func (k Keeper) QueryEVM(
	ctx sdk.Context,
	method string,
	contract evmtypes.CompiledContract,
	querier common.Address,
	contractAddr *common.Address,
	args ...interface{},
) (*evmtypes.MsgEthereumTxResponse, error) {
	gasLimit := DefaultGasLimit
	nonce, err := k.accountKeeper.GetSequence(ctx, types.ModuleAddress.Bytes())
	if err != nil {
		return nil, err
	}
	// pack method args
	data, err := contract.ABI.Pack(method, args...)
	if err != nil {
		return nil, sdkerrors.Wrapf(types.ErrContractCall, "EVM::CallMethod there was an issue packing the arguments into the method signature: %s", err.Error())
	}

	commit := false // Do not modify state, just simulate and return results
	// Create the EVM msg
	msg := ethtypes.NewMessage(
		querier,
		contractAddr,
		nonce,
		big.NewInt(0), // amount
		gasLimit,      // gasLimit
		big.NewInt(0), // gasPrice
		big.NewInt(0), // gasFeeCap
		big.NewInt(0), // gasTipCap
		data,
		ethtypes.AccessList{}, // AccessList
		!commit,
	)

	// Apply the msg to the EVM keeper
	res, err := k.evmKeeper.ApplyMessage(ctx, msg, evmtypes.NewNoOpTracer(), commit)
	if err != nil {
		return nil, err
	}

	if res.Failed() {
		return nil, sdkerrors.Wrap(evmtypes.ErrVMExecution, res.VmError)
	}
	return res, nil
}
