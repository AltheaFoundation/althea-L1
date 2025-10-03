package keeper_test

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"

	"github.com/evmos/ethermint/x/evm/statedb"
	evmtypes "github.com/evmos/ethermint/x/evm/types"

	"github.com/AltheaFoundation/althea-L1/testutil"
	"github.com/AltheaFoundation/althea-L1/x/erc20/keeper"
	"github.com/AltheaFoundation/althea-L1/x/erc20/types"
)

// nolint: dupl
func (suite *KeeperTestSuite) TestConvertCoinNativeCoin() {
	testCases := []struct {
		name           string
		mint           int64
		burn           int64
		malleate       func(common.Address)
		extra          func()
		expPass        bool
		selfdestructed bool
	}{
		{
			"ok - sufficient funds",
			100,
			10,
			func(common.Address) {},
			func() {},
			true,
			false,
		},
		{
			"ok - equal funds",
			10,
			10,
			func(common.Address) {},
			func() {},
			true,
			false,
		},
		{
			"ok - suicided contract",
			10,
			10,
			func(erc20 common.Address) {
				stateDb := suite.StateDB()
				ok := stateDb.Suicide(erc20)
				suite.Require().True(ok)
				suite.Require().NoError(stateDb.Commit())
			},
			func() {},
			true,
			true,
		},
		{
			"fail - insufficient funds",
			0,
			10,
			func(common.Address) {},
			func() {},
			false,
			false,
		},
		{
			"fail - minting disabled",
			100,
			10,
			func(common.Address) {
				params := types.DefaultParams()
				params.EnableErc20 = false
				suite.app.Erc20Keeper.SetParams(suite.ctx, params)
			},
			func() {},
			false,
			false,
		},
		{
			"fail - deleted module account - force fail", 100, 10, func(common.Address) {},
			func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			}, false, false,
		},
		{
			"fail - force evm fail", 100, 10, func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, subspaceFound := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(subspaceFound)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			}, false, false,
		},
		{
			"fail - force evm balance error", 100, 10, func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				// first balance of
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// convert coin
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil).Once()
				// second balance of
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, fmt.Errorf("third")).Once()
				// Extra call on test
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			}, false, false,
		},
		{
			"fail - force balance error", 100, 10, func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Times(4)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			}, false, false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			var err error
			suite.mintFeeCollector = true
			suite.SetupTest()
			metadata, pair := suite.setupRegisterCoin()
			suite.Require().NotNil(metadata)
			erc20 := pair.GetERC20Contract()
			tc.malleate(erc20)
			suite.Commit()

			ctx := sdk.WrapSDKContext(suite.ctx)
			coins := sdk.NewCoins(sdk.NewCoin(cosmosTokenBase, sdk.NewInt(tc.mint)))
			sender := sdk.AccAddress(suite.address.Bytes())
			msg := types.NewMsgConvertCoin(
				sdk.NewCoin(cosmosTokenBase, sdk.NewInt(tc.burn)),
				suite.address,
				sender,
			)

			err = suite.app.BankKeeper.MintCoins(suite.ctx, types.ModuleName, coins)
			suite.Require().NoError(err)
			err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sender, coins)
			suite.Require().NoError(err)

			tc.extra()
			res, err := suite.app.Erc20Keeper.ConvertCoin(ctx, msg)
			expRes := &types.MsgConvertCoinResponse{}
			suite.Commit()
			balance := suite.BalanceOf(common.HexToAddress(pair.Erc20Address), suite.address)
			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, metadata.Base)

			if tc.expPass {
				suite.Require().NoError(err, tc.name)

				acc := suite.app.EvmKeeper.GetAccountWithoutBalance(suite.ctx, erc20)
				if tc.selfdestructed {
					suite.Require().Nil(acc, "expected contract to be destroyed")
				} else {
					suite.Require().NotNil(acc)
				}

				if tc.selfdestructed || !acc.IsContract() {
					id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, erc20.String())
					_, found := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
					suite.Require().False(found)
				} else {
					suite.Require().Equal(expRes, res)
					suite.Require().Equal(cosmosBalance.Amount.Int64(), sdk.NewInt(tc.mint-tc.burn).Int64())
					suite.Require().Equal(balance.(*big.Int).Int64(), big.NewInt(tc.burn).Int64())
				}
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
	suite.mintFeeCollector = false
}

// nolint: dupl
func (suite *KeeperTestSuite) TestConvertERC20NativeCoin() {
	testCases := []struct {
		name      string
		mint      int64
		burn      int64
		reconvert int64
		malleate  func()
		expPass   bool
	}{
		{"ok - sufficient funds", 100, 10, 5, func() {}, true},
		{"ok - equal funds", 10, 10, 10, func() {}, true},
		{"fail - insufficient funds", 10, 1, 5, func() {}, false},
		{"fail ", 10, 1, -5, func() {}, false},
		{
			"fail - deleted module account - force fail", 100, 10, 5,
			func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			false,
		},
		{
			"fail - force evm fail", 100, 10, 5,
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			false,
		},
		{
			"fail - force fail second balance", 100, 10, 5,
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				// first balance of
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// convert coin
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil).Once()
				// second balance of
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, fmt.Errorf("third")).Once()
				// Extra call on test
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			false,
		},
		{
			"fail - force fail second balance", 100, 10, 5,
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				// first balance of
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// convert coin
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil).Once()
				// second balance of
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// Extra call on test
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			false,
		},
		{
			"fail - force fail unescrow", 100, 10, 5,
			func() {
				//nolint: exhaustruct
				mockBankKeeper := &MockBankKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, mockBankKeeper, suite.app.EvmKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("failed to unescrow"))
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: "coin", Amount: sdk.OneInt()})
			},
			false,
		},
		{
			"fail - force fail balance after transfer", 100, 10, 5,
			func() {
				//nolint: exhaustruct
				mockBankKeeper := &MockBankKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, mockBankKeeper, suite.app.EvmKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: "acoin", Amount: sdk.OneInt()})
			},
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			var err error
			suite.mintFeeCollector = true
			suite.SetupTest()
			metadata, pair := suite.setupRegisterCoin()
			suite.Require().NotNil(metadata)
			suite.Require().NotNil(pair)

			// Precondition: Convert Coin to ERC20
			coins := sdk.NewCoins(sdk.NewCoin(cosmosTokenBase, sdk.NewInt(tc.mint)))
			sender := sdk.AccAddress(suite.address.Bytes())
			err = suite.app.BankKeeper.MintCoins(suite.ctx, types.ModuleName, coins)
			suite.Require().NoError(err)
			err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sender, coins)
			suite.Require().NoError(err)
			msg := types.NewMsgConvertCoin(
				sdk.NewCoin(cosmosTokenBase, sdk.NewInt(tc.burn)),
				suite.address,
				sender,
			)

			ctx := sdk.WrapSDKContext(suite.ctx)
			_, err = suite.app.Erc20Keeper.ConvertCoin(ctx, msg)
			suite.Require().NoError(err, tc.name)
			suite.Commit()
			balance := suite.BalanceOf(common.HexToAddress(pair.Erc20Address), suite.address)
			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, metadata.Base)
			suite.Require().Equal(cosmosBalance.Amount.Int64(), sdk.NewInt(tc.mint-tc.burn).Int64())
			suite.Require().Equal(balance, big.NewInt(tc.burn))

			// Convert ERC20s back to Coins
			ctx = sdk.WrapSDKContext(suite.ctx)
			contractAddr := common.HexToAddress(pair.Erc20Address)
			msgConvertERC20 := types.NewMsgConvertERC20(
				sdk.NewInt(tc.reconvert),
				sender,
				contractAddr,
				suite.address,
			)

			tc.malleate()
			res, err := suite.app.Erc20Keeper.ConvertERC20(ctx, msgConvertERC20)
			expRes := &types.MsgConvertERC20Response{}
			suite.Commit()
			balance = suite.BalanceOf(contractAddr, suite.address)
			cosmosBalance = suite.app.BankKeeper.GetBalance(suite.ctx, sender, pair.Denom)
			if tc.expPass {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(expRes, res)
				suite.Require().Equal(cosmosBalance.Amount.Int64(), sdk.NewInt(tc.mint-tc.burn+tc.reconvert).Int64())
				suite.Require().Equal(balance.(*big.Int).Int64(), big.NewInt(tc.burn-tc.reconvert).Int64())
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
	suite.mintFeeCollector = false
}

// nolint: dupl
func (suite *KeeperTestSuite) TestConvertERC20NativeERC20() {
	var contractAddr common.Address
	var coinName string

	testCases := []struct {
		name           string
		mint           int64
		transfer       int64
		malleate       func(common.Address)
		extra          func()
		contractType   int
		expPass        bool
		selfdestructed bool
	}{
		{
			"ok - sufficient funds",
			100,
			10,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			true,
			false,
		},
		{
			"ok - equal funds",
			10,
			10,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			true,
			false,
		},
		{
			"ok - equal funds",
			10,
			10,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			true,
			false,
		},
		{
			"ok - suicided contract",
			10,
			10,
			func(erc20 common.Address) {
				stateDb := suite.StateDB()
				ok := stateDb.Suicide(erc20)
				suite.Require().True(ok)
				suite.Require().NoError(stateDb.Commit())
			},
			func() {},
			contractMinterBurner,
			true,
			true,
		},
		{
			"fail - insufficient funds - callEVM",
			0,
			10,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - minting disabled",
			100,
			10,
			func(common.Address) {
				params := types.DefaultParams()
				params.EnableErc20 = false
				suite.app.Erc20Keeper.SetParams(suite.ctx, params)
			},
			func() {},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - direct balance manipulation contract",
			100,
			10,
			func(common.Address) {},
			func() {},
			contractDirectBalanceManipulation,
			false,
			false,
		},
		{
			"fail - delayed malicious contract",
			10,
			10,
			func(common.Address) {},
			func() {},
			contractMaliciousDelayed,
			false,
			false,
		},
		{
			"fail - negative transfer contract",
			10,
			-10,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - no module address",
			100,
			10,
			func(common.Address) {
			},
			func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - force evm fail",
			100,
			10,
			func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - force get balance fail",
			100,
			10,
			func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				balance[31] = uint8(1)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Twice()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced balance error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - force transfer unpack fail",
			100,
			10,
			func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractMinterBurner,
			false,
			false,
		},

		{
			"fail - force invalid transfer fail",
			100,
			10,
			func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - force mint fail",
			100,
			10,
			func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockBankKeeper := &MockBankKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, mockBankKeeper, suite.app.EvmKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				mockBankKeeper.On("MintCoins", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("failed to mint"))
				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("failed to unescrow"))
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: "coin", Amount: sdk.OneInt()})
			},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - force send minted fail",
			100,
			10,
			func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockBankKeeper := &MockBankKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, mockBankKeeper, suite.app.EvmKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				mockBankKeeper.On("MintCoins", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("failed to unescrow"))
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: "coin", Amount: sdk.OneInt()})
			},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - force bank balance fail",
			100,
			10,
			func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockBankKeeper := &MockBankKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, mockBankKeeper, suite.app.EvmKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				mockBankKeeper.On("MintCoins", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: coinName, Amount: sdk.NewInt(int64(10))})
			},
			contractMinterBurner,
			false,
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.mintFeeCollector = true
			suite.SetupTest()

			contractAddr = suite.setupRegisterERC20Pair(tc.contractType)

			tc.malleate(contractAddr)
			suite.Require().NotNil(contractAddr)
			suite.Commit()

			coinName = types.CreateDenom(contractAddr.String())
			sender := sdk.AccAddress(suite.address.Bytes())
			msg := types.NewMsgConvertERC20(
				sdk.NewInt(tc.transfer),
				sender,
				contractAddr,
				suite.address,
			)

			suite.MintERC20Token(contractAddr, suite.address, suite.address, big.NewInt(tc.mint))
			suite.Commit()
			ctx := sdk.WrapSDKContext(suite.ctx)

			tc.extra()
			res, err := suite.app.Erc20Keeper.ConvertERC20(ctx, msg)

			expRes := &types.MsgConvertERC20Response{}
			suite.Commit()
			balance := suite.BalanceOf(contractAddr, suite.address)
			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, coinName)
			if tc.expPass {
				suite.Require().NoError(err, tc.name)

				acc := suite.app.EvmKeeper.GetAccountWithoutBalance(suite.ctx, contractAddr)
				if tc.selfdestructed {
					suite.Require().Nil(acc, "expected contract to be destroyed")
				} else {
					suite.Require().NotNil(acc)
				}

				if tc.selfdestructed || !acc.IsContract() {
					id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, contractAddr.String())
					_, found := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
					suite.Require().False(found)
				} else {
					suite.Require().Equal(expRes, res)
					suite.Require().Equal(cosmosBalance.Amount, sdk.NewInt(tc.transfer))
					suite.Require().Equal(balance.(*big.Int).Int64(), big.NewInt(tc.mint-tc.transfer).Int64())
				}
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
	suite.mintFeeCollector = false
}

// nolint: dupl
func (suite *KeeperTestSuite) TestConvertCoinNativeERC20() {
	var contractAddr common.Address

	testCases := []struct {
		name         string
		mint         int64
		convert      int64
		malleate     func(common.Address)
		extra        func()
		contractType int
		expPass      bool
	}{
		{
			"ok - sufficient funds",
			100,
			10,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			true,
		},
		{
			"ok - equal funds",
			100,
			100,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			true,
		},
		{
			"fail - insufficient funds",
			100,
			200,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			false,
		},
		{
			"fail - direct balance manipulation contract",
			100,
			10,
			func(common.Address) {},
			func() {},
			contractDirectBalanceManipulation,
			false,
		},
		{
			"fail - malicious delayed contract",
			100,
			10,
			func(common.Address) {},
			func() {},
			contractMaliciousDelayed,
			false,
		},
		{
			"fail - deleted module address - force fail",
			100,
			10,
			func(common.Address) {},
			func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			contractMinterBurner,
			false,
		},
		{
			"fail - force evm fail",
			100,
			10,
			func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractMinterBurner,
			false,
		},
		{
			"fail - force invalid transfer",
			100,
			10,
			func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractMinterBurner,
			false,
		},
		{
			"fail - force fail second balance",
			100,
			10,
			func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				balance[31] = uint8(1)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Twice()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("fail second balance"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractMinterBurner,
			false,
		},
		{
			"fail - force fail transfer",
			100,
			10,
			func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractMinterBurner,
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.mintFeeCollector = true
			suite.SetupTest()
			contractAddr = suite.setupRegisterERC20Pair(tc.contractType)
			suite.Require().NotNil(contractAddr)

			id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, contractAddr.String())
			pair, _ := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
			coins := sdk.NewCoins(sdk.NewCoin(pair.Denom, sdk.NewInt(tc.mint)))
			coinName := types.CreateDenom(contractAddr.String())
			sender := sdk.AccAddress(suite.address.Bytes())

			// Precondition: Mint Coins to convert on sender account
			err := suite.app.BankKeeper.MintCoins(suite.ctx, types.ModuleName, coins)
			suite.Require().NoError(err)
			err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sender, coins)
			suite.Require().NoError(err)

			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, coinName)
			suite.Require().Equal(sdk.NewInt(tc.mint), cosmosBalance.Amount)

			// Precondition: Mint escrow tokens on module account
			suite.GrantERC20Token(contractAddr, suite.address, types.ModuleAddress, "MINTER_ROLE")
			suite.MintERC20Token(contractAddr, types.ModuleAddress, types.ModuleAddress, big.NewInt(tc.mint))
			tokenBalance := suite.BalanceOf(contractAddr, types.ModuleAddress)
			suite.Require().Equal(big.NewInt(tc.mint), tokenBalance)

			tc.malleate(contractAddr)
			suite.Commit()

			// Convert Coins back to ERC20s
			receiver := suite.address
			ctx := sdk.WrapSDKContext(suite.ctx)
			msg := types.NewMsgConvertCoin(
				sdk.NewCoin(coinName, sdk.NewInt(tc.convert)),
				receiver,
				sender,
			)

			tc.extra()
			res, err := suite.app.Erc20Keeper.ConvertCoin(ctx, msg)

			expRes := &types.MsgConvertCoinResponse{}
			suite.Commit()
			tokenBalance = suite.BalanceOf(contractAddr, suite.address)
			cosmosBalance = suite.app.BankKeeper.GetBalance(suite.ctx, sender, coinName)
			if tc.expPass {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(expRes, res)
				suite.Require().Equal(sdk.NewInt(tc.mint-tc.convert), cosmosBalance.Amount)
				suite.Require().Equal(big.NewInt(tc.convert), tokenBalance.(*big.Int))
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
	suite.mintFeeCollector = false
}

// nolint: dupl
func (suite *KeeperTestSuite) TestWrongPairOwnerERC20NativeCoin() {
	testCases := []struct {
		name      string
		mint      int64
		burn      int64
		reconvert int64
		expPass   bool
	}{
		{"ok - sufficient funds", 100, 10, 5, true},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.mintFeeCollector = true
			suite.SetupTest()
			metadata, pair := suite.setupRegisterCoin()
			suite.Require().NotNil(metadata)
			suite.Require().NotNil(pair)

			// Precondition: Convert Coin to ERC20
			coins := sdk.NewCoins(sdk.NewCoin(cosmosTokenBase, sdk.NewInt(tc.mint)))
			sender := sdk.AccAddress(suite.address.Bytes())
			var err error
			err = suite.app.BankKeeper.MintCoins(suite.ctx, types.ModuleName, coins)
			suite.Require().NoError(err)
			err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sender, coins)
			suite.Require().NoError(err)
			msg := types.NewMsgConvertCoin(
				sdk.NewCoin(cosmosTokenBase, sdk.NewInt(tc.burn)),
				suite.address,
				sender,
			)

			pair.ContractOwner = types.OWNER_UNSPECIFIED
			suite.app.Erc20Keeper.SetTokenPair(suite.ctx, *pair)

			ctx := sdk.WrapSDKContext(suite.ctx)
			_, err = suite.app.Erc20Keeper.ConvertCoin(ctx, msg)
			suite.Require().Error(err, tc.name)

			// Convert ERC20s back to Coins
			ctx = sdk.WrapSDKContext(suite.ctx)
			contractAddr := common.HexToAddress(pair.Erc20Address)
			msgConvertERC20 := types.NewMsgConvertERC20(
				sdk.NewInt(tc.reconvert),
				sender,
				contractAddr,
				suite.address,
			)

			_, err = suite.app.Erc20Keeper.ConvertERC20(ctx, msgConvertERC20)
			suite.Require().Error(err, tc.name)
		})
	}
}

// nolint: dupl
func (suite *KeeperTestSuite) TestConvertCoinNativeIBCVoucher() {
	testCases := []struct {
		name           string
		mint           int64
		burn           int64
		malleate       func(common.Address)
		extra          func()
		expPass        bool
		selfdestructed bool
	}{
		{
			"ok - sufficient funds",
			100,
			10,
			func(common.Address) {},
			func() {},
			true,
			false,
		},
		{
			"ok - equal funds",
			10,
			10,
			func(common.Address) {},
			func() {},
			true,
			false,
		},
		{
			"ok - suicided contract",
			10,
			10,
			func(erc20 common.Address) {
				stateDb := suite.StateDB()
				ok := stateDb.Suicide(erc20)
				suite.Require().True(ok)
				suite.Require().NoError(stateDb.Commit())
			},
			func() {},
			true,
			true,
		},
		{
			"fail - insufficient funds",
			0,
			10,
			func(common.Address) {},
			func() {},
			false,
			false,
		},
		{
			"fail - minting disabled",
			100,
			10,
			func(common.Address) {
				params := types.DefaultParams()
				params.EnableErc20 = false
				suite.app.Erc20Keeper.SetParams(suite.ctx, params)
			},
			func() {},
			false,
			false,
		},
		{
			"fail - deleted module account - force fail", 100, 10, func(common.Address) {},
			func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			}, false, false,
		},
		{
			"fail - force evm fail", 100, 10, func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			}, false, false,
		},
		{
			"fail - force evm balance error", 100, 10, func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				// first balance of
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// convert coin
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil).Once()
				// second balance of
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, fmt.Errorf("third")).Once()
				// Extra call on test
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			}, false, false,
		},
		{
			"fail - force balance error", 100, 10, func(common.Address) {},
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Times(4)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			}, false, false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.mintFeeCollector = true
			suite.SetupTest()
			metadata, pair := suite.setupRegisterIBCVoucher()
			suite.Require().NotNil(metadata)
			erc20 := pair.GetERC20Contract()
			tc.malleate(erc20)
			suite.Commit()

			ctx := sdk.WrapSDKContext(suite.ctx)
			coins := sdk.NewCoins(sdk.NewCoin(ibcBase, sdk.NewInt(tc.mint)))
			sender := sdk.AccAddress(suite.address.Bytes())
			msg := types.NewMsgConvertCoin(
				sdk.NewCoin(ibcBase, sdk.NewInt(tc.burn)),
				suite.address,
				sender,
			)

			var err error
			err = suite.app.BankKeeper.MintCoins(suite.ctx, types.ModuleName, coins)
			suite.Require().NoError(err)
			err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sender, coins)
			suite.Require().NoError(err)

			tc.extra()
			res, err := suite.app.Erc20Keeper.ConvertCoin(ctx, msg)
			expRes := &types.MsgConvertCoinResponse{}
			suite.Commit()
			balance := suite.BalanceOf(common.HexToAddress(pair.Erc20Address), suite.address)
			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, metadata.Base)

			if tc.expPass {
				suite.Require().NoError(err, tc.name)

				acc := suite.app.EvmKeeper.GetAccountWithoutBalance(suite.ctx, erc20)
				if tc.selfdestructed {
					suite.Require().Nil(acc, "expected contract to be destroyed")
				} else {
					suite.Require().NotNil(acc)
				}

				if tc.selfdestructed || !acc.IsContract() {
					id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, erc20.String())
					_, found := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
					suite.Require().False(found)
				} else {
					suite.Require().Equal(expRes, res)
					suite.Require().Equal(cosmosBalance.Amount.Int64(), sdk.NewInt(tc.mint-tc.burn).Int64())
					suite.Require().Equal(balance.(*big.Int).Int64(), big.NewInt(tc.burn).Int64())
				}
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
	suite.mintFeeCollector = false
}

// nolint: dupl
func (suite *KeeperTestSuite) TestConvertERC20NativeIBCVoucher() {
	testCases := []struct {
		name      string
		mint      int64
		burn      int64
		reconvert int64
		malleate  func()
		expPass   bool
	}{
		{"ok - sufficient funds", 100, 10, 5, func() {}, true},
		{"ok - equal funds", 10, 10, 10, func() {}, true},
		{"fail - insufficient funds", 10, 1, 5, func() {}, false},
		{"fail ", 10, 1, -5, func() {}, false},
		{
			"fail - deleted module account - force fail", 100, 10, 5,
			func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			false,
		},
		{
			"fail - force evm fail", 100, 10, 5,
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			false,
		},
		{
			"fail - force fail second balance", 100, 10, 5,
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				// first balance of
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// convert coin
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil).Once()
				// second balance of
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, fmt.Errorf("third")).Once()
				// Extra call on test
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			false,
		},
		{
			"fail - force fail second balance", 100, 10, 5,
			func() {
				//nolint: exhaustruct
				mockEVMKeeper := &MockEVMKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, mockEVMKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				//nolint: exhaustruct
				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				// first balance of
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// convert coin
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil).Once()
				// second balance of
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// Extra call on test
				//nolint: exhaustruct
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			false,
		},
		{
			"fail - force fail unescrow", 100, 10, 5,
			func() {
				//nolint: exhaustruct
				mockBankKeeper := &MockBankKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, mockBankKeeper, suite.app.EvmKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("failed to unescrow"))
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: "coin", Amount: sdk.OneInt()})
			},
			false,
		},
		{
			"fail - force fail balance after transfer", 100, 10, 5,
			func() {
				//nolint: exhaustruct
				mockBankKeeper := &MockBankKeeper{}
				sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
				suite.Require().True(found)
				erc20Keeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, mockBankKeeper, suite.app.EvmKeeper, suite.app.GasfreeKeeper)
				erc20Keeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
				suite.app.Erc20Keeper = &erc20Keeper

				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: ibcBase, Amount: sdk.OneInt()})
			},
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.mintFeeCollector = true
			suite.SetupTest()
			metadata, pair := suite.setupRegisterIBCVoucher()
			suite.Require().NotNil(metadata)
			suite.Require().NotNil(pair)

			// Precondition: Convert Coin to ERC20
			coins := sdk.NewCoins(sdk.NewCoin(ibcBase, sdk.NewInt(tc.mint)))
			sender := sdk.AccAddress(suite.address.Bytes())
			var err error
			err = suite.app.BankKeeper.MintCoins(suite.ctx, types.ModuleName, coins)
			suite.Require().NoError(err)
			err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sender, coins)
			suite.Require().NoError(err)
			msg := types.NewMsgConvertCoin(
				sdk.NewCoin(ibcBase, sdk.NewInt(tc.burn)),
				suite.address,
				sender,
			)

			ctx := sdk.WrapSDKContext(suite.ctx)
			_, err = suite.app.Erc20Keeper.ConvertCoin(ctx, msg)
			suite.Require().NoError(err, tc.name)
			suite.Commit()
			balance := suite.BalanceOf(common.HexToAddress(pair.Erc20Address), suite.address)
			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, metadata.Base)
			suite.Require().Equal(cosmosBalance.Amount.Int64(), sdk.NewInt(tc.mint-tc.burn).Int64())
			suite.Require().Equal(balance, big.NewInt(tc.burn))

			// Convert ERC20s back to Coins
			ctx = sdk.WrapSDKContext(suite.ctx)
			contractAddr := common.HexToAddress(pair.Erc20Address)
			msgConvertERC20 := types.NewMsgConvertERC20(
				sdk.NewInt(tc.reconvert),
				sender,
				contractAddr,
				suite.address,
			)

			tc.malleate()
			res, err := suite.app.Erc20Keeper.ConvertERC20(ctx, msgConvertERC20)
			expRes := &types.MsgConvertERC20Response{}
			suite.Commit()
			balance = suite.BalanceOf(contractAddr, suite.address)
			cosmosBalance = suite.app.BankKeeper.GetBalance(suite.ctx, sender, pair.Denom)
			if tc.expPass {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(expRes, res)
				suite.Require().Equal(cosmosBalance.Amount.Int64(), sdk.NewInt(tc.mint-tc.burn+tc.reconvert).Int64())
				suite.Require().Equal(balance.(*big.Int).Int64(), big.NewInt(tc.burn-tc.reconvert).Int64())
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
	suite.mintFeeCollector = false
}

// TestSendCoinToEVM tests MsgSendCoinToEVM which wraps ConvertCoin then deducts a gasfree fee.
// It validates:
// - ERC20 balance minted equals send amount
// - Bank balance after equals initial - sendAmount - fee (when fee > 0)
// - Fee collector module received the fee
// - Escrow module account received the sent amount
// - Event attributes (sender, receiver, cosmos_coin, amount, fees_collected) are correct
// MockGasfreeKeeper implements types.GasfreeKeeper allowing custom basis points per test.
type MockGasfreeKeeper struct {
	bp     uint64
	tokens []string
}

func (m MockGasfreeKeeper) GetGasfreeErc20InteropFeeBasisPoints(ctx sdk.Context) (uint64, error) {
	return m.bp, nil
}
func (m MockGasfreeKeeper) GetGasfreeErc20InteropTokens(ctx sdk.Context) ([]string, error) {
	return m.tokens, nil
}

// helper to compute expected fee (mirrors keeper.getGasfreeFeeForAmount logic)
func calcFee(amount sdkmath.Int, basisPoints uint64) sdkmath.Int {
	return sdk.NewDecFromInt(amount).MulInt64(int64(basisPoints)).QuoInt64(10000).TruncateInt()
}

// helper to make new Ints
func mustIntFromString(s string) sdkmath.Int {
	i, ok := sdkmath.NewIntFromString(s)
	if !ok {
		panic("invalid int string")
	}
	return i
}

func (suite *KeeperTestSuite) mockKeepers(gasfreeKeeper types.GasfreeKeeper, ibcTransferKeeper *testutil.MockTransferKeeper) {
	// Inject mock gasfree keeper with chosen basis points
	sp, found := suite.app.ParamsKeeper.GetSubspace(types.ModuleName)
	suite.Require().True(found)
	newKeeper := keeper.NewKeeper(suite.app.GetKey("erc20"), suite.app.AppCodec(), sp, suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.EvmKeeper, gasfreeKeeper)
	if ibcTransferKeeper != nil {
		newKeeper.SetIBCTransferKeeper(ibcTransferKeeper)
	} else {
		newKeeper.SetIBCTransferKeeper(*suite.app.IbcTransferKeeper)
	}
	suite.app.Erc20Keeper = &newKeeper
}

func (suite *KeeperTestSuite) mintCoin(coin sdk.Coin, receiver sdk.AccAddress, module string) {
	coins := sdk.NewCoins(coin)
	err := suite.app.BankKeeper.MintCoins(suite.ctx, module, coins)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, module, receiver, coins)
	suite.Require().NoError(err)
}

func (suite *KeeperTestSuite) mintCoinAndConvert(coin sdk.Coin, receiver sdk.AccAddress, module string) {
	suite.mintCoin(coin, receiver, module)
	convertMsg := types.NewMsgConvertCoin(coin, suite.address, receiver)
	_, err := suite.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(suite.ctx), convertMsg)
	suite.Require().NoError(err, "Unable to convert coin for test setup")
}

func (suite *KeeperTestSuite) TestSendCoinToEVM() {
	cosmosDenom := cosmosTokenBase

	feeScenarios := []uint64{1, 50, 100, 250, 1234} // 0%, 0.01%, 0.5%, 1%, 2.5%, 12.34%
	badAmts := []sdkmath.Int{
		sdkmath.NewInt(1), sdkmath.NewInt(1000), sdkmath.NewInt(9999),
	}
	sendAmts := []sdkmath.Int{
		sdkmath.NewInt(10000),
		mustIntFromString("1000000"),                     // 1 ATOM
		mustIntFromString("100000000"),                   // 1 BTC
		mustIntFromString("1000000000000000000"),         // 1 ETH
		mustIntFromString("123456789012345678910111213"), // >10,000,000 ETH
	}

	type testCase struct {
		feeBasisPoints uint64
		approvedTokens []string
		sendAmts       []sdkmath.Int
		expError       bool
	}
	testCases := []testCase{
		// Regular testcases
		{feeScenarios[0], []string{cosmosDenom}, sendAmts, false},
		{feeScenarios[1], []string{cosmosDenom}, sendAmts, false},
		{feeScenarios[2], []string{cosmosDenom}, sendAmts, false},
		{feeScenarios[3], []string{cosmosDenom}, sendAmts, false},
		{feeScenarios[4], []string{cosmosDenom, "othercoin"}, sendAmts, false},
		// Unapproved token
		{feeScenarios[1], []string{"othercoin"}, sendAmts, true},
		{feeScenarios[2], []string{}, sendAmts, true},
		// Insufficient amount for fee collection
		{feeScenarios[0], []string{cosmosDenom}, badAmts, true},
		{feeScenarios[1], []string{cosmosDenom}, badAmts, true},
		{feeScenarios[2], []string{cosmosDenom}, badAmts, true},
		{feeScenarios[3], []string{cosmosDenom}, badAmts, true},
		{feeScenarios[4], []string{cosmosDenom, "othercoin"}, badAmts, true},
		// No fee
		{0, []string{cosmosDenom}, sendAmts, false},
		{0, []string{cosmosDenom}, badAmts, false},
	}
	// Run subtests for all combinations of approved tokens, fee basis points, and send amounts
	// Each subtest mints the initial coins to the sender, then calls SendCoinToEVM and checks balances and events
	// The fee collector module is minted some coins to ensure it exists before each test

	for _, tc := range testCases {
		bp := tc.feeBasisPoints
		tokens := tc.approvedTokens
		suite.Run(fmt.Sprintf("fee_bp=%d|tokens=%v", bp, tokens), func() {
			for _, send := range tc.sendAmts {
				suite.mintFeeCollector = true
				suite.SetupTest()

				mockGasfreeKeeper := MockGasfreeKeeper{bp: bp, tokens: tokens}
				suite.mockKeepers(mockGasfreeKeeper, nil)

				// Register coin
				metadata, pair := suite.setupRegisterCoin()
				suite.Require().NotNil(metadata)

				sender := sdk.AccAddress(suite.address.Bytes())
				expectedFee := calcFee(send, bp)
				initial := send.Add(expectedFee)

				// Mint initial coins to sender
				coin := sdk.NewCoin(cosmosDenom, initial)
				suite.mintCoin(coin, sender, types.ModuleName)

				feeCollectorAddr := suite.app.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
				feeCollectorBalPre := suite.app.BankKeeper.GetBalance(suite.ctx, feeCollectorAddr, cosmosDenom)

				msg := types.NewMsgSendCoinToEVM(sdk.NewCoin(cosmosDenom, send), sender)
				ctx := sdk.WrapSDKContext(suite.ctx)
				preEvents := len(suite.ctx.EventManager().Events())
				resp, err2 := suite.app.Erc20Keeper.SendCoinToEVM(ctx, msg)

				// For expected errors, we want the error and to skip remaining checks
				if tc.expError {
					suite.Require().Error(err2)
					return
				}
				// Otherwise, we must check the state changed as expected
				suite.Require().NoError(err2)
				suite.Require().NotNil(resp)

				// Bank balances after
				remaining := suite.app.BankKeeper.GetBalance(suite.ctx, sender, cosmosDenom)
				suite.Require().True(sdkmath.NewInt(initial.Sub(send).Sub(expectedFee).Int64()).Equal(remaining.Amount), "remaining balance mismatch")
				// Module escrow balance equals sent amount
				moduleEscrow := suite.app.BankKeeper.GetBalance(suite.ctx, types.ModuleAddress.Bytes(), cosmosDenom)
				suite.Require().True(send.Equal(moduleEscrow.Amount))
				// Fee collector balance equals expected fee (use .Equal to avoid zero internal representation differences)
				feeCollectorBal := suite.app.BankKeeper.GetBalance(suite.ctx, feeCollectorAddr, cosmosDenom)
				actualFeeCollected := feeCollectorBal.Amount.Sub(feeCollectorBalPre.Amount)
				suite.Require().True(expectedFee.Equal(actualFeeCollected), "fee collector balance mismatch: expected=%s actual=%s", expectedFee.String(), actualFeeCollected.String())

				// ERC20 balance
				erc20Addr := common.HexToAddress(pair.Erc20Address)
				erc20Bal := suite.BalanceOf(erc20Addr, suite.address)
				suite.Require().Equal(send.BigInt(), erc20Bal.(*big.Int))

				// Event assertions
				events := suite.ctx.EventManager().Events()[preEvents:]
				var foundSend bool
				expectedFeeCoin := sdk.NewCoin(cosmosDenom, expectedFee)
				for _, ev := range events {
					if ev.Type == types.EventTypeSendCoinToEVM {
						foundSend = true
						attrMap := map[string]string{}
						for _, a := range ev.Attributes {
							attrMap[string(a.Key)] = string(a.Value)
						}
						suite.Require().Equal(msg.Sender, attrMap[sdk.AttributeKeySender])
						suite.Require().Equal(suite.address.Hex(), attrMap[types.AttributeKeyReceiver])
						suite.Require().Equal(cosmosDenom, attrMap[types.AttributeKeyCosmosCoin])
						suite.Require().Equal(send.String(), attrMap[sdk.AttributeKeyAmount])
						suite.Require().Equal(expectedFeeCoin.String(), attrMap[types.AttributeKeyFeesCollected])
					}
				}
				suite.Require().True(foundSend, "expected send_coin_to_evm event not found")
			}
		})
	}
	suite.mintFeeCollector = false
}

// TestSendERC20ToCosmos tests MsgSendERC20ToCosmos which wraps ConvertERC20 then deducts a gasfree fee.
// Flow setup per amount:
// 1. ConvertCoin (mint ERC20 to user, escrow coins in module) for the send amount.
// 2. Call SendERC20ToCosmos for same amount.
// Assertions:
// - User ERC20 balance goes from sendAmt -> 0
// - Module escrow coin balance goes from sendAmt -> 0
// - User coin balance after = sendAmt - fee
// - Fee collector increased by fee
// - Event emitted with correct attributes
// - When neither denom nor erc20 address approved, handler errors.
func (suite *KeeperTestSuite) TestSendERC20ToCosmos() {
	feeScenarios := []uint64{1, 50, 100, 250, 1234}
	badAmts := []sdkmath.Int{
		sdkmath.NewInt(1), sdkmath.NewInt(5), sdkmath.NewInt(8),
	}
	sendAmts := []sdkmath.Int{
		sdkmath.NewInt(10000),
		mustIntFromString("1000000"),                  // 1 ATOM
		mustIntFromString("100000000"),                // 1 BTC
		mustIntFromString("1000000000000000000"),      // 1 ETH
		mustIntFromString("123456789012345678910111"), // big
	}
	cosmosDenom := cosmosTokenBase

	type testCase struct {
		feeBP    uint64
		approved []string // entries for mock gasfree list
		sendAmts []sdkmath.Int
		expError bool
	}
	testCases := []testCase{
		// Approved via denom
		{feeScenarios[0], []string{cosmosDenom}, sendAmts, false},
		{feeScenarios[1], []string{cosmosDenom}, sendAmts, false},
		{feeScenarios[2], []string{cosmosDenom}, sendAmts, false},
		{feeScenarios[3], []string{cosmosDenom}, sendAmts, false},
		{feeScenarios[4], []string{cosmosDenom}, sendAmts, false},
		{feeScenarios[0], []string{cosmosDenom, "othercoin"}, sendAmts, false},
		{feeScenarios[1], []string{cosmosDenom, "othercoin", "anothercoin"}, sendAmts, false},
		// Unapproved token (by denom or erc20 address)
		{feeScenarios[1], []string{"other"}, sendAmts, true},
		{feeScenarios[2], []string{}, sendAmts, true},
		// Insufficient amount for fee collection
		{feeScenarios[0], []string{cosmosDenom}, badAmts, true},
		{feeScenarios[1], []string{cosmosDenom}, badAmts, true},
		{feeScenarios[2], []string{cosmosDenom}, badAmts, true},
		{feeScenarios[3], []string{cosmosDenom}, badAmts, true},
		{feeScenarios[4], []string{cosmosDenom}, badAmts, true},
		{feeScenarios[0], []string{cosmosDenom, "othercoin"}, badAmts, true},
		{feeScenarios[1], []string{cosmosDenom, "othercoin", "anothercoin"}, badAmts, true},
		// No fee
		{0, []string{cosmosDenom}, sendAmts, false},
		{0, []string{cosmosDenom}, badAmts, false},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("fee_bp=%d|tokens=%v", tc.feeBP, tc.approved), func() {
			for _, sendAmt := range tc.sendAmts {
				suite.mintFeeCollector = true
				suite.SetupTest()

				// Register coin first to obtain pair (denom<->erc20 address)
				metadata, pair := suite.setupRegisterCoin()
				suite.Require().NotNil(metadata)

				erc20Addr := common.HexToAddress(pair.Erc20Address)
				// Build approved token list: if list contains placeholder "$ERC20" replace with actual contract address
				approvedTokens := append([]string{}, tc.approved...)
				// Add variant test where approval is by erc20 address but not denom
				// We accomplish this by rerunning when approved list holds special marker "erc20_only"
				// Instead simpler: if approved list == {cosmosDenom} we still separately test erc20-only below.

				runOne := func(label string, tokens []string, expectError bool) {
					mockGasfreeKeeper := MockGasfreeKeeper{bp: tc.feeBP, tokens: tokens}
					suite.mockKeepers(mockGasfreeKeeper, nil)

					sender := sdk.AccAddress(suite.address.Bytes())
					fee := calcFee(sendAmt, tc.feeBP)
					totalAmt := sendAmt.Add(fee)

					// Step 1: ConvertCoin to get ERC20 tokens (and escrow coins)
					initialCoin := sdk.NewCoin(cosmosDenom, totalAmt)
					suite.mintCoinAndConvert(initialCoin, sender, types.ModuleName)

					// Pre state assertions
					userERC20Bal := suite.BalanceOf(erc20Addr, suite.address).(*big.Int)
					suite.Require().Equal(totalAmt.BigInt().String(), userERC20Bal.String())
					userCoinPre := suite.app.BankKeeper.GetBalance(suite.ctx, sender, cosmosDenom)
					moduleCoinBal := suite.app.BankKeeper.GetBalance(suite.ctx, types.ModuleAddress.Bytes(), cosmosDenom)
					suite.Require().Equal(totalAmt.String(), moduleCoinBal.Amount.String())

					feeCollector := suite.app.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
					feeCollectorPre := suite.app.BankKeeper.GetBalance(suite.ctx, feeCollector, cosmosDenom)

					// Step 2: SendERC20ToCosmos
					msg := types.NewMsgSendERC20ToCosmos(sendAmt, erc20Addr, suite.address)
					preEvents := len(suite.ctx.EventManager().Events())
					resp, err2 := suite.app.Erc20Keeper.SendERC20ToCosmos(sdk.WrapSDKContext(suite.ctx), msg)

					if expectError && err2 == nil {
						fmt.Println("fee bp:", tc.feeBP, "approved:", tokens, "send amt:", sendAmt.String())
					}
					if expectError {
						suite.Require().Error(err2, label)
						return
					}
					suite.Require().NoError(err2, label)
					suite.Require().NotNil(resp)

					// Post state assertions
					userERC20BalAfter := suite.BalanceOf(erc20Addr, suite.address).(*big.Int)
					suite.Require().Equal("0", userERC20BalAfter.String())
					userCoinBalAfter := suite.app.BankKeeper.GetBalance(suite.ctx, sender, cosmosDenom)
					expectedUserCoin := sendAmt
					userCoinDelta := userCoinBalAfter.Amount.Sub(userCoinPre.Amount)
					suite.Require().Equal(expectedUserCoin.String(), userCoinDelta.String())
					moduleCoinBalAfter := suite.app.BankKeeper.GetBalance(suite.ctx, types.ModuleAddress.Bytes(), cosmosDenom)
					suite.Require().True(moduleCoinBalAfter.Amount.IsZero())
					feeCollectorAfter := suite.app.BankKeeper.GetBalance(suite.ctx, feeCollector, cosmosDenom)
					feeDelta := feeCollectorAfter.Amount.Sub(feeCollectorPre.Amount)
					suite.Require().True(fee.Equal(feeDelta), "fee collector mismatch expected=%s actual=%s", fee.String(), feeDelta.String())

					// Event assertions
					events := suite.ctx.EventManager().Events()[preEvents:]
					var foundEvent bool
					expectedFeeCoin := sdk.NewCoin(cosmosDenom, fee)
					for _, ev := range events {
						if ev.Type == types.EventTypeSendERC20ToCosmos {
							foundEvent = true
							attrMap := map[string]string{}
							for _, a := range ev.Attributes {
								attrMap[string(a.Key)] = string(a.Value)
							}
							suite.Require().Equal(msg.Sender, attrMap[sdk.AttributeKeySender])
							suite.Require().Equal(erc20Addr.Hex(), attrMap[types.AttributeKeyERC20Token])
							suite.Require().Equal(sendAmt.String(), attrMap[sdk.AttributeKeyAmount])
							suite.Require().Equal(expectedFeeCoin.String(), attrMap[types.AttributeKeyFeesCollected])
						}
					}
					suite.Require().True(foundEvent, "expected send_erc20_to_cosmos event not found")
				}

				// Run with current approvedTokens as-is
				runOne("denom_or_given", approvedTokens, tc.expError)
				// Additional case: erc20-only approval (skip if already error or erc20 address already present or denom absent making same)
				if !tc.expError { // only meaningful if original expected success
					runOne("erc20_only", []string{pair.Erc20Address}, false)
				}
			}
		})
	}
	suite.mintFeeCollector = false
}

// TestSendERC20ToCosmosAndIBCTransfer mirrors TestSendERC20ToCosmos but exercises the IBC transfer path.
// Additional assertions:
// - Tokens (minus fee) are escrowed in the IBC transfer module account (module holds full send amount; fee deducted from separately pre-minted balance).
// - SendERC20IBCTransfer event emitted with correct attributes.
// Strategy for fee deduction ordering issue: we pre-mint expected fee amount of coins to the user BEFORE calling the message so that after
// ConvertERC20 (which credits sendAmt coins) and the IBC escrow (which moves sendAmt coins), the user still has the fee amount available for deduction.
func (suite *KeeperTestSuite) TestSendERC20ToCosmosAndIBCTransfer() {
	feeScenarios := []uint64{1, 50, 100, 250, 1234}
	badAmts := []sdkmath.Int{
		sdkmath.NewInt(1), sdkmath.NewInt(5), sdkmath.NewInt(8),
	}
	sendAmts := []sdkmath.Int{
		sdkmath.NewInt(10000),
		mustIntFromString("1000000"),                  // 1 ATOM (6dp example)
		mustIntFromString("100000000"),                // 1 BTC (8dp example)
		mustIntFromString("1000000000000000000"),      // 1 ETH (18dp)
		mustIntFromString("123456789012345678910111"), // large
	}
	cosmosDenom := cosmosTokenBase
	destPort := "transfer"
	destChannel := "channel-0"
	destReceiver := sdk.AccAddress(suite.address.Bytes())
	altheaChannel := "channel-0"

	type tc struct {
		feeBP    uint64
		approved []string
		sendAmts []sdkmath.Int
		expError bool
	}
	var testCases []tc
	for _, bp := range feeScenarios {
		testCases = append(testCases, tc{bp, []string{cosmosDenom}, sendAmts, false})
	}
	// Unapproved token
	testCases = append(testCases,
		tc{feeScenarios[1], []string{"other"}, sendAmts, true},
		tc{feeScenarios[2], []string{}, sendAmts, true},
	)
	// Bad amounts
	for _, bp := range feeScenarios {
		testCases = append(testCases, tc{bp, []string{cosmosDenom}, badAmts, true})
	}
	// No fee
	testCases = append(testCases,
		tc{0, []string{cosmosDenom}, sendAmts, false},
		tc{0, []string{cosmosDenom}, badAmts, false},
	)
	transferModuleAddr := suite.app.AccountKeeper.GetModuleAddress(ibctransfertypes.ModuleName)

	path := fmt.Sprintf("%s/%s", ibctransfertypes.PortID, altheaChannel)
	// Set Denom Trace
	denomTrace := ibctransfertypes.DenomTrace{
		Path:      path,
		BaseDenom: cosmosDenom,
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("IBC fee_bp=%d|tokens=%v", tc.feeBP, tc.approved), func() {
			for _, sendAmt := range tc.sendAmts {
				suite.mintFeeCollector = true
				suite.SetupTest()

				// Register coin to obtain pair.
				metadata, pair := suite.setupRegisterCoin()
				suite.Require().NotNil(metadata)
				erc20Addr := common.HexToAddress(pair.Erc20Address)

				runOne := func(label string, tokens []string, expectError bool) {
					mockGasfree := MockGasfreeKeeper{bp: tc.feeBP, tokens: tokens}
					mockTransferKeeper := &testutil.MockTransferKeeper{Keeper: suite.app.BankKeeper, Sequences: make(map[string]uint64)}
					mockTransferKeeper.On("GetDenomTrace", mock.Anything, mock.Anything).Return(denomTrace, true)
					mockTransferKeeper.On("SendTransfer", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
					suite.mockKeepers(mockGasfree, mockTransferKeeper)

					senderAcc := sdk.AccAddress(suite.address.Bytes())
					fee := calcFee(sendAmt, tc.feeBP)
					totalAmt := sendAmt.Add(fee)

					// Step 1: Initial ConvertCoin to produce ERC20 tokens & escrow coins
					initialCoin := sdk.NewCoin(cosmosDenom, totalAmt)
					suite.mintCoinAndConvert(initialCoin, senderAcc, types.ModuleName)

					// Pre-state checks
					erc20Bal := suite.BalanceOf(erc20Addr, suite.address).(*big.Int)
					suite.Require().Equal(totalAmt.BigInt().String(), erc20Bal.String())
					bankBal := suite.app.BankKeeper.GetBalance(suite.ctx, senderAcc, cosmosDenom)
					suite.Require().Equal("0", bankBal.Amount.String())
					moduleEscrow := suite.app.BankKeeper.GetBalance(suite.ctx, types.ModuleAddress.Bytes(), cosmosDenom)
					suite.Require().Equal(totalAmt.String(), moduleEscrow.Amount.String())

					feeCollector := suite.app.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
					feeCollectorPre := suite.app.BankKeeper.GetBalance(suite.ctx, feeCollector, cosmosDenom)
					transferEscrowPre := suite.app.BankKeeper.GetBalance(suite.ctx, transferModuleAddr, cosmosDenom)

					// Step 2: SendERC20ToCosmosAndIBCTransfer
					msg := types.NewMsgSendERC20ToCosmosAndIBCTransfer(sendAmt, erc20Addr, suite.address, destPort, destChannel, destReceiver)
					preEvents := len(suite.ctx.EventManager().Events())
					resp, err2 := suite.app.Erc20Keeper.SendERC20ToCosmosAndIBCTransfer(sdk.WrapSDKContext(suite.ctx), msg)

					if expectError {
						suite.Require().Error(err2, label)
						return
					}
					suite.Require().NoError(err2, label)
					suite.Require().NotNil(resp)

					// Post-state: user ERC20 balance zero
					erc20BalAfter := suite.BalanceOf(erc20Addr, suite.address).(*big.Int)
					suite.Require().Equal("0", erc20BalAfter.String())
					// Module erc20 escrow coins zero
					moduleEscrowAfter := suite.app.BankKeeper.GetBalance(suite.ctx, types.ModuleAddress.Bytes(), cosmosDenom)
					suite.Require().True(moduleEscrowAfter.Amount.IsZero())
					// Transfer module escrow gained sendAmt
					transferEscrowAfter := suite.app.BankKeeper.GetBalance(suite.ctx, transferModuleAddr, cosmosDenom)
					transferDelta := transferEscrowAfter.Amount.Sub(transferEscrowPre.Amount)
					suite.Require().Equal(sendAmt.String(), transferDelta.String())
					// Fee collector increased by fee (if any)
					feeCollectorAfter := suite.app.BankKeeper.GetBalance(suite.ctx, feeCollector, cosmosDenom)
					feeDelta := feeCollectorAfter.Amount.Sub(feeCollectorPre.Amount)
					suite.Require().Equal(fee.String(), feeDelta.String(), "fee collector mismatch")
					// User bank balance zero after fee deduction
					userBankAfter := suite.app.BankKeeper.GetBalance(suite.ctx, senderAcc, cosmosDenom)
					suite.Require().True(userBankAfter.Amount.IsZero())

					// Events
					events := suite.ctx.EventManager().Events()[preEvents:]
					var foundEvent bool
					expectedFeeCoin := sdk.NewCoin(cosmosDenom, fee)
					for _, ev := range events {
						if ev.Type == types.EventTypeSendERC20IBCTransfer {
							attrMap := map[string]string{}
							for _, a := range ev.Attributes {
								attrMap[string(a.Key)] = string(a.Value)
							}
							if attrMap[sdk.AttributeKeySender] == msg.Sender && attrMap[types.AttributeKeyERC20Token] == erc20Addr.Hex() && attrMap[sdk.AttributeKeyAmount] == sendAmt.String() && attrMap[types.AttributeKeyPort] == destPort && attrMap[types.AttributeKeyChannel] == destChannel && attrMap[types.AttributeKeyReceiver] == destReceiver.String() && attrMap[types.AttributeKeyFeesCollected] == expectedFeeCoin.String() {
								foundEvent = true
							}
						}
					}
					suite.Require().True(foundEvent, "expected send_erc20_ibc_transfer event not found")
				}

				// Run with given approved token list
				runOne("denom_or_given", tc.approved, tc.expError)
				// Additional: erc20-only approval (if base case succeeded)
				if !tc.expError {
					runOne("erc20_only", []string{pair.Erc20Address}, false)
				}
			}
		})
	}
	suite.mintFeeCollector = false
}

// TestInsufficientGasfreeFees ensures that when the user has enough balance to cover the transfer amount
// but NOT enough to also pay the gasfree fee, just the SendCoinToEVM and SendERC20ToCosmosAndIBCTransfer
// messages fail with the expected error.
// Note: SendERC20ToCosmos does not have this issue as the user is guaranteed to have enough balance to cover
// the fee since they just converted the full amount to coins
func (suite *KeeperTestSuite) TestInsufficientGasfreeFees() {
	const feeBP uint64 = 250 // 2.5% for clear fee > 0
	cosmosDenom := cosmosTokenBase

	// Helper: set up keeper with mock gasfree tokens including the denom so validation passes and we reach fee deduction
	setupKeeper := func(tokens []string) {
		mockGasfree := MockGasfreeKeeper{bp: feeBP, tokens: tokens}
		suite.mockKeepers(mockGasfree, nil)
	}

	// Case 1: MsgSendCoinToEVM - fund exactly the send amount, fee should be >0 and deduction should fail
	suite.Run("SendCoinToEVM insufficient fee", func() {
		suite.mintFeeCollector = true
		suite.SetupTest()
		// Register coin so pair exists and token is whitelisted
		_, _ = suite.setupRegisterCoin()
		setupKeeper([]string{cosmosDenom})

		sender := sdk.AccAddress(suite.address.Bytes())
		sendAmt := sdk.NewInt(1_000_000)
		coin := sdk.NewCoin(cosmosDenom, sendAmt)
		suite.mintCoin(coin, sender, types.ModuleName) // fund only send amount, no fee margin

		// Build and execute message
		msg := types.NewMsgSendCoinToEVM(coin, sender)
		ctx := sdk.WrapSDKContext(suite.ctx)
		_, err := suite.app.Erc20Keeper.SendCoinToEVM(ctx, msg)
		suite.Require().Error(err, "expected error when fee cannot be collected (Cosmos->ERC20)")
		suite.Require().Contains(fmt.Sprintf("%v", err), "unable to collect gasfree fees")
	})

	// Case 2: MsgSendERC20ToCosmos - need to first convert coins to ERC20 to hold ERC20 balance, but only fund exactly amount (no fee margin)
	suite.Run("SendERC20ToCosmos insufficient fee", func() {
		suite.mintFeeCollector = true
		suite.SetupTest()
		_, pair := suite.setupRegisterCoin()
		erc20Addr := common.HexToAddress(pair.Erc20Address)
		setupKeeper([]string{cosmosDenom, pair.Erc20Address})

		sender := sdk.AccAddress(suite.address.Bytes())
		sendAmt := sdk.NewInt(50_000_000)
		// Mint only send amount of native coin and convert to ERC20 (escrows coins, mints ERC20 to user)
		suite.mintCoinAndConvert(sdk.NewCoin(cosmosDenom, sendAmt), sender, types.ModuleName)
		// Sanity: user holds ERC20 balance but zero bank coin balance (fees must come from coins minted during conversion; none available for fee)
		balERC20 := suite.BalanceOf(erc20Addr, suite.address).(*big.Int)
		suite.Require().Equal(sendAmt.BigInt().String(), balERC20.String())
		bankBal := suite.app.BankKeeper.GetBalance(suite.ctx, sender, cosmosDenom)
		suite.Require().True(bankBal.Amount.IsZero())

		msg := types.NewMsgSendERC20ToCosmos(sendAmt, erc20Addr, suite.address)
		_, err := suite.app.Erc20Keeper.SendERC20ToCosmos(sdk.WrapSDKContext(suite.ctx), msg)
		suite.Require().Error(err, "expected error when fee cannot be collected (Cosmos->ERC20)")
		suite.Require().Contains(fmt.Sprintf("%v", err), "contract call failed: method 'burnCoins'")
	})

	// Case 3: MsgSendERC20ToCosmosAndIBCTransfer - similar to case 2 but with IBC transfer path
	suite.Run("SendERC20ToCosmosAndIBCTransfer insufficient fee", func() {
		suite.mintFeeCollector = true
		suite.SetupTest()
		_, pair := suite.setupRegisterCoin()
		erc20Addr := common.HexToAddress(pair.Erc20Address)
		tokens := []string{cosmosDenom, pair.Erc20Address}
		mockGasfree := MockGasfreeKeeper{bp: feeBP, tokens: tokens}
		path := fmt.Sprintf("%s/%s", ibctransfertypes.PortID, "channel-0")
		// Set Denom Trace
		denomTrace := ibctransfertypes.DenomTrace{
			Path:      path,
			BaseDenom: cosmosDenom,
		}
		mockTransferKeeper := &testutil.MockTransferKeeper{Keeper: suite.app.BankKeeper, Sequences: make(map[string]uint64)}
		mockTransferKeeper.On("GetDenomTrace", mock.Anything, mock.Anything).Return(denomTrace, true)
		mockTransferKeeper.On("SendTransfer", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		suite.mockKeepers(mockGasfree, mockTransferKeeper)

		sender := sdk.AccAddress(suite.address.Bytes())
		sendAmt := sdk.NewInt(75_000)
		suite.mintCoinAndConvert(sdk.NewCoin(cosmosDenom, sendAmt), sender, types.ModuleName)
		// Ensure no extra fee coins minted
		bankBal := suite.app.BankKeeper.GetBalance(suite.ctx, sender, cosmosDenom)
		suite.Require().True(bankBal.Amount.IsZero())

		msg := types.NewMsgSendERC20ToCosmosAndIBCTransfer(sendAmt, erc20Addr, suite.address, "transfer", "channel-0", sender)

		_, err := suite.app.Erc20Keeper.SendERC20ToCosmosAndIBCTransfer(sdk.WrapSDKContext(suite.ctx), msg)
		suite.Require().Error(err, "expected error when fee cannot be collected (Cosmos->ERC20)")
		suite.Require().Contains(fmt.Sprintf("%v", err), "contract call failed: method 'burnCoins'")
	})
}
