package nativedex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/AltheaFoundation/althea-L1/contracts"
	"github.com/AltheaFoundation/althea-L1/x/nativedex/keeper"
	"github.com/AltheaFoundation/althea-L1/x/nativedex/types"
)

// Callpaths to be used in governance proposals
const BOOT_PATH uint16 = 0
const COLD_PATH uint16 = 3
const SAFEMODE_PATH uint16 = 9999

const (
	UPGRADE_PROXY_CMD      uint8 = 21
	COLLECT_TREASURY_CMD   uint8 = 40
	SET_TREASURY_CMD       uint8 = 41
	AUTHORITY_TRANSFER_CMD uint8 = 20
	HOT_PATH_OPEN_CMD      uint8 = 22
	SET_SAFE_MODE_CMD      uint8 = 23
)

// Return governance handler to process dex governance proposals
func NewNativeDexProposalHandler(k *keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.UpgradeProxyProposal:
			return handleUpgradeProxyProposal(ctx, k, c)
		case *types.CollectTreasuryProposal:
			return handleCollectTreasuryProposal(ctx, k, c)
		case *types.SetTreasuryProposal:
			return handleSetTreasuryProposal(ctx, k, c)
		case *types.AuthorityTransferProposal:
			return handleAuthorityTransferProposal(ctx, k, c)
		case *types.HotPathOpenProposal:
			return handleHotPathOpenProposal(ctx, k, c)
		case *types.SetSafeModeProposal:
			return handleSetSafeModeProposal(ctx, k, c)
		case *types.TransferGovernanceProposal:
			return handleTransferGovernanceProposal(ctx, k, c)
		case *types.OpsProposal:
			return handleOpsProposal(ctx, k, c)

		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized %s proposal content type: %T", types.ModuleName, c)
		}
	}
}

// nolint: dupl
func handleUpgradeProxyProposal(ctx sdk.Context, k *keeper.Keeper, p *types.UpgradeProxyProposal) error {
	err := p.ValidateBasic()
	if err != nil {
		return err
	}
	md := p.GetMetadata()
	callpath := BOOT_PATH
	cmd_code := UPGRADE_PROXY_CMD

	// protocolCmd ABI: (uint8, address, uint16)
	encodedProtocolCmd, err := contracts.EncodeTypes([]string{"uint8", "address", "uint16"}, []interface{}{cmd_code, common.HexToAddress(md.CallpathAddress), uint16(md.CallpathIndex)})
	if err != nil {
		ctx.Logger().Error("Encode protocolCmd args UpgradeProxyProposal", "err", err)
		return err
	}
	// CrocPolicy ABI: treasuryResolution (address, uint16, bytes, bool)
	_, err = k.EVMKeeper.CallEVM(ctx, contracts.CrocPolicyContract.ABI, types.ModuleEVMAddress, k.GetVerifiedCrocPolicyAddress(ctx), true, "treasuryResolution", k.GetNativeDexAddress(ctx), callpath, encodedProtocolCmd, true)
	if err != nil {
		ctx.Logger().Error("Unable to call CrocPolicy.treasuryResolution() for UpgradeProxyProposal", "err", err)
		return err
	}

	return nil
}

// nolint: dupl
func handleCollectTreasuryProposal(ctx sdk.Context, k *keeper.Keeper, p *types.CollectTreasuryProposal) error {
	err := p.ValidateBasic()
	if err != nil {
		return err
	}
	md := p.GetMetadata()
	callpath := COLD_PATH
	if p.InSafeMode {
		callpath = SAFEMODE_PATH
	}
	cmd_code := COLLECT_TREASURY_CMD

	// protocolCmd ABI: (uint8, address)
	encodedProtocolCmd, err := contracts.EncodeTypes([]string{"uint8", "address"}, []interface{}{cmd_code, common.HexToAddress(md.TokenAddress)})
	if err != nil {
		ctx.Logger().Error("Encode protocolCmd args CollectTreasuryProposal", "err", err)
		return err
	}
	// CrocPolicy ABI: treasuryResolution (address, uint16, bytes, bool)
	_, err = k.EVMKeeper.CallEVM(ctx, contracts.CrocPolicyContract.ABI, types.ModuleEVMAddress, k.GetVerifiedCrocPolicyAddress(ctx), true, "treasuryResolution", k.GetNativeDexAddress(ctx), callpath, encodedProtocolCmd, true)
	if err != nil {
		ctx.Logger().Error("Unable to call CrocPolicy. treasuryResolution() for CollectTreasuryProposal", "err", err)
		return err
	}

	return nil
}

// nolint: dupl
func handleSetTreasuryProposal(ctx sdk.Context, k *keeper.Keeper, p *types.SetTreasuryProposal) error {
	err := p.ValidateBasic()
	if err != nil {
		return err
	}
	md := p.GetMetadata()
	callpath := COLD_PATH
	if p.InSafeMode {
		callpath = SAFEMODE_PATH
	}
	cmd_code := SET_TREASURY_CMD

	// protocolCmd ABI: (uint8, address)
	encodedProtocolCmd, err := contracts.EncodeTypes([]string{"uint8", "address"}, []interface{}{cmd_code, common.HexToAddress(md.TreasuryAddress)})
	if err != nil {
		ctx.Logger().Error("Encode protocolCmd args SetTreasuryProposal", "err", err)
		return err
	}
	// CrocPolicy ABI: treasuryResolution (address, uint16, bytes, bool)
	_, err = k.EVMKeeper.CallEVM(ctx, contracts.CrocPolicyContract.ABI, types.ModuleEVMAddress, k.GetVerifiedCrocPolicyAddress(ctx), true, "treasuryResolution", k.GetNativeDexAddress(ctx), callpath, encodedProtocolCmd, true)
	if err != nil {
		ctx.Logger().Error("Unable to call CrocPolicy.treasuryResolution() for SetTreasuryProposal", "err", err)
		return err
	}
	return nil
}

// nolint: dupl
func handleAuthorityTransferProposal(ctx sdk.Context, k *keeper.Keeper, p *types.AuthorityTransferProposal) error {
	err := p.ValidateBasic()
	if err != nil {
		return err
	}
	md := p.GetMetadata()
	callpath := COLD_PATH
	if p.InSafeMode {
		callpath = SAFEMODE_PATH
	}
	cmd_code := AUTHORITY_TRANSFER_CMD

	// protocolCmd ABI: (uint8, address)
	encodedProtocolCmd, err := contracts.EncodeTypes([]string{"uint8", "address"}, []interface{}{cmd_code, common.HexToAddress(md.AuthAddress)})
	if err != nil {
		ctx.Logger().Error("Encode protocolCmd args AuthorityTransferProposal", "err", err)
		return err
	}
	// CrocPolicy ABI: treasuryResolution (address, uint16, bytes, bool)
	_, err = k.EVMKeeper.CallEVM(ctx, contracts.CrocPolicyContract.ABI, types.ModuleEVMAddress, k.GetVerifiedCrocPolicyAddress(ctx), true, "treasuryResolution", k.GetNativeDexAddress(ctx), callpath, encodedProtocolCmd, true)
	if err != nil {
		ctx.Logger().Error("Unable to call CrocPolicy.treasuryResolution() for AuthorityTransferProposal", "err", err)
		return err
	}
	return nil
}

// nolint: dupl
func handleHotPathOpenProposal(ctx sdk.Context, k *keeper.Keeper, p *types.HotPathOpenProposal) error {
	err := p.ValidateBasic()
	if err != nil {
		return err
	}
	md := p.GetMetadata()
	callpath := COLD_PATH
	if p.InSafeMode {
		callpath = SAFEMODE_PATH
	}
	cmd_code := HOT_PATH_OPEN_CMD

	// protocolCmd ABI: (uint8, address)
	encodedProtocolCmd, err := contracts.EncodeTypes([]string{"uint8", "bool"}, []interface{}{cmd_code, md.Open})
	if err != nil {
		ctx.Logger().Error("Encode protocolCmd args HotPathOpenProposal", "err", err)
		return err
	}
	// CrocPolicy ABI: treasuryResolution (address, uint16, bytes, bool)
	_, err = k.EVMKeeper.CallEVM(ctx, contracts.CrocPolicyContract.ABI, types.ModuleEVMAddress, k.GetVerifiedCrocPolicyAddress(ctx), true, "treasuryResolution", k.GetNativeDexAddress(ctx), callpath, encodedProtocolCmd, true)
	if err != nil {
		ctx.Logger().Error("Unable to call CrocPolicy.treasuryResolution() for HotPathOpenProposal", "err", err)
		return err
	}
	return nil
}

// nolint: dupl
func handleSetSafeModeProposal(ctx sdk.Context, k *keeper.Keeper, p *types.SetSafeModeProposal) error {
	err := p.ValidateBasic()
	if err != nil {
		return err
	}
	md := p.GetMetadata()
	callpath := COLD_PATH
	if p.InSafeMode {
		callpath = SAFEMODE_PATH
	}
	cmd_code := SET_SAFE_MODE_CMD

	// protocolCmd ABI: (uint8, address)
	encodedProtocolCmd, err := contracts.EncodeTypes([]string{"uint8", "bool"}, []interface{}{cmd_code, md.LockDex})
	if err != nil {
		ctx.Logger().Error("Encode protocolCmd args SetSafeModeProposal", "err", err)
		return err
	}
	// CrocPolicy ABI: treasuryResolution (address, uint16, bytes, bool)
	_, err = k.EVMKeeper.CallEVM(ctx, contracts.CrocPolicyContract.ABI, types.ModuleEVMAddress, k.GetVerifiedCrocPolicyAddress(ctx), true, "treasuryResolution", k.GetNativeDexAddress(ctx), callpath, encodedProtocolCmd, true)
	if err != nil {
		ctx.Logger().Error("Unable to call CrocPolicy.treasuryResolution() for SetSafeModeProposal", "err", err)
		return err
	}
	return nil
}

// nolint: dupl
func handleTransferGovernanceProposal(ctx sdk.Context, k *keeper.Keeper, p *types.TransferGovernanceProposal) error {
	err := p.ValidateBasic()
	if err != nil {
		return err
	}
	md := p.GetMetadata()
	// This proposal does not directly work on the DEX, so no callpath nor cmd_code are used
	// CrocPolicy ABI: transferGovernance (address ops, address treasury, address emergency)
	ops := common.HexToAddress(md.Ops)
	emergency := common.HexToAddress(md.Emergency)
	_, err = k.EVMKeeper.CallEVM(ctx, contracts.CrocPolicyContract.ABI, types.ModuleEVMAddress, k.GetVerifiedCrocPolicyAddress(ctx), true, "transferGovernance", ops, types.ModuleEVMAddress, emergency)
	if err != nil {
		ctx.Logger().Error("Unable to call CrocPolicy.transferGovernance() for TransferGovernanceProposal", "err", err)
		return err
	}
	return nil
}

// nolint: dupl
func handleOpsProposal(ctx sdk.Context, k *keeper.Keeper, p *types.OpsProposal) error {
	err := p.ValidateBasic()
	if err != nil {
		return err
	}
	md := p.GetMetadata()
	callpath := uint16(md.Callpath)
	// CrocPolicy ABI: opsResolution (address minion, uint16 proxyPath, bytes cmd)
	_, err = k.EVMKeeper.CallEVM(ctx, contracts.CrocPolicyContract.ABI, types.ModuleEVMAddress, k.GetVerifiedCrocPolicyAddress(ctx), true, "opsResolution", k.GetNativeDexAddress(ctx), callpath, md.CmdArgs)
	if err != nil {
		ctx.Logger().Error("Unable to call CrocPolicy.opsResolution() for OpsProposal", "err", err)
		return err
	}
	return nil
}
