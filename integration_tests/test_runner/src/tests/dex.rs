//! Contains usage tests for the Ambient (CrocSwap) dex deployment

use std::time::Duration;

use crate::bootstrapping::DexAddresses;
use crate::dex_utils::{
    croc_policy_ops_resolution, croc_policy_treasury_resolution, croc_query_curve_tick,
    croc_query_dex, croc_query_pool_params, croc_query_pool_template, croc_query_price,
    croc_query_range_position, dex_authority_transfer, dex_direct_protocol_cmd,
    dex_mint_ambient_in_amount, dex_mint_ranged_in_amount, dex_mint_ranged_pos,
    dex_query_authority, dex_query_safe_mode, dex_swap, dex_user_cmd, OpsResolutionArgs,
    ProtocolCmdArgs, SwapArgs, UserCmdArgs, BOOT_PATH, COLD_PATH, MAX_PRICE, MIN_PRICE, WARM_PATH,
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
use num::{Bounded, Zero};
use num256::Int256;
use num_traits::ToPrimitive;
use rand::Rng;
use web30::client::Web3;
use web30::jsonrpc::error::Web3Error;
use web30::types::TransactionResponse;

lazy_static! {
    pub static ref POOL_IDX: Uint256 = 36000u32.into();
}

pub async fn basic_dex_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: Vec<ValidatorKeys>,
    evm_user_keys: Vec<EthermintUserKey>,
    erc20_contracts: Vec<EthAddress>,
    dex_contracts: DexAddresses,
    walthea: EthAddress,
) {
    info!("Start dex test");
    let DexTestParams {
        evm_user,
        caller: _,
        query,
        dex,
        pool_base,
        pool_quote,
    } = setup_params(
        web3,
        evm_user_keys,
        dex_contracts.clone(),
        erc20_contracts.clone(),
    )
    .await;

    basic_dex_setup(
        contact,
        web3,
        dex,
        query,
        dex_contracts.policy,
        &evm_user,
        &validator_keys,
        pool_base,
        pool_quote,
        walthea,
    )
    .await;
    let (a, b) = if walthea < pool_base {
        (walthea, pool_base)
    } else {
        (pool_base, walthea)
    };
    populate_pool_basic(web3, &dex_contracts, &evm_user, a, b).await;

    populate_pool_basic(web3, &dex_contracts, &evm_user, pool_base, pool_quote).await;

    info!("Successfully tested DEX");
}

pub async fn populate_pool_basic(
    web3: &Web3,
    dex_contracts: &DexAddresses,
    evm_user: &EthermintUserKey,
    base: EthAddress,
    quote: EthAddress,
) {
    if base != EthAddress::default()
        && web3
            .get_erc20_allowance(base, evm_user.eth_address, dex_contracts.dex)
            .await
            .expect("Unable to check erc20 approval")
            < Uint256::max_value() / 2u8.into()
    {
        web3.erc20_approve(
            base,
            Uint256::max_value(), // Bad practice but it's a test env so we don't care
            evm_user.eth_privkey,
            dex_contracts.dex,
            Some(OPERATION_TIMEOUT),
            vec![],
        )
        .await
        .expect("Unable to approve erc20");
    }
    if web3
        .get_erc20_allowance(quote, evm_user.eth_address, dex_contracts.dex)
        .await
        .expect("Unable to check erc20 approval")
        < Uint256::max_value() / 2u8.into()
    {
        web3.erc20_approve(
            quote,
            Uint256::max_value(), // Bad practice but it's a test env so we don't care
            evm_user.eth_privkey,
            dex_contracts.dex,
            Some(OPERATION_TIMEOUT),
            vec![],
        )
        .await
        .expect("Unable to approve erc20");
    }

    let ambient_qty = one_eth() * 100000u32.into();
    dex_mint_ambient_in_amount(
        web3,
        dex_contracts.dex,
        dex_contracts.query,
        evm_user.eth_privkey,
        evm_user.eth_address,
        base,
        quote,
        *POOL_IDX,
        ambient_qty,
        false,
    )
    .await;

    let tick = Int256::zero();

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
        base,
        quote,
        *POOL_IDX,
        bid_tick,
        ask_tick,
    )
    .await
    .expect("Could not query position");
    if range_pos.liq > 0u8.into() {
        info!("Range position already exists: {range_pos:?}");
    } else {
        let qty: Uint256 = one_eth() * 1024u32.into(); // 1024 eth
        let bb = web3
            .get_erc20_balance(base, evm_user.eth_address)
            .await
            .unwrap();
        let qb = web3
            .get_erc20_balance(quote, evm_user.eth_address)
            .await
            .unwrap();
        #[allow(deprecated)]
        let ba = web3
            .check_erc20_approved(base, evm_user.eth_address, dex_contracts.dex)
            .await
            .unwrap();
        #[allow(deprecated)]
        let qa = web3
            .check_erc20_approved(quote, evm_user.eth_address, dex_contracts.dex)
            .await
            .unwrap();
        let one_eth_f = one_eth().to_f64().unwrap();
        let bb_f = bb.to_i128().unwrap().to_f64().unwrap();
        let qb_f = qb.to_i128().unwrap().to_f64().unwrap();
        info!("Before minting ranged position: base balance: {}, quote balance: {}, base approved: {}, quote approved: {}", (bb_f) / one_eth_f, (qb_f) / one_eth_f, ba, qa);
        // Mint using the base token
        dex_mint_ranged_pos(
            web3,
            dex_contracts.dex,
            dex_contracts.query,
            evm_user.eth_privkey,
            evm_user.eth_address,
            base,
            quote,
            *POOL_IDX,
            bid_tick,
            ask_tick,
            qty,
        )
        .await;
    }

    // Finally, perform many smaller swaps to ensure the pool is working as expected
    swap_many(web3, dex_contracts, base, quote, evm_user, 30, None).await;
}

// Generates some additional positions, minting and burning ambient and ranged and knockout positions
pub async fn advanced_dex_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: Vec<ValidatorKeys>,
    evm_user_keys: Vec<EthermintUserKey>,
    erc20_contracts: Vec<EthAddress>,
    dex_contracts: DexAddresses,
    walthea: EthAddress,
) {
    info!("Start advanced dex test");
    let DexTestParams {
        evm_user,
        caller: _,
        query: _,
        dex: _,
        pool_base: base,
        pool_quote: quote,
    } = setup_params(
        web3,
        evm_user_keys,
        dex_contracts.clone(),
        erc20_contracts.clone(),
    )
    .await;

    basic_dex_setup(
        contact,
        web3,
        dex_contracts.dex,
        dex_contracts.query,
        dex_contracts.policy,
        &evm_user,
        &validator_keys,
        base,
        quote,
        walthea,
    )
    .await;

    mint_ambient_and_ranged_positions(
        web3,
        dex_contracts.clone(),
        evm_user,
        base,
        quote,
        *POOL_IDX,
    )
    .await;

    mint_ambient_and_ranged_positions(
        web3,
        dex_contracts,
        evm_user,
        EthAddress::default(),
        quote,
        *POOL_IDX,
    )
    .await;

    info!("Successfully tested DEX");
}

pub async fn mint_ambient_and_ranged_positions(
    web3: &Web3,
    dex_contracts: DexAddresses,
    evm_user: EthermintUserKey,
    base: EthAddress,
    quote: EthAddress,
    pool_idx: Uint256,
) {
    let ambient_qty = one_eth() * 100000u32.into();

    dex_mint_ambient_in_amount(
        web3,
        dex_contracts.dex,
        dex_contracts.query,
        evm_user.eth_privkey,
        evm_user.eth_address,
        base,
        quote,
        pool_idx,
        ambient_qty,
        false,
    )
    .await;

    let tick = Int256::zero();

    let liq = one_eth() * 1024u32.into();
    let tick_range = 10240;
    for i in 0..10 {
        let position_width = tick_range / 2; // Make the positions 1/2 of the total range
        let tick_offset = tick_range / 10 * i - (tick_range / 2); // Divide into 10 portions, center around the current tick
        let bid_tick = tick + tick_offset.into() - (Int256::from(position_width) / 2i8.into());
        let ask_tick = tick + tick_offset.into() + (Int256::from(position_width) / 2i8.into());

        info!("Minting position between ticks {bid_tick} and {ask_tick}");

        // Mint two ranged positions (in each token)
        dex_mint_ranged_in_amount(
            web3,
            dex_contracts.dex,
            evm_user.eth_privkey,
            base,
            quote,
            pool_idx,
            bid_tick,
            ask_tick,
            liq,
            false,
        )
        .await;
    }
}

pub async fn dex_swap_many(
    web3: &Web3,
    evm_user_keys: Vec<EthermintUserKey>,
    erc20_contracts: Vec<EthAddress>,
    dex_contracts: DexAddresses,
) {
    info!("Start dex swap test");
    let DexTestParams {
        evm_user,
        caller: _,
        query: _,
        dex: _,
        pool_base,
        pool_quote,
    } = setup_params(
        web3,
        evm_user_keys,
        dex_contracts.clone(),
        erc20_contracts.clone(),
    )
    .await;
    swap_many(
        web3,
        &dex_contracts,
        pool_base,
        pool_quote,
        &evm_user,
        40,
        Some(true),
    )
    .await;
    info!("Successfully swapped many times");
}

pub struct DexTestParams {
    pub evm_user: EthermintUserKey,
    pub caller: Option<EthAddress>,
    pub query: EthAddress,
    pub dex: EthAddress,
    pub pool_base: EthAddress,
    pub pool_quote: EthAddress,
}

pub async fn setup_params(
    web3: &Web3,
    evm_user_keys: Vec<EthermintUserKey>,
    dex_contracts: DexAddresses,
    erc20_contracts: Vec<EthAddress>,
) -> DexTestParams {
    let evm_user = *evm_user_keys.first().unwrap();
    let optional_caller = Some(evm_user.eth_address);
    let croc_query_contract = dex_contracts.query;
    let dex_result = croc_query_dex(web3, croc_query_contract, optional_caller).await;
    assert!(dex_result.is_ok(), "Bad result");
    let dex = dex_result.unwrap();
    assert_eq!(dex, dex_contracts.dex, "Dex contract address mismatch");

    let (pool_base, pool_quote) = pool_tokens(erc20_contracts);

    DexTestParams {
        evm_user,
        caller: optional_caller,
        query: croc_query_contract,
        dex,
        pool_base,
        pool_quote,
    }
}

pub async fn dex_upgrade_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: Vec<ValidatorKeys>,
    evm_user_keys: Vec<EthermintUserKey>,
    erc20_contracts: Vec<EthAddress>,
    dex_contracts: DexAddresses,
    walthea: EthAddress,
) {
    info!("Start dex upgrade test");
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
        walthea,
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
    walthea: EthAddress,
) {
    info!("Start dex safe mode test");
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
        walthea,
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
    walthea: EthAddress,
) {
    info!("Start dex OpsProposal test");
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
        walthea,
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
    dex_contracts: &DexAddresses,
    pool_base: EthAddress,
    pool_quote: EthAddress,
    evm_user: &EthermintUserKey,
    swaps: usize,
    direction: Option<bool>,
) {
    let mut is_buy = direction.unwrap_or(true);
    let mut rng = rand::thread_rng();

    for _ in 0..swaps {
        let mut qty_multi: u8 = rng.gen(); // 0 to 255
        if qty_multi == 0 {
            qty_multi = 1;
        }
        let qty = Uint256::from(1000000000000000u128) * qty_multi.into(); // .001 eth * qty_multi
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
            if is_buy { "quote" } else { "base" }
        );
        let native_in = if pool_base == EthAddress::default() {
            Some(qty)
        } else {
            None
        };
        dex_swap(
            web3,
            dex_contracts.dex,
            evm_user.eth_privkey,
            swap_args,
            native_in,
            Some(OPERATION_TIMEOUT),
        )
        .await
        .expect("Unable to swap");
        let price = croc_query_price(
            web3,
            dex_contracts.query,
            Some(*MINER_ETH_ADDRESS),
            pool_base,
            pool_quote,
            *POOL_IDX,
        )
        .await
        .expect("Unable to query price");
        info!("Price: {price}");
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
            info!(
                "Swapped {} base for {} quote",
                pre_swap_base - post_swap_base,
                post_swap_quote - pre_swap_quote
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
            info!(
                "Swapped {} quote for {} base",
                pre_swap_quote - post_swap_quote,
                post_swap_base - pre_swap_base,
            );
        }

        if direction.is_none() {
            is_buy = !is_buy;
        }
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

/// Performs critical DEX setup operations, including:
/// - Transferring authority to the CrocPolicy contract if not done yet
/// - Disabling safe mode if enabled
/// - Wrapping native token if needed
/// - Creating or preparing a pool with base and quote tokens
/// - Creating or preparing a pool with native token and quote tokens
/// - Creating or preparing a pool with wrapped althea and base tokens
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
    walthea: EthAddress,
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
    info!(
        "Evm user native token balance: {}",
        web3.eth_get_balance(evm_user.eth_address).await.unwrap()
    );
    if web3
        .get_erc20_balance(walthea, evm_user.eth_address)
        .await
        .unwrap()
        < one_eth() * 1_000_000u32.into()
    {
        web3.wrap_eth(
            one_eth() * 1_000_000u32.into(),
            evm_user.eth_privkey,
            Some(walthea),
            Some(Duration::from_secs(15)),
        )
        .await
        .expect("Unable to wrap native token!");
    }

    // Create or prepare the pool with Base and wAlthea tokens
    let (a, b) = if walthea < pool_base {
        (walthea, pool_base)
    } else {
        (pool_base, walthea)
    };
    create_or_prepare_pool(web3, dex, query, evm_user, a, b, *POOL_IDX).await;
    // Create or prepare the pool with Base and Quote tokens
    create_or_prepare_pool(web3, dex, query, evm_user, pool_base, pool_quote, *POOL_IDX).await;

    // Create or prepare a pool with native token and quote tokens
    create_or_prepare_pool(
        web3,
        dex,
        query,
        evm_user,
        EthAddress::default(),
        pool_quote,
        *POOL_IDX,
    )
    .await;
}

pub async fn create_or_prepare_pool(
    web3: &Web3,
    dex: EthAddress,
    query: EthAddress,
    evm_user: &EthermintUserKey,
    base: EthAddress,
    quote: EthAddress,
    pool_idx: Uint256,
) {
    if base != EthAddress::default() {
        web3.erc20_approve(
            base,
            Uint256::max_value(), // bad practice but it's a test env so we don't care
            evm_user.eth_privkey,
            dex,
            Some(OPERATION_TIMEOUT),
            vec![],
        )
        .await
        .expect("Could not approve base token");
    }
    web3.erc20_approve(
        quote,
        Uint256::max_value(), // bad practice but it's a test env so we don't care
        evm_user.eth_privkey,
        dex,
        Some(OPERATION_TIMEOUT),
        vec![],
    )
    .await
    .expect("Could not approve quote token");
    let pool = croc_query_pool_params(
        web3,
        query,
        Some(evm_user.eth_address),
        base,
        quote,
        pool_idx,
    )
    .await;
    if pool.is_ok() && pool.unwrap().tick_size != 0 {
        info!("Pool already created");
    } else {
        info!("Creating {base}/{quote} pool");
        init_pool(
            web3,
            evm_user.eth_privkey,
            dex,
            base,
            quote,
            Some(pool_idx),
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
    // Make the price 1:1 by providing sqrt(1.0) * 2^64
    let price: Uint256 = (f64::sqrt(1f64) * 2f64.powf(64.0))
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
        web3.erc20_approve(
            pool_base,
            Uint256::max_value(),
            evm_privkey,
            dex,
            Some(OPERATION_TIMEOUT),
            vec![],
        )
        .await
        .expect("Could not approve base token");
    }
    web3.erc20_approve(
        pool_quote,
        Uint256::max_value(),
        evm_privkey,
        dex,
        Some(OPERATION_TIMEOUT),
        vec![],
    )
    .await
    .expect("Could not approve quote token");

    let native_in = if pool_base == EthAddress::default() {
        Some(one_eth())
    } else {
        None
    };

    dex_user_cmd(web3, dex, evm_privkey, init_pool_args, native_in, None).await
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
                        value: format!("\"{dex_contract}\""),
                    },
                    ParamChange {
                        subspace: "nativedex".to_string(),
                        key: "VerifiedCrocPolicyAddress".to_string(),
                        value: format!("\"{policy_contract}\""),
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
    info!("Gov proposal executed with {res:?}");
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
        .create_legacy_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {res:?}");
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
        .create_legacy_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {res:?}");
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
        .create_legacy_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {res:?}");
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
        .create_legacy_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {res:?}");
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
        .create_legacy_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {res:?}");
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
        .create_legacy_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {res:?}");
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
        .create_legacy_gov_proposal(
            any,
            deposit,
            fee,
            keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Gov proposal executed with {res:?}");
}
