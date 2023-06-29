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

	"github.com/althea-net/althea-chain/x/microtx/types"
)

// Default gas limit for eth txs from the module account
var DefaultGasLimit uint64 = 30000000

// The ID for the Account token, the only token controlled by a TokenizedAccountNFT
// Note when using this value as an argument, the EVM requires it to be a *big.Int, the non-pointer
// type will cause execution failure
var AccountId *big.Int = big.NewInt(1)

// DoTokenizedAccount will deploy a TokenizedAccountNFT smart contract for the given account.
// The token will then be transferred to the given account and live under its control.
// Transfer to another owner requires interacting with the EVM
func (k Keeper) DoTokenizeAccount(
	ctx sdk.Context,
	account sdk.AccAddress,
) (common.Address, error) {
	nftAddr, err := k.deployTokenizedAccountContract(ctx, account)
	if err != nil {
		return common.Address{}, sdkerrors.Wrapf(types.ErrContractDeployment,
			"EVM::TokenizeAccount error deploying TokenizedAccountNFT: %s", err.Error())
	}
	if err := k.addTokenizedAccountEntry(ctx, account, nftAddr); err != nil {
		return common.Address{}, sdkerrors.Wrapf(err, "unable to map bech32 -> NFT address")
	}

	ctx.EventManager().EmitEvent(types.NewEventTokenizedAccount(account.String(), nftAddr))
	k.Logger(ctx).Info("Account Tokenized", "account", account.String(), "owner", account.String(), "nft", nftAddr.Hex())

	return nftAddr, nil
}

// deployTokenizedAccountContract deploys an NFT contract for the given `account` and then transfers ownership of the
// underlying NFT to the given `account`
func (k Keeper) deployTokenizedAccountContract(ctx sdk.Context, account sdk.AccAddress) (common.Address, error) {
	contract, err := k.DeployContract(ctx, types.TokenizedAccountNFT, account.String())
	if err != nil {
		return common.Address{}, sdkerrors.Wrap(err, "tokenized account contract deployment failed")
	}

	_, err = k.transferTokenizedAccountNFTFromModuleToAddress(ctx, contract, account)
	if err != nil {
		return common.Address{}, sdkerrors.Wrapf(err, "could not transfer nft from %v to %v", types.ModuleEVMAddress.Hex(), SDKToEVMAddress(account).Hex())
	}

	return contract, nil
}

// transferTokenizedAccountNFTFromModuleToAddress calls the ERC721 "transferFrom" method on `contract`,
// passing the arguments needed to transfer the TokenizedAccountNFT Account token
// from the x/microtx module account to to the `newOwner`
func (k Keeper) transferTokenizedAccountNFTFromModuleToAddress(ctx sdk.Context, contract common.Address, newOwner sdk.AccAddress) (*evmtypes.MsgEthereumTxResponse, error) {
	// ABI: transferFrom(   address from,           address to,                uint256 tokenId)
	var args = ToMethodArgs(types.ModuleEVMAddress, SDKToEVMAddress(newOwner), &AccountId)
	return k.CallMethod(ctx, "transferFrom", types.TokenizedAccountNFT, types.ModuleEVMAddress, &contract, &big.Int{}, args...)
}

// addTokenizedAccountEntry Sets a new TokenizedAccount entry in the bech32 -> EVM NFT address mapping
// accAddress - The account to Tokenize
// nftAddress - The deployed TokenizedAccountNFT contract address
func (k Keeper) addTokenizedAccountEntry(ctx sdk.Context, accAddress sdk.AccAddress, nftAddress common.Address) error {
	store := ctx.KVStore(k.storeKey)
	key := types.GetTokenizedAccountKey(accAddress)

	if store.Has(key) {
		return sdkerrors.Wrapf(types.ErrContractDeployment, "account %v already tokenized", accAddress.String())
	}

	store.Set(key, nftAddress.Bytes())
	return nil
}

// queryTokenizedAccountOwner is used by the module to provide a convenient query interface, it calls the ERC721 ownerOf() function
// with the only token used by TokenizedAccountNFTs (0x1) and returns the owner's eth address
func (k Keeper) queryTokenizedAccountOwner(ctx sdk.Context, nftAddress common.Address) (*common.Address, error) {
	// ABI: ownerOf(uint256 tokenId) public view virtual override returns (address)
	var args = ToMethodArgs(&AccountId)
	res, err := k.QueryEVM(ctx, "ownerOf", types.TokenizedAccountNFT, types.ModuleEVMAddress, &nftAddress, args...)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "unable to call ownerOf with arg0=1")
	}
	owner := common.BytesToAddress(res.Ret)
	return &owner, nil
}

// queryTokenizedAccountThresholds is used by the module to control tokenized account balances, it calls the
// TokenizedAccountNFT getThresholds() function and formats the output into a useable type
func (k Keeper) queryTokenizedAccountThresholds(ctx sdk.Context, nftAddress common.Address) ([]types.TokenizedAccountThreshold, error) {
	// ABI: getThresholds() public virtual view returns (address[] memory, uint256[] memory)
	res, err := k.QueryEVM(ctx, "getThresholds", types.TokenizedAccountNFT, types.ModuleEVMAddress, &nftAddress)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "unable to call getThresholds with no arguments")
	}

	// Use the ABI to unpack values. Expecting ([addr, ...], [uint256, ...])
	values, err := types.TokenizedAccountNFT.ABI.Unpack("getThresholds", res.Ret)
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

	var output []types.TokenizedAccountThreshold
	for i := 0; i < len(addresses); i++ {
		amount := amounts[i]
		if amount == nil {
			return nil, fmt.Errorf("discovered invalid amount at %d -th location in thresholds result", i)
		}
		output = append(output, types.NewTokenizedAccountThreshold(addresses[i], *amount))
	}

	return output, nil
}

// GetTokenizedAccountEntry fetches the TokenizedAccountNFT contract address for the given `accAddress`
// returns nil, ErrNoTokenizedAccount if `accAddress` has not been tokenized (no record found)
func (k Keeper) GetTokenizedAccountEntry(ctx sdk.Context, accAddress sdk.AccAddress) (*common.Address, error) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetTokenizedAccountKey(accAddress)

	contractAddressBz := store.Get(key)
	if len(contractAddressBz) == 0 {
		return nil, types.ErrNoTokenizedAccount
	}

	contractAddress := common.BytesToAddress(contractAddressBz)
	return &contractAddress, nil
}

// GetTokenizedAccount fetches info about a TokenizedAccount
// returns nil, ErrNoTokenizedAccount if `accAddress` has not been tokenized (no record found)
func (k Keeper) GetTokenizedAccount(ctx sdk.Context, accAddress sdk.AccAddress) (*types.TokenizedAccount, error) {
	contractAddress, err := k.GetTokenizedAccountEntry(ctx, accAddress)
	if err != nil {
		return nil, err
	}

	owner, err := k.queryTokenizedAccountOwner(ctx, *contractAddress)
	if err != nil {
		return nil, err
	}

	return &types.TokenizedAccount{
		Owner:            EVMToSDKAddress(*owner).String(),
		TokenizedAccount: accAddress.String(),
		NftAddress:       contractAddress.Hex(),
	}, nil
}

// IsTokenizedAccount checks if the input account is a Tokenized Account
func (k Keeper) IsTokenizedAccount(ctx sdk.Context, account sdk.AccAddress) bool {
	isTokenized, _ := k.IsTokenizedAccountWithValue(ctx, account)
	return isTokenized
}

// IsTokenizedAccountWithValue checks if the input account is a Tokenized Account and returns the account's nft contract
func (k Keeper) IsTokenizedAccountWithValue(ctx sdk.Context, account sdk.AccAddress) (bool, *common.Address) {
	taEntry, err := k.GetTokenizedAccountEntry(ctx, account)
	k.Logger(ctx).Debug("IsTokenizedAccount", "taEntry", taEntry)
	return err == nil && taEntry != nil, taEntry
}

// GetTokenizedAccountByNFTAddress fetches info about a TokenizedAccount given the address of the TokenizedAccountNFT in the EVM
// returns nil, ErrNoTokenizedAccount if `nftAddress` has not been tokenized (no record found)
func (k Keeper) GetTokenizedAccountByNFTAddress(ctx sdk.Context, nftAddress common.Address) (*types.TokenizedAccount, error) {
	var tokenizedAccount *types.TokenizedAccount = nil
	var err error = nil

	k.IterateTokenizedAccounts(
		ctx,
		func(_ []byte, accAddress sdk.AccAddress, owner common.Address, nftAddr common.Address) (stop bool) {
			if nftAddr == nftAddress {
				tokenizedAccount = &types.TokenizedAccount{
					Owner:            EVMToSDKAddress(owner).String(),
					TokenizedAccount: accAddress.String(),
					NftAddress:       nftAddr.Hex(),
				}
				return true
			}
			return false
		},
	)

	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to find tokenized account by NFT")
	}

	return tokenizedAccount, nil
}

// GetTokenizedAccountsByCosmosOwner fetches info about a TokenizedAccount given the bech32 address of the TokenizedAccountNFT holder
// returns nil, ErrNoTokenizedAccount if `nftAddress` has not been tokenized (no record found)
func (k Keeper) GetTokenizedAccountsByCosmosOwner(ctx sdk.Context, ownerAddress sdk.AccAddress) ([]*types.TokenizedAccount, error) {
	owner := SDKToEVMAddress(ownerAddress)
	return k.GetTokenizedAccountsByEVMOwner(ctx, owner)
}

// GetTokenizedAccountsByEVMOwner fetches info about a TokenizedAccount given the EVM address of the TokenizedAccountNFT holder
// returns nil, ErrNoTokenizedAccount if `nftAddress` has not been tokenized (no record found)
func (k Keeper) GetTokenizedAccountsByEVMOwner(ctx sdk.Context, ownerAddress common.Address) ([]*types.TokenizedAccount, error) {
	var tokenizedAccounts []*types.TokenizedAccount

	k.IterateTokenizedAccounts(
		ctx,
		func(_ []byte, accAddress sdk.AccAddress, owner common.Address, nftAddr common.Address) (stop bool) {
			if owner == ownerAddress {
				tokenizedAccounts = append(tokenizedAccounts, &types.TokenizedAccount{
					Owner:            EVMToSDKAddress(owner).String(),
					TokenizedAccount: accAddress.String(),
					NftAddress:       nftAddr.Hex(),
				})
			}
			return true
		},
	)

	return tokenizedAccounts, nil
}

// GetTokenizedAccountByNFTAddress fetches info about a TokenizedAccount given the address of the TokenizedAccountNFT in the EVM
// returns nil, ErrNoTokenizedAccount if `nftAddress` has not been tokenized (no record found)
func (k Keeper) CollectTokenizedAccounts(ctx sdk.Context) ([]*types.TokenizedAccount, error) {
	var tokenizedAccounts []*types.TokenizedAccount
	var err error = nil

	k.IterateTokenizedAccounts(
		ctx,
		func(_ []byte, accAddress sdk.AccAddress, owner common.Address, nftAddr common.Address) (stop bool) {
			tokenizedAccounts = append(tokenizedAccounts, &types.TokenizedAccount{
				Owner:            EVMToSDKAddress(owner).String(),
				TokenizedAccount: accAddress.String(),
				NftAddress:       nftAddr.Hex(),
			})
			return false
		},
	)

	if err != nil {
		return nil, sdkerrors.Wrap(err, "unable to collect tokenized accounts")
	}

	return tokenizedAccounts, nil
}

// IterateTokenizedAccounts calls the provided callback `cb` on every discovered TokenizedAccount entry. Return stop=true to end iteration early.
func (k Keeper) IterateTokenizedAccounts(ctx sdk.Context, cb func(key []byte, accAddress sdk.AccAddress, owner common.Address, nftAddress common.Address) (stop bool)) {
	pStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.TokenizedAccountsKey)
	iterator := pStore.Iterator(nil, nil) // Iterate through all entries
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		accAddressBz := key[len(types.TokenizedAccountsKey):]
		accAddress := sdk.AccAddress(accAddressBz)
		nftAddressBz := iterator.Value()
		nftAddress := common.BytesToAddress(nftAddressBz)
		owner, err := k.queryTokenizedAccountOwner(ctx, nftAddress)
		if err != nil {
			break
		}

		if cb(key, accAddress, *owner, nftAddress) {
			break
		}
	}
}

// RedirectTokenizedAccountExcessBalances will check if this account is a TokenizedAccount,
// then may funnel any excess balances to the registered TokenizedAccountNFT depending on the set thresholds
// If no threshold is set for any coin in `changedAmounts`, its balance WILL NOT be sent to the NFT
func (k Keeper) RedirectTokenizedAccountExcessBalances(ctx sdk.Context, account sdk.AccAddress, changedErc20s []*common.Address) error {
	logger := k.Logger(ctx)
	logger.Debug("Enter RedirectTokenizedAccountExcessBalances", "receiver", account.String(), "changedBalances", changedErc20s)
	isTokenized, nft := k.IsTokenizedAccountWithValue(ctx, account)
	if !isTokenized {
		logger.Debug("receiver not tokenized")
		return nil // Do nothing for non-Tokenized Accounts
	}
	// If the account IS tokenized, it MUST have its NFT registered
	if nft == nil {
		panic("discovered nil nft address in tokenized account entry")
	}
	logger.Debug("Discovered TokenizedAccount->NFT entry", "account", account.String(), "nft", nft.Hex())

	thresholds, err := k.queryTokenizedAccountThresholds(ctx, *nft)
	if err != nil {
		panic(fmt.Errorf("failed to query evm for tokenized account thresholds: %s", err.Error()))
	}

	changedThresholdedErc20s := types.FindThresholdIntersection(thresholds, changedErc20s)
	logger.Debug("Found thresholds + modified balances intersection", "thresholds", thresholds, "changedBalances", changedErc20s, "intersection", changedThresholdedErc20s)
	var funneledAmounts []sdk.Coin
	for _, toRedirect := range changedThresholdedErc20s {
		pair, found := k.erc20Keeper.GetTokenPair(ctx, k.erc20Keeper.GetTokenPairID(ctx, toRedirect.Token.Hex()))
		if !found || !pair.Enabled {
			// This should've been checked earlier in msg_server.go
			panic("threshold token pair does not exist or is inactive, should have been caught earlier")
		}
		logger.Debug("Found no pair for threshold token", "token", toRedirect.Token.Hex())
		balance := k.bankKeeper.GetBalance(ctx, account, pair.Denom)
		logger.Debug("Found new balance of token", "balance", balance.String())

		balanceInExcess := balance.Amount.BigInt().Cmp(&toRedirect.Amount) > 0
		logger.Debug("Checking if balance is in excess of threshold", "balance", balance.String(), "threshold", toRedirect.Amount.String(), "exceeded", balanceInExcess)
		if balanceInExcess {
			logger.Debug("Redirecting balance to nft", "account", account.String(), "nft", nft.Hex(), "exceeded", balanceInExcess)
			funneled, err := k.RedirectBalanceToToken(ctx, account, *nft, balance, toRedirect.Amount)
			if err != nil {
				return err
			}
			logger.Debug("Redirected to nft", "amount", funneled.String())
			funneledAmounts = append(funneledAmounts, *funneled)
		}
	}

	logger.Debug("Emitting balance redirect event to log")
	// Emit an event for the block's event log
	ctx.EventManager().EmitEvent(
		types.NewEventBalanceRedirect(account.String(), sdk.Coins(funneledAmounts)),
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
// are deployed by the module account.
func (k Keeper) DeployContract(
	ctx sdk.Context,
	contract evmtypes.CompiledContract,
	args ...interface{},
) (common.Address, error) {
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
	nonce, err := k.accountKeeper.GetSequence(ctx, types.ModuleAddress.Bytes())
	if err != nil {
		return common.Address{},
			sdkerrors.Wrapf(types.ErrContractDeployment,
				"EVM::DeployContract error retrieving nonce: %s", err.Error())
	}

	amount := big.NewInt(0)
	_, err = k.CallEVM(ctx, types.ModuleEVMAddress, nil, amount, data, true)
	if err != nil {
		return common.Address{},
			sdkerrors.Wrapf(types.ErrContractDeployment,
				"EVM::DeployContract error deploying contract: %s", err.Error())
	}

	// Derive the newly created module smart contract using the module address and nonce
	return crypto.CreateAddress(types.ModuleEVMAddress, nonce), nil
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

	// As evmKeeper.ApplyMessage does not directly increment the gas meter, any transaction
	// completed through the CSR module account will technically be 'free'. As such, we can
	// set the gas limit to some arbitrarily high enough number such that every transaction
	// from the module account will always go through.
	// see: https://github.com/evmos/ethermint/blob/35850e620d2825327a175f46ec3e8c60af84208d/x/evm/keeper/state_transition.go#L466
	gasLimit := DefaultGasLimit

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
