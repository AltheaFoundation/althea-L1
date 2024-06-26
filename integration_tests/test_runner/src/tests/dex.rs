//! Contains usage tests for the Ambient (CrocSwap) dex deployment

use std::time::Duration;

use crate::bootstrapping::DexAddresses;
use crate::dex_utils::{
    croc_policy_ops_resolution, croc_policy_treasury_resolution, croc_query_curve_tick,
    croc_query_dex, croc_query_pool_params, croc_query_pool_template, croc_query_range_position,
    dex_authority_transfer, dex_direct_protocol_cmd, dex_query_authority, dex_query_safe_mode,
    dex_swap, dex_user_cmd, OpsResolutionArgs, ProtocolCmdArgs, SwapArgs, UserCmdArgs, BOOT_PATH,
    COLD_PATH, MAX_PRICE, MIN_PRICE, WARM_PATH,
};
use crate::type_urls::{
    COLLECT_TREASURY_PROPOSAL_TYPE_URL, HOT_PATH_OPEN_PROPOSAL_TYPE_URL, OPS_PROPOSAL_TYPE_URL,
    SET_SAFE_MODE_PROPOSAL_TYPE_URL, SET_TREASURY_PROPOSAL_TYPE_URL,
    TRANSFER_GOVERNANCE_PROPOSAL_TYPE_URL, UPGRADE_PROXY_PROPOSAL_TYPE_URL,
};
use crate::utils::{
    encode_any, one_atom, one_eth, vote_yes_on_proposals, wait_for_proposals_to_execute,
    EthermintUserKey, ValidatorKeys, MINER_ETH_ADDRESS, MINER_PRIVATE_KEY, OPERATION_TIMEOUT,
    STAKING_TOKEN,
};
use althea_proto::althea::nativedex::v1::{
    CollectTreasuryMetadata, CollectTreasuryProposal, HotPathOpenMetadata, HotPathOpenProposal,
    OpsMetadata, OpsProposal, SetSafeModeMetadata, SetSafeModeProposal, SetTreasuryMetadata,
    SetTreasuryProposal, TransferGovernanceMetadata, TransferGovernanceProposal,
    UpgradeProxyMetadata, UpgradeProxyProposal,
};
use althea_proto::cosmos_sdk_proto::cosmos::params::v1beta1::{
    ParamChange, ParameterChangeProposal,
};
use clarity::{Address as EthAddress, PrivateKey, Uint256};
use deep_space::{Coin, Contact};
use num::ToPrimitive;

use rand::Rng;
use web30::client::Web3;
use web30::jsonrpc::error::Web3Error;
use web30::types::TransactionResponse;

lazy_static! {
    pub static ref POOL_IDX: Uint256 = 36000u32.into();
}

pub async fn dex_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: Vec<ValidatorKeys>,
    evm_user_keys: Vec<EthermintUserKey>,
    erc20_contracts: Vec<EthAddress>,
    dex_contracts: DexAddresses,
) {
    let evm_user = evm_user_keys.first().unwrap();
    let evm_privkey = evm_user.eth_privkey;
    let optional_caller = Some(evm_user.eth_address);
    let croc_query_contract = dex_contracts.query;
    let dex_result = croc_query_dex(web3, croc_query_contract, optional_caller).await;
    assert!(dex_result.is_ok(), "Bad result");
    let dex = dex_result.unwrap();
    assert_eq!(dex, dex_contracts.dex, "Dex contract address mismatch");

    let (pool_base, pool_quote) = pool_tokens(erc20_contracts.clone());

    basic_dex_setup(
        contact,
        web3,
        dex_contracts.dex,
        dex_contracts.query,
        dex_contracts.policy,
        evm_user,
        &validator_keys,
        pool_base,
        pool_quote,
    )
    .await;

    let tick = croc_query_curve_tick(
        web3,
        dex_contracts.query,
        Some(evm_user.eth_address),
        pool_base,
        pool_quote,
        *POOL_IDX,
    )
    .await
    .expect("Could not get curve tick for pool");

    let bid_tick = tick - 75u8.into();
    let ask_tick = tick + 75u8.into();
    // uint8 code, address base, address quote, uint256 poolIdx,
    //  int24 bidTick, int24 askTick, uint128 liq,
    //  uint128 limitLower, uint128 limitHigher, //  uint8 reserveFlags, address lpConduit
    let range_pos = croc_query_range_position(
        web3,
        dex_contracts.query,
        None,
        evm_user.eth_address,
        pool_base,
        pool_quote,
        *POOL_IDX,
        bid_tick,
        ask_tick,
    )
    .await
    .expect("Could not query position");
    if range_pos.liq > 0u8.into() {
        info!("Range position already exists: {:?}", range_pos);
    } else {
        let mint_ranged_pos_args = UserCmdArgs {
            callpath: WARM_PATH, // Warm Path index
            cmd: vec![
                Uint256::from(1u8).into(),            // Mint Ranged Liq in base token code
                pool_base.into(),                     // base
                pool_quote.into(),                    // quote
                (*POOL_IDX).into(),                   // poolIdx
                bid_tick.into(),                      // bid (lower) tick
                ask_tick.into(),                      // ask (upper) tick
                (one_eth() * 10240u32.into()).into(), // liq (in liquidity units, which must be a multiple of 1024)
                (*MIN_PRICE).into(),                  // limitLower
                (*MAX_PRICE).into(),                  // limitHigher
                Uint256::from(0u8).into(),            // reserveFlags
                EthAddress::default().into(),         // lpConduit
            ],
        };
        info!("Minting position in both tokens: {mint_ranged_pos_args:?}");
        dex_user_cmd(
            web3,
            dex_contracts.dex,
            evm_privkey,
            mint_ranged_pos_args,
            None,
            None,
        )
        .await
        .expect("Failed to mint position in pool");
        let range_pos = croc_query_range_position(
            web3,
            dex_contracts.query,
            None,
            evm_user.eth_address,
            pool_base,
            pool_quote,
            *POOL_IDX,
            bid_tick,
            ask_tick,
        )
        .await
        .expect("Could not query position");
        assert!(range_pos.liq > 0u8.into());
    }

    // Finally, perform many smaller swaps to ensure the pool is working as expected
    swap_many(web3, dex_contracts, pool_base, pool_quote, evm_user, 30).await;

    info!("Successfully tested DEX");
}

pub async fn dex_upgrade_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: Vec<ValidatorKeys>,
    evm_user_keys: Vec<EthermintUserKey>,
    erc20_contracts: Vec<EthAddress>,
    dex_contracts: DexAddresses,
) {
    let evm_user = evm_user_keys.first().unwrap();
    let emergency_user = evm_user_keys.last().unwrap();
    let (pool_base, pool_quote) = pool_tokens(erc20_contracts.clone());

    let evm_privkey = evm_user.eth_privkey;

    basic_dex_setup(
        contact,
        web3,
        dex_contracts.dex,
        dex_contracts.query,
        dex_contracts.policy,
        evm_user,
        &validator_keys,
        pool_base,
        pool_quote,
    )
    .await;

    // Here we init a new pool using the callpath instaleld in the upgrade proposal (index 33). We want to use the same tokens, so we need to use a different
    // pool index
    init_pool(
        web3,
        evm_privkey,
        dex_contracts.dex,
        pool_base,
        pool_quote,
        Some(36001u64.into()),
        Some(33),
    )
    .await
    .expect_err("Callpath should not have been installed yet");

    // Try to upgrade the dex contract with the Ops address, not in sudo mode
    bad_upgrade_proxy_call(
        web3,
        dex_contracts.dex,
        dex_contracts.upgrade,
        33,
        *MINER_PRIVATE_KEY,
        Some(OPERATION_TIMEOUT),
        false,
    )
    .await
    .expect_err("Expected bad upgrade call to fail");
    // ... and in sudo mode
    bad_upgrade_proxy_call(
        web3,
        dex_contracts.dex,
        dex_contracts.upgrade,
        33,
        *MINER_PRIVATE_KEY,
        Some(OPERATION_TIMEOUT),
        true,
    )
    .await
    .expect_err("Expected bad upgrade call to fail");
    // Try to upgrade the dex contract with the Emergency address, not in sudo mode
    bad_upgrade_proxy_call(
        web3,
        dex_contracts.dex,
        dex_contracts.upgrade,
        33,
        emergency_user.eth_privkey,
        Some(OPERATION_TIMEOUT),
        false,
    )
    .await
    .expect_err("Expected bad upgrade call to fail");
    // ... and in sudo mode
    bad_upgrade_proxy_call(
        web3,
        dex_contracts.dex,
        dex_contracts.upgrade,
        33,
        emergency_user.eth_privkey,
        Some(OPERATION_TIMEOUT),
        true,
    )
    .await
    .expect_err("Expected bad upgrade call to fail");

    info!("Testing upgrade proposal");
    submit_and_pass_upgrade_proxy_proposal(contact, &validator_keys, 33, dex_contracts.upgrade)
        .await;

    // Here we init a new pool using the callpath instaleld in the upgrade proposal (index 33). We want to use the same tokens, so we need to use a different
    // pool index
    init_pool(
        web3,
        evm_privkey,
        dex_contracts.dex,
        pool_base,
        pool_quote,
        Some(36001u64.into()),
        Some(33),
    )
    .await
    .expect("Could not create pool");

    // Now we query the pool params to ensure it was set up correctly, we use the new pool index to locate the right pool
    let params = croc_query_pool_params(
        web3,
        dex_contracts.query,
        Some(evm_user.eth_address),
        pool_base,
        pool_quote,
        36001u64.into(),
    )
    .await
    .expect("Could not query pool");
    assert!(params.tick_size != 0, "Pool not created");

    info!("Attempt to steal DEX control away from CrocPolicy contract");
    dex_direct_protocol_cmd(
        web3,
        dex_contracts.dex,
        *MINER_PRIVATE_KEY,
        ProtocolCmdArgs {
            callpath: COLD_PATH,
            cmd: vec![Uint256::from(20u16).into(), (*MINER_ETH_ADDRESS).into()],
            sudo: true,
        },
        None,
        None,
    )
    .await
    .expect_err("Miner should not be able to take control away from CrocPolicy");
    croc_policy_treasury_resolution(
        web3,
        dex_contracts.policy,
        dex_contracts.dex,
        *MINER_PRIVATE_KEY,
        ProtocolCmdArgs {
            callpath: COLD_PATH,
            cmd: vec![Uint256::from(20u16).into(), (*MINER_ETH_ADDRESS).into()],
            sudo: true,
        },
        None,
        None,
    )
    .await
    .expect_err("Miner should not be able to take control away from CrocPolicy");

    info!("Successfully tested DEX upgrade via nativedex governance");
}

pub async fn dex_safe_mode_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: Vec<ValidatorKeys>,
    evm_user_keys: Vec<EthermintUserKey>,
    erc20_contracts: Vec<EthAddress>,
    dex_contracts: DexAddresses,
) {
    let emergency_user = evm_user_keys.last().unwrap();
    let evm_user = evm_user_keys.first().unwrap();
    let (pool_base, pool_quote) = pool_tokens(erc20_contracts.clone());

    basic_dex_setup(
        contact,
        web3,
        dex_contracts.dex,
        dex_contracts.query,
        dex_contracts.policy,
        evm_user,
        &validator_keys,
        pool_base,
        pool_quote,
    )
    .await;
    let tick = croc_query_curve_tick(
        web3,
        dex_contracts.query,
        Some(evm_user.eth_address),
        pool_base,
        pool_quote,
        *POOL_IDX,
    )
    .await
    .expect("Could not get curve tick for pool");

    let bid_tick = tick - 75u8.into();
    let ask_tick = tick + 75u8.into();

    // These paramers mint a position in the base token (11) active from bid tick to ask tick with as many tokens it takes to 1 billion lots of liquidity
    // but the tx will be reverted if the price at execution is outside of [MIN_PRICE, MAX_PRICE]
    let liquidity_amount: Uint256 = Uint256::from(1_000_000_000u32) * 1024u32.into();
    let mint_ranged_pos_args = UserCmdArgs {
        callpath: WARM_PATH, // Warm Path index
        cmd: vec![
            Uint256::from(11u8).into(),   // Mint Ranged Liq with base token
            pool_base.into(),             // base
            pool_quote.into(),            // quote
            (*POOL_IDX).into(),           // poolIdx
            bid_tick.into(),              // bid (lower) tick
            ask_tick.into(),              // ask (upper) tick
            liquidity_amount.into(),      // liq
            (*MIN_PRICE).into(),          // limitLower
            (*MAX_PRICE).into(),          // limitHigher
            Uint256::from(0u8).into(),    // reserveFlags
            EthAddress::default().into(), // lpConduit
        ],
    };
    info!("Minting position in pool: {mint_ranged_pos_args:?}");
    dex_user_cmd(
        web3,
        dex_contracts.dex,
        evm_user.eth_privkey,
        mint_ranged_pos_args,
        None,
        None,
    )
    .await
    .expect("Failed to mint position in pool");

    // Set the Ops and Emergency roles on CrocPolicy: ops = Miner, emergency = last evm user
    submit_and_pass_transfer_governance_proposal(
        contact,
        &validator_keys,
        *MINER_ETH_ADDRESS,
        emergency_user.eth_address,
    )
    .await;

    info!("Testing safe mode");
    submit_and_pass_safe_mode_proposal(contact, &validator_keys, true, false).await;

    safe_mode_operations(
        true,
        web3,
        dex_contracts.clone(),
        evm_user_keys.clone(),
        &validator_keys,
        pool_base,
        pool_quote,
    )
    .await;
    info!("Disabling safe mode");
    submit_and_pass_safe_mode_proposal(contact, &validator_keys, false, true).await;

    assert!(
        !dex_query_safe_mode(web3, dex_contracts.dex, Some(evm_user.eth_address))
            .await
            .expect("Unable to query safe mode"),
        "dex should not be in safe mode"
    );

    safe_mode_operations(
        false,
        web3,
        dex_contracts.clone(),
        evm_user_keys.clone(),
        &validator_keys,
        pool_base,
        pool_quote,
    )
    .await;

    info!("Successfully tested DEX safe mode");
}

pub async fn dex_ops_proposal_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: Vec<ValidatorKeys>,
    evm_user_keys: Vec<EthermintUserKey>,
    erc20_contracts: Vec<EthAddress>,
    dex_contracts: DexAddresses,
) {
    let evm_user = evm_user_keys.first().unwrap();
    let (pool_base, pool_quote) = pool_tokens(erc20_contracts.clone());

    basic_dex_setup(
        contact,
        web3,
        dex_contracts.dex,
        dex_contracts.query,
        dex_contracts.policy,
        evm_user,
        &validator_keys,
        pool_base,
        pool_quote,
    )
    .await;
    let callpath = COLD_PATH;
    let code: Uint256 = 110u8.into();
    let index: Uint256 = 36003u32.into();
    let fee_rate: Uint256 = 1234u16.into();
    let tick_size: Uint256 = 100u16.into();
    let jit_thresh: Uint256 = 0u8.into();
    let knockout: Uint256 = 0u8.into();
    let oracle: Uint256 = 0u8.into();

    let cmd_args = clarity::abi::encode_tokens(&[
        code.into(),
        index.into(),
        fee_rate.into(),
        tick_size.into(),
        jit_thresh.into(),
        knockout.into(),
        oracle.into(),
    ]);

    let pre_template = croc_query_pool_template(
        web3,
        dex_contracts.query,
        Some(evm_user_keys.first().unwrap().eth_address),
        index,
    )
    .await
    .expect("Unable to query pool template");
    assert!(
        tick_size != pre_template.tick_size.into(),
        "Pool template already exists!"
    );
    // function setTemplate (bytes calldata input) private {
    //     (, uint256 poolIdx, uint16 feeRate, uint16 tickSize, uint8 jitThresh,
    //      uint8 knockout, uint8 oracleFlags) =
    //         abi.decode(input, (uint8, uint256, uint16, uint16, uint8, uint8, uint8));
    submit_and_pass_nativedex_ops_proposal(contact, &validator_keys, callpath, cmd_args).await;

    let post_template = croc_query_pool_template(
        web3,
        dex_contracts.query,
        Some(evm_user_keys.first().unwrap().eth_address),
        index,
    )
    .await
    .expect("Unable to query pool template");
    assert!(
        fee_rate == post_template.fee_rate.into()
            && tick_size == post_template.tick_size.into()
            && jit_thresh == post_template.jit_thresh.into(),
        "Pool template not updated"
    );

    info!("Successfully tested nativedex OpsProposal");
}

pub fn pool_tokens(erc20_contracts: Vec<EthAddress>) -> (EthAddress, EthAddress) {
    let tokens = erc20_contracts[0..2].to_vec();
    if tokens[0] < tokens[1] {
        (tokens[0], tokens[1])
    } else {
        (tokens[1], tokens[0])
    }
}

async fn swap_many(
    web3: &Web3,
    dex_contracts: DexAddresses,
    pool_base: EthAddress,
    pool_quote: EthAddress,
    evm_user: &EthermintUserKey,
    swaps: usize,
) {
    let mut is_buy = true; // Switch direction frequently, starting with base for quote
    let mut rng = rand::thread_rng();

    for _ in 0..swaps {
        let qty_multi: u32 = rng.gen();
        let qty = one_atom() * qty_multi.into();
        let pre_swap_base = web3
            .get_erc20_balance(pool_base, evm_user.eth_address)
            .await
            .expect("Unable to get erc20 balance");
        let pre_swap_quote = web3
            .get_erc20_balance(pool_quote, evm_user.eth_address)
            .await
            .expect("Unable to get erc20 balance");
        // Swap quote for base, expecting one atom of base out
        let swap_args = SwapArgs {
            base: pool_base,
            quote: pool_quote,
            pool_idx: *POOL_IDX,
            is_buy,
            in_base_qty: is_buy, // Always specify the input qty, not the output
            qty,
            tip: 0,
            limit_price: if is_buy { *MAX_PRICE } else { *MIN_PRICE }, // Eliminate price checking failures
            min_out: 0u8.into(), // Eliminate output checking failures
            reserve_flags: 0u8,
        };

        info!(
            "Swapping {} for {}",
            qty.to_string() + if is_buy { "base" } else { "quote" },
            if is_buy { "quote" } else { "base" }.to_string()
        );
        dex_swap(
            web3,
            dex_contracts.dex,
            evm_user.eth_privkey,
            swap_args,
            None,
            Some(OPERATION_TIMEOUT),
        )
        .await
        .expect("Unable to swap quote for base");
        let post_swap_base = web3
            .get_erc20_balance(pool_base, evm_user.eth_address)
            .await
            .expect("Unable to get erc20 balance");
        let post_swap_quote = web3
            .get_erc20_balance(pool_quote, evm_user.eth_address)
            .await
            .expect("Unable to get erc20 balance");

        if is_buy {
            assert!(
                post_swap_base < pre_swap_base && pre_swap_base - post_swap_base == qty,
                "Swap did not decrease base token balance"
            );
            assert!(
                post_swap_quote > pre_swap_quote,
                "Swap did not increase quote token balance"
            );
        } else {
            assert!(
                post_swap_quote < pre_swap_quote && pre_swap_quote - post_swap_quote == qty,
                "Swap did not decrease quote token balance"
            );
            assert!(
                post_swap_base > pre_swap_base,
                "Swap did not increase base token balance"
            );
        }

        is_buy = !is_buy;
    }
}

async fn safe_mode_operations(
    in_safe_mode: bool,
    web3: &Web3,
    dex_contracts: DexAddresses,
    evm_user_keys: Vec<EthermintUserKey>,
    _validator_keys: &[ValidatorKeys],
    pool_base: EthAddress,
    pool_quote: EthAddress,
) {
    let user = evm_user_keys.first().unwrap();

    // These swap args perform a swap from quote to base token (is_buy) with 1_000_000 (qty) of quote tokens input (in_base_qty), and the transaction will revert if
    // the swap results in less than 0 base tokens out or if the price goes below MIN_PRICE (below because is_buy = false)
    let swap_args = SwapArgs {
        base: pool_base,
        quote: pool_quote,
        pool_idx: *POOL_IDX,
        is_buy: false,
        in_base_qty: false,
        qty: one_atom(),
        tip: 0,
        limit_price: *MIN_PRICE,
        min_out: 0u8.into(),
        reserve_flags: 0u8,
    };
    let test_swap = dex_swap(
        web3,
        dex_contracts.dex,
        user.eth_privkey,
        swap_args,
        None,
        Some(OPERATION_TIMEOUT),
    )
    .await;

    if in_safe_mode {
        test_swap.expect_err("Swap should fail in safe mode");
    } else {
        test_swap.expect("Swap should succeed outside of safe mode");
    }
    let set_liq_res = croc_policy_ops_resolution(
        web3,
        dex_contracts.policy,
        *MINER_PRIVATE_KEY,
        OpsResolutionArgs {
            minion: dex_contracts.dex,
            callpath: COLD_PATH,
            cmd: vec![Uint256::from(112u8).into(), Uint256::from(10u128).into()],
        },
        None,
        None,
    )
    .await;
    if in_safe_mode {
        set_liq_res.expect_err("Non sudo command should fail in safe mode");
    } else {
        set_liq_res.expect("Non sudo command should succeed outside of safe mode");
    }
}
#[allow(clippy::too_many_arguments)]
pub async fn basic_dex_setup(
    contact: &Contact,
    web3: &Web3,
    dex: EthAddress,
    query: EthAddress,
    policy: EthAddress,
    evm_user: &EthermintUserKey,
    validator_keys: &[ValidatorKeys],
    pool_base: EthAddress,
    pool_quote: EthAddress,
) {
    let current_auth = dex_query_authority(web3, dex, Some(evm_user.eth_address))
        .await
        .expect("Unable to query current authority");
    if current_auth != policy {
        // Transfer authority to the CrocPolicy contract, so nativedex governance can manage the DEX
        dex_authority_transfer(
            web3,
            dex,
            policy,
            *MINER_PRIVATE_KEY,
            Some(OPERATION_TIMEOUT),
        )
        .await
        .expect("Unable to transfer dex ownership to the CrocPolicy contract");
        info!("Transferred DEX authority_ address to CrocPolicy contract for nativedex governance control");
        // Submit and pass ParamChangeProposal to use these contracts with the nativedex module
        submit_and_pass_nativedex_config_proposal(contact, validator_keys, dex, policy).await;
    }

    let safe_mode = dex_query_safe_mode(web3, dex, Some(evm_user.eth_address))
        .await
        .expect("Unable to query safe mode");
    if safe_mode {
        info!("Dex is in safe mode, disabling safe mode");
        submit_and_pass_safe_mode_proposal(contact, validator_keys, false, true).await;
    }
    let pool = croc_query_pool_params(
        web3,
        query,
        Some(evm_user.eth_address),
        pool_base,
        pool_quote,
        *POOL_IDX,
    )
    .await;
    if pool.is_ok() && pool.unwrap().tick_size != 0 {
        info!("Pool already created, approving use of base and quote tokens");
        web3.approve_erc20_transfers(
            pool_base,
            evm_user.eth_privkey,
            dex,
            Some(OPERATION_TIMEOUT),
            vec![],
        )
        .await
        .expect("Could not approve base token");
        web3.approve_erc20_transfers(
            pool_quote,
            evm_user.eth_privkey,
            dex,
            Some(OPERATION_TIMEOUT),
            vec![],
        )
        .await
        .expect("Could not approve base token");
    } else {
        info!("Creating pool");
        init_pool(
            web3,
            evm_user.eth_privkey,
            dex,
            pool_base,
            pool_quote,
            None,
            None,
        )
        .await
        .expect("Could not create pool");
    }
}

pub async fn init_pool(
    web3: &Web3,
    evm_privkey: PrivateKey,
    dex: EthAddress,
    pool_base: EthAddress,
    pool_quote: EthAddress,
    template_override: Option<Uint256>,
    callpath_override: Option<u16>,
) -> Result<TransactionResponse, Web3Error> {
    let callpath: u16 = callpath_override.unwrap_or(COLD_PATH);
    let template = template_override.unwrap_or(*POOL_IDX);
    if pool_base >= pool_quote {
        panic!("Base token must be lexically less than quote token");
    }
    let price: Uint256 = (f64::sqrt(10f64.powf(-12.0)) * 2f64.powf(64.0))
        .round()
        .to_u128()
        .unwrap()
        .into();
    let init_pool_args = UserCmdArgs {
        callpath,
        cmd: vec![
            71u8.into(), // Init pool code
            pool_base.into(),
            pool_quote.into(),
            template.into(),
            price.into(),
        ],
    };
    if pool_base != EthAddress::default() {
        web3.approve_erc20_transfers(pool_base, evm_privkey, dex, Some(OPERATION_TIMEOUT), vec![])
            .await
            .expect("Could not approve base token");
    }
    web3.approve_erc20_transfers(
        pool_quote,
        evm_privkey,
        dex,
        Some(OPERATION_TIMEOUT),
        vec![],
    )
    .await
    .expect("Could not approve quote token");

    dex_user_cmd(web3, dex, evm_privkey, init_pool_args, None, None).await
}

/// Configures the nativedex module to use the given addresses as the CrocSwapDEX and CrocPolicy when executing gov proposals
pub async fn submit_and_pass_nativedex_config_proposal(
    contact: &Contact,
    keys: &[ValidatorKeys],
    dex_contract: EthAddress,
    policy_contract: EthAddress,
) {
    let deposit = Coin {
        amount: one_atom() * 100u8.into(),
        denom: STAKING_TOKEN.clone(),
    };
    let fee = Coin {
        amount: 0u8.into(),
        denom: STAKING_TOKEN.clone(),
    };
    let res = contact
        .submit_parameter_change_proposal(
            ParameterChangeProposal {
                title: "Configure nativedex module".to_string(),
                description: "Configure nativedex module".to_string(),
                changes: vec![
                    // subspace defined at x/nativedex/types/keys.go
                    // keys defined at     x/nativedex/types/genesis.go
                    ParamChange {
                        subspace: "nativedex".to_string(),
                        key: "VerifiedNativeDexAddress".to_string(),
                        value: format!("\"{}\"", dex_contract),
                    },
                    ParamChange {
                        subspace: "nativedex".to_string(),
                        key: "VerifiedCrocPolicyAddress".to_string(),
                        value: format!("\"{}\"", policy_contract),
                    },
                ],
            },
            deposit,
            fee,
            keys[0].validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {:?}", res);
}

pub async fn submit_and_pass_upgrade_proxy_proposal(
    contact: &Contact,
    keys: &[ValidatorKeys],
    callpath: u64,
    contract_address: EthAddress,
) {
    let deposit = Coin {
        amount: one_atom() * 100u8.into(),
        denom: STAKING_TOKEN.clone(),
    };
    let fee = Coin {
        amount: 0u8.into(),
        denom: STAKING_TOKEN.clone(),
    };

    let proposal = UpgradeProxyProposal {
        title: "Upgrade Proposal".to_string(),
        description: "Upgrade proposal".to_string(),
        metadata: Some(UpgradeProxyMetadata {
            callpath_address: contract_address.to_string(),
            callpath_index: callpath,
        }),
    };
    let any = encode_any(proposal, UPGRADE_PROXY_PROPOSAL_TYPE_URL.to_string());
    let res = contact
        .create_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {:?}", res);
}

pub async fn submit_and_pass_collect_treasury_proposal(
    contact: &Contact,
    keys: &[ValidatorKeys],
    token_address: EthAddress,
    in_safe_mode: bool,
) {
    let deposit = Coin {
        amount: one_atom() * 100u8.into(),
        denom: STAKING_TOKEN.clone(),
    };
    let fee = Coin {
        amount: 0u8.into(),
        denom: STAKING_TOKEN.clone(),
    };

    let proposal = CollectTreasuryProposal {
        title: "Collect Treasury Proposal".to_string(),
        description: "Collect Treasury proposal".to_string(),
        metadata: Some(CollectTreasuryMetadata {
            token_address: token_address.to_string(),
        }),
        in_safe_mode,
    };
    let any = encode_any(proposal, COLLECT_TREASURY_PROPOSAL_TYPE_URL.to_string());
    let res = contact
        .create_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {:?}", res);
}

pub async fn submit_and_pass_set_treasury_proposal(
    contact: &Contact,
    keys: &[ValidatorKeys],
    treasury_address: EthAddress,
    in_safe_mode: bool,
) {
    let deposit = Coin {
        amount: one_atom() * 100u8.into(),
        denom: STAKING_TOKEN.clone(),
    };
    let fee = Coin {
        amount: 0u8.into(),
        denom: STAKING_TOKEN.clone(),
    };

    let proposal = SetTreasuryProposal {
        title: "Set Treasury Proposal".to_string(),
        description: "Set Treasury proposal".to_string(),
        metadata: Some(SetTreasuryMetadata {
            treasury_address: treasury_address.to_string(),
        }),
        in_safe_mode,
    };
    let any = encode_any(proposal, SET_TREASURY_PROPOSAL_TYPE_URL.to_string());
    let res = contact
        .create_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {:?}", res);
}

pub async fn submit_and_pass_hot_path_open_proposal(
    contact: &Contact,
    keys: &[ValidatorKeys],
    hot_path_open: bool,
    in_safe_mode: bool,
) {
    let deposit = Coin {
        amount: one_atom() * 100u8.into(),
        denom: STAKING_TOKEN.clone(),
    };
    let fee = Coin {
        amount: 0u8.into(),
        denom: STAKING_TOKEN.clone(),
    };

    let proposal = HotPathOpenProposal {
        title: "Hot Path Open Proposal".to_string(),
        description: "Hot Path Open proposal".to_string(),
        metadata: Some(HotPathOpenMetadata {
            open: hot_path_open,
        }),
        in_safe_mode,
    };
    let any = encode_any(proposal, HOT_PATH_OPEN_PROPOSAL_TYPE_URL.to_string());
    let res = contact
        .create_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {:?}", res);
}

pub async fn submit_and_pass_safe_mode_proposal(
    contact: &Contact,
    keys: &[ValidatorKeys],
    lock_dex: bool,
    in_safe_mode: bool,
) {
    let deposit = Coin {
        amount: one_atom() * 100u8.into(),
        denom: STAKING_TOKEN.clone(),
    };
    let fee = Coin {
        amount: 0u8.into(),
        denom: STAKING_TOKEN.clone(),
    };

    let proposal = SetSafeModeProposal {
        title: "Set safe mode".to_string(),
        description: "Set safe mode".to_string(),
        metadata: Some(SetSafeModeMetadata { lock_dex }),
        in_safe_mode,
    };
    let any = encode_any(proposal, SET_SAFE_MODE_PROPOSAL_TYPE_URL.to_string());
    let res = contact
        .create_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {:?}", res);
}

pub async fn submit_and_pass_transfer_governance_proposal(
    contact: &Contact,
    keys: &[ValidatorKeys],
    ops: EthAddress,
    emergency: EthAddress,
) {
    let deposit = Coin {
        amount: one_atom() * 100u8.into(),
        denom: STAKING_TOKEN.clone(),
    };
    let fee = Coin {
        amount: 0u8.into(),
        denom: STAKING_TOKEN.clone(),
    };

    let proposal = TransferGovernanceProposal {
        title: "Transfer Governance Proposal".to_string(),
        description: "Transfer Governance proposal".to_string(),
        metadata: Some(TransferGovernanceMetadata {
            ops: ops.to_string(),
            emergency: emergency.to_string(),
        }),
    };
    let any = encode_any(proposal, TRANSFER_GOVERNANCE_PROPOSAL_TYPE_URL.to_string());
    let res = contact
        .create_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {:?}", res);
}

pub async fn bad_upgrade_proxy_call(
    web30: &Web3,
    dex_contract: EthAddress,
    upgrade: EthAddress,
    callpath: u16,
    wallet: PrivateKey,
    timeout: Option<Duration>,
    sudo: bool,
) -> Result<TransactionResponse, Web3Error> {
    let code: Uint256 = 21u8.into();

    // ABI: upgradeProxy (code uint8, contract address, callpath_index uint16)
    let cmd_args = vec![code.into(), upgrade.into(), Uint256::from(callpath).into()];

    dex_direct_protocol_cmd(
        web30,
        dex_contract,
        wallet,
        ProtocolCmdArgs {
            callpath: BOOT_PATH,
            cmd: cmd_args,
            sudo,
        },
        None,
        timeout,
    )
    .await
}

/// Submits an OpsProposal, to call CrocPolicy.opsResolution with the given args
pub async fn submit_and_pass_nativedex_ops_proposal(
    contact: &Contact,
    keys: &[ValidatorKeys],
    callpath: u16,
    cmd_args: Vec<u8>,
) {
    let deposit = Coin {
        amount: one_atom() * 100u8.into(),
        denom: STAKING_TOKEN.clone(),
    };
    let fee = Coin {
        amount: 0u8.into(),
        denom: STAKING_TOKEN.clone(),
    };

    let proposal = OpsProposal {
        title: "Ops Proposal".to_string(),
        description: "Ops proposal".to_string(),
        metadata: Some(OpsMetadata {
            callpath: callpath as u64,
            cmd_args,
        }),
    };
    let any = encode_any(proposal, OPS_PROPOSAL_TYPE_URL.to_string());
    let res = contact
        .create_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {:?}", res);
}
