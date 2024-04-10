//! Contains usage tests for the Ambient (CrocSwap) dex deployment

use crate::bootstrapping::DexAddresses;
use crate::dex_utils::{
    croc_query_curve_tick, croc_query_dex, croc_query_pool_params, croc_query_range_position,
    dex_user_cmd, UserCmdArgs,
};
use crate::utils::{EthermintUserKey, ValidatorKeys, OPERATION_TIMEOUT};
use clarity::{Address as EthAddress, Uint256};
use deep_space::Contact;
use num::ToPrimitive;
use web30::client::Web3;
use web30::jsonrpc::error::Web3Error;
use web30::types::TransactionResponse;

lazy_static! {
    pub static ref POOL_IDX: Uint256 = 36000u32.into();
}

pub async fn dex_test(
    _contact: &Contact,
    web3: &Web3,
    _validator_keys: Vec<ValidatorKeys>,
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

    let tokens = erc20_contracts[0..2].to_vec();
    let (pool_base, pool_quote) = if tokens[0] < tokens[1] {
        (tokens[0], tokens[1])
    } else {
        (tokens[1], tokens[0])
    };

    let pool = croc_query_pool_params(
        web3,
        dex_contracts.query,
        Some(evm_user.eth_address),
        pool_base,
        pool_quote,
        *POOL_IDX,
    )
    .await;
    if pool.is_ok() && pool.unwrap().tick_size != 0 {
        info!("Pool already created");
    } else {
        info!("Creating pool");
        init_pool(web3, evm_user, dex_contracts.dex, pool_base, pool_quote)
            .await
            .expect("Could not create pool");
    }

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

    let mint_ranged_pos_args = UserCmdArgs {
        callpath: 2, // Warm Path index
        cmd: vec![
            Uint256::from(11u8).into(),   // Mint Ranged Liq in base token code
            pool_base.into(),             // base
            pool_quote.into(),            // quote
            (*POOL_IDX).into(),           // poolIdx
            bid_tick.into(),              // bid (lower) tick
            ask_tick.into(),              // ask (upper) tick
            1_000_000u128.into(),         // liq (in base token)
            18446744073u128.into(),       // limitLower
            18446744073709000u128.into(), // limitHigher
            Uint256::from(0u8).into(),    // reserveFlags
            EthAddress::default().into(), // lpConduit
        ],
    };
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
    info!("Range position: {:?}", range_pos);

    info!("Successfully minted position");
}

pub async fn init_pool(
    web3: &Web3,
    evm_user: &EthermintUserKey,
    dex: EthAddress,
    pool_base: EthAddress,
    pool_quote: EthAddress,
) -> Result<TransactionResponse, Web3Error> {
    if pool_base >= pool_quote {
        panic!("Base token must be lexically less than quote token");
    }
    let evm_privkey = evm_user.eth_privkey;

    let price: Uint256 = (f64::sqrt(10f64.powf(-12.0)) * 2f64.powf(64.0))
        .round()
        .to_u128()
        .unwrap()
        .into();
    let init_pool_args = UserCmdArgs {
        callpath: 3, // Cold Path index
        cmd: vec![
            71u8.into(), // Init pool code
            pool_base.into(),
            pool_quote.into(),
            (*POOL_IDX).into(),
            price.into(),
        ],
    };
    web3.approve_erc20_transfers(pool_base, evm_privkey, dex, Some(OPERATION_TIMEOUT), vec![])
        .await
        .expect("Could not approve base token");
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
