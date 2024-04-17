use std::time::Duration;

use clarity::{abi::AbiToken, Address as EthAddress, PrivateKey, Uint256};
use num256::Int256;

use num::ToPrimitive;
use web30::{
    client::Web3,
    jsonrpc::error::Web3Error,
    types::{TransactionRequest, TransactionResponse},
};

use crate::utils::OPERATION_TIMEOUT;

// Callpath Indices
pub const BOOT_PATH: u16 = 0;
pub const HOT_PROXY: u16 = 1;
pub const WARM_PATH: u16 = 2;
pub const COLD_PATH: u16 = 3;
pub const LONG_PATH: u16 = 4;
pub const MICRO_PATHS: u16 = 5;
pub const KNOCKOUT_LIQ_PATH: u16 = 7;
pub const KNOCKOUT_FLAG_PATH: u16 = 3500;
pub const SAFE_MODE_PATH: u16 = 9999;

lazy_static! {
    pub static ref MIN_TICK: Int256 = Int256::from(-665454i64);
    pub static ref MAX_TICK: Int256 = Int256::from(831818i64);
    pub static ref MIN_PRICE: Uint256 = Uint256::from(65538u32);
    pub static ref MAX_PRICE: Uint256 = Uint256::from(21267430153580247136652501917186561137u128);
}

// ABI result parsing notes:
// a struct with static fields is returned like a static tuple, (static fields have no tail)
// enc(X1, ..., Xk) = head(X(1)) ... head(X(k)) tail(X(1)) ... tail(X(k)) = enc(X1) ... enc(Xk)
// uint<M>: enc(X) is the big-endian encoding of X, padded on the higher-order (left) side with zero-bytes such that the length is 32 bytes.

// ==========================================================================================================================================
//                                                   AMBIENT DEX CONVENIENCE FUNCTIONS
// ==========================================================================================================================================

// CrocQuery functions
pub async fn croc_query_dex(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
) -> Result<EthAddress, Web3Error> {
    let caller = caller.unwrap_or(croc_query_contract);
    // ABI: address immutable public dex_;
    let payload = clarity::abi::encode_call("dex_()", &[])?;
    let dex_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;
    let dex_address = EthAddress::from_slice(match dex_res.get(12..32) {
        Some(val) => val,
        None => {
            return Err(Web3Error::ContractCallError(
                "croc_query dex_() failed".to_string(),
            ))
        }
    })?;

    Ok(dex_address)
}
#[derive(Debug, Clone)]
pub struct CrocQueryCurveState {
    pub price_root: Uint256,
    pub ambient_seeds: Uint256,
    pub conc_liq: Uint256,
    pub seed_deflator: Uint256,
    pub conc_growth: Uint256,
}
pub async fn croc_query_curve(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    base: EthAddress,  // The base token, must be lexically smaller than quote
    quote: EthAddress, // The quote token, must be lexically larger than base
    pool_idx: Uint256, // The index of the pool's template
) -> Result<CrocQueryCurveState, Web3Error> {
    if base.gt(&quote) {
        return Err(Web3Error::ContractCallError(
            "croc_query_curve: base must be lexically smaller than quote".to_string(),
        ));
    }
    // ABI: queryCurve (address base, address quote, uint256 poolIdx) returns (CurveMath.CurveState memory curve)
    // struct CurveState { uint128 priceRoot_; uint128 ambientSeeds_; uint128 concLiq_; uint64 seedDeflator_; uint64 concGrowth_; }
    let caller = caller.unwrap_or(croc_query_contract);
    let payload = clarity::abi::encode_call(
        "queryCurve(address,address,uint256)",
        &[base.into(), quote.into(), pool_idx.into()],
    )?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;

    // Parse the results:

    let mut i: usize = 0;
    let price_root = Uint256::from_be_bytes(&query_res[i..i + 32]);
    i += 32;
    let ambient_seeds = Uint256::from_be_bytes(&query_res[i..i + 32]);
    i += 32;
    let conc_liq = Uint256::from_be_bytes(&query_res[i..i + 32]);
    i += 32;
    let seed_deflator = Uint256::from_be_bytes(&query_res[i..i + 32]);
    i += 32;
    let conc_growth = Uint256::from_be_bytes(&query_res[i..i + 32]);

    Ok(CrocQueryCurveState {
        price_root,
        ambient_seeds,
        conc_liq,
        seed_deflator,
        conc_growth,
    })
}

#[derive(Debug, Clone)]
pub struct CrocQueryPoolParams {
    pub schema: u8,
    pub fee_rate: u16,
    pub protocol_take: u8,
    pub tick_size: u16,
    pub jit_thresh: u8,
    pub knockout_bits: u8,
    pub oracle_flags: u8,
}
pub async fn croc_query_pool_params(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    base: EthAddress,  // The base token, must be lexically smaller than quote
    quote: EthAddress, // The quote token, must be lexically larger than base
    pool_idx: Uint256, // The index of the pool's template
) -> Result<CrocQueryPoolParams, Web3Error> {
    if base.gt(&quote) {
        return Err(Web3Error::ContractCallError(
            "croc_query_pool_params: base must be lexically smaller than quote".to_string(),
        ));
    }
    // ABI: queryPoolParams (address base, address quote, uint256 poolIdx) returns (PoolSpecs.Pool memory pool)
    // struct Pool { uint8 schema_; uint16 feeRate_; uint8 protocolTake_; uint16 tickSize_; uint8 jitThresh_; uint8 knockoutBits_; uint8 oracleFlags_; }

    let caller = caller.unwrap_or(croc_query_contract);
    let payload = clarity::abi::encode_call(
        "queryPoolParams(address,address,uint256)",
        &[base.into(), quote.into(), pool_idx.into()],
    )?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;

    Ok(parse_croc_query_pool_params(query_res))
}
fn parse_croc_query_pool_params(query_res: Vec<u8>) -> CrocQueryPoolParams {
    let mut i: usize = 0;
    let schema = query_res[i + 31];
    i += 32;
    let fee_rate = u16::from_be_bytes([query_res[i + 31 - 1], query_res[i + 31]]);
    i += 32;
    let protocol_take = query_res[i + 31];
    i += 32;
    let tick_size = u16::from_be_bytes([query_res[i + 31 - 1], query_res[i + 31]]);
    i += 32;
    let jit_thresh = query_res[i + 31];
    i += 32;
    let knockout_bits = query_res[i + 31];
    i += 32;
    let oracle_flags = query_res[i + 31];

    CrocQueryPoolParams {
        schema,
        fee_rate,
        protocol_take,
        tick_size,
        jit_thresh,
        knockout_bits,
        oracle_flags,
    }
}

pub async fn croc_query_pool_template(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    pool_idx: Uint256, // The index of the pool's template
) -> Result<CrocQueryPoolParams, Web3Error> {
    // ABI: queryPoolTemplate (uint256 poolIdx) returns (PoolSpecs.Pool memory pool)
    // struct Pool { uint8 schema_; uint16 feeRate_; uint8 protocolTake_; uint16 tickSize_; uint8 jitThresh_; uint8 knockoutBits_; uint8 oracleFlags_; }
    let caller = caller.unwrap_or(croc_query_contract);
    let payload = clarity::abi::encode_call("queryPoolTemplate(uint256)", &[pool_idx.into()])?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;

    Ok(parse_croc_query_pool_params(query_res))
}

pub async fn croc_query_curve_tick(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    base: EthAddress,  // The base token, must be lexically smaller than quote
    quote: EthAddress, // The quote token, must be lexically larger than base
    pool_idx: Uint256, // The index of the pool's template
) -> Result<Int256, Web3Error> {
    if base.gt(&quote) {
        return Err(Web3Error::ContractCallError(
            "croc_query_curve_tick: base must be lexically smaller than quote".to_string(),
        ));
    }
    // ABI: queryCurveTick (address base, address quote, uint256 poolIdx) returns (int24)
    let caller = caller.unwrap_or(croc_query_contract);
    let payload = clarity::abi::encode_call(
        "queryCurveTick(address,address,uint256)",
        &[base.into(), quote.into(), pool_idx.into()],
    )?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;

    Ok(Int256::from_be_bytes(&query_res))
}

pub async fn croc_query_liquidity(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    base: EthAddress,  // The base token, must be lexically smaller than quote
    quote: EthAddress, // The quote token, must be lexically larger than base
    pool_idx: Uint256, // The index of the pool's template
) -> Result<Uint256, Web3Error> {
    if base.gt(&quote) {
        return Err(Web3Error::ContractCallError(
            "croc_query_liquidity: base must be lexically smaller than quote".to_string(),
        ));
    }

    // ABI: queryLiquidity (address base, address quote, uint256 poolIdx) returns (uint128)
    let caller = caller.unwrap_or(croc_query_contract);
    let payload = clarity::abi::encode_call(
        "queryLiquidity(address,address,uint256)",
        &[base.into(), quote.into(), pool_idx.into()],
    )?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;

    Ok(Uint256::from_be_bytes(&query_res))
}

pub async fn croc_query_price(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    base: EthAddress,  // The base token, must be lexically smaller than quote
    quote: EthAddress, // The quote token, must be lexically larger than base
    pool_idx: Uint256, // The index of the pool's template
) -> Result<Uint256, Web3Error> {
    if base.gt(&quote) {
        return Err(Web3Error::ContractCallError(
            "croc_query_price: base must be lexically smaller than quote".to_string(),
        ));
    }

    // ABI: queryPrice (address base, address quote, uint256 poolIdx) returns (uint128)
    let caller = caller.unwrap_or(croc_query_contract);
    let payload = clarity::abi::encode_call(
        "queryPrice(address,address,uint256)",
        &[base.into(), quote.into(), pool_idx.into()],
    )?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;

    Ok(Uint256::from_be_bytes(&query_res))
}

#[derive(Debug, Clone)]
pub struct CrocQueryRangePosition {
    pub liq: Uint256,
    pub fee: u64,
    pub timestamp: u32,
    pub atomic: bool,
}

#[allow(clippy::too_many_arguments)]
pub async fn croc_query_range_position(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    owner: EthAddress,
    base: EthAddress,   // The base token, must be lexically smaller than quote
    quote: EthAddress,  // The quote token, must be lexically larger than base
    pool_idx: Uint256,  // The index of the pool's template
    lower_tick: Int256, // The lower tick boundary of the position
    upper_tick: Int256, // The lower tick boundary of the position
) -> Result<CrocQueryRangePosition, Web3Error> {
    if base.gt(&quote) {
        return Err(Web3Error::ContractCallError(
            "croc_query_range_position: base must be lexically smaller than quote".to_string(),
        ));
    }

    // ABI: queryRangePosition (address owner, address base, address quote, uint256 poolIdx, int24 lowerTick, int24 upperTick)
    // returns (uint128 liq, uint64 fee, uint32 timestamp, bool atomic)
    let caller = caller.unwrap_or(owner);
    // TODO: This probably won't work until Clarity is updated with Int256 support
    let payload = clarity::abi::encode_call(
        "queryRangePosition(address,address,address,uint256,int24,int24)",
        &[
            owner.into(),
            base.into(),
            quote.into(),
            pool_idx.into(),
            lower_tick.into(),
            upper_tick.into(),
        ],
    )?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;

    let mut i: usize = 0;
    let liq = Uint256::from_be_bytes(&query_res[i..i + 32]);
    i += 32;
    let fee = Uint256::from_be_bytes(&query_res[i..i + 32])
        .to_u64()
        .unwrap();
    i += 32;
    let timestamp = Uint256::from_be_bytes(&query_res[i..i + 32])
        .to_u32()
        .unwrap();
    i += 32;
    let atomic = query_res[i] > 0u8;

    Ok(CrocQueryRangePosition {
        liq,
        fee,
        timestamp,
        atomic,
    })
}

#[derive(Debug, Clone)]
pub struct CrocQueryAmbientPosition {
    pub seeds: Uint256,
    pub timestamp: u32,
}
#[allow(clippy::too_many_arguments)]
pub async fn croc_query_ambient_position(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    owner: EthAddress,
    base: EthAddress,  // The base token, must be lexically smaller than quote
    quote: EthAddress, // The quote token, must be lexically larger than base
    pool_idx: Uint256, // The index of the pool's template
) -> Result<CrocQueryAmbientPosition, Web3Error> {
    if base.gt(&quote) {
        return Err(Web3Error::ContractCallError(
            "croc_query_ambient_position: base must be lexically smaller than quote".to_string(),
        ));
    }

    // ABI: queryAmbientPosition (address owner, address base, address quote, uint256 poolIdx)
    // returns (uint128 seeds, uint32 timestamp) {
    let caller = caller.unwrap_or(owner);
    // TODO: This probably won't work until Clarity is updated with Int256 support
    let payload = clarity::abi::encode_call(
        "queryAmbientPosition(address,address,address,uint256)",
        &[owner.into(), base.into(), quote.into(), pool_idx.into()],
    )?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;

    let mut i: usize = 0;
    let seeds = Uint256::from_be_bytes(&query_res[i..i + 32]);
    i += 32;
    let timestamp = Uint256::from_be_bytes(&query_res[i..i + 32])
        .to_u32()
        .unwrap();

    Ok(CrocQueryAmbientPosition { seeds, timestamp })
}

#[derive(Debug, Clone)]
pub struct CrocQueryConcRewards {
    pub liq_rewards: Uint256,
    pub base_rewards: Uint256,
    pub quote_rewards: Uint256,
}
#[allow(clippy::too_many_arguments)]
pub async fn croc_query_conc_rewards(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    owner: EthAddress,
    base: EthAddress,   // The base token, must be lexically smaller than quote
    quote: EthAddress,  // The quote token, must be lexically larger than base
    pool_idx: Uint256,  // The index of the pool's template
    lower_tick: Int256, // The lower tick boundary of the position
    upper_tick: Int256, // The lower tick boundary of the position
) -> Result<CrocQueryConcRewards, Web3Error> {
    if base.gt(&quote) {
        return Err(Web3Error::ContractCallError(
            "croc_query_conc_rewards: base must be lexically smaller than quote".to_string(),
        ));
    }

    // ABI: queryConcRewards (address owner, address base, address quote, uint256 poolIdx, int24 lowerTick, int24 upperTick)
    // returns (uint128 liqRewards, uint128 baseRewards, uint128 quoteRewards)
    let caller = caller.unwrap_or(owner);
    // TODO: This probably won't work until Clarity is updated with Int256 support
    let payload = clarity::abi::encode_call(
        "queryConcRewards(address,address,address,uint256,int24,int24)",
        &[
            owner.into(),
            base.into(),
            quote.into(),
            pool_idx.into(),
            lower_tick.into(),
            upper_tick.into(),
        ],
    )?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;

    let mut i: usize = 0;
    let liq_rewards = Uint256::from_be_bytes(&query_res[i..i + 32]);
    i += 32;
    let base_rewards = Uint256::from_be_bytes(&query_res[i..i + 32]);
    i += 32;
    let quote_rewards = Uint256::from_be_bytes(&query_res[i..i + 32]);

    Ok(CrocQueryConcRewards {
        liq_rewards,
        base_rewards,
        quote_rewards,
    })
}

/// Specifies a swap() call on the DEX contract
#[derive(Debug, Clone)]
pub struct SwapArgs {
    pub base: EthAddress,
    pub quote: EthAddress,
    pub pool_idx: Uint256,
    pub is_buy: bool,
    pub in_base_qty: bool,
    pub qty: Uint256,
    pub tip: u16,
    pub limit_price: Uint256,
    pub min_out: Uint256,
    pub reserve_flags: u8,
}

/// Perform a swap on `dex_contract` using funds owned by `wallet`
/// Note: it is possible to use the native token in Ambient pools, so `native_in` will be used as the value for the transaction, if provided
/// Warning: It is unfortunately impossible to read the result of the swap without a wrapper contract because of Ethereum limitations
/// and because Ambient does not emit any events for swaps
pub async fn dex_swap(
    web30: &Web3,
    dex_contract: EthAddress,
    wallet: PrivateKey,
    swap_args: SwapArgs,
    native_in: Option<Uint256>,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let timeout = timeout.unwrap_or(OPERATION_TIMEOUT);
    if swap_args.base.gt(&swap_args.quote) {
        return Err(Web3Error::ContractCallError(
            "dex_swap: base must be lexically smaller than quote".to_string(),
        ));
    }
    // ABI: swap (address base, address quote, uint256 poolIdx, bool isBuy, bool inBaseQty, uint128 qty, uint16 tip, uint128 limitPrice, uint128 minOut, uint8 reserveFlags)
    // returns (int128 baseQuote, int128 quoteFlow) {
    let payload = clarity::abi::encode_call(
        "swap(address,address,uint256,bool,bool,uint128,uint16,uint128,uint128,uint8)",
        &[
            swap_args.base.into(),
            swap_args.quote.into(),
            swap_args.pool_idx.into(),
            swap_args.is_buy.into(),
            swap_args.in_base_qty.into(),
            swap_args.qty.into(),
            swap_args.tip.into(),
            swap_args.limit_price.into(),
            swap_args.min_out.into(),
            swap_args.reserve_flags.into(),
        ],
    )?;
    let native_in = native_in.unwrap_or(0u8.into());
    let txhash = web30
        .send_prepared_transaction(
            web30
                .prepare_transaction(dex_contract, payload, native_in, wallet, vec![])
                .await?,
        )
        .await?;
    web30.wait_for_transaction(txhash, timeout, None).await
}

/// Specifies a userCmd call to be made on the DEX contract
#[derive(Debug, Clone)]
pub struct UserCmdArgs {
    pub callpath: u16,
    pub cmd: Vec<AbiToken>,
}

/// Calls any userCmd on the DEX contract
pub async fn dex_user_cmd(
    web30: &Web3,
    dex_contract: EthAddress,
    wallet: PrivateKey,
    cmd_args: UserCmdArgs,
    native_in: Option<Uint256>,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let timeout = timeout.unwrap_or(OPERATION_TIMEOUT);
    // ABI: userCmd (uint16 callpath, bytes calldata cmd) returns (bytes memory)

    let cmd = clarity::abi::encode_tokens(&cmd_args.cmd);
    let payload = clarity::abi::encode_call(
        "userCmd(uint16,bytes)",
        &[cmd_args.callpath.into(), cmd.into()],
    )?;
    let native_in = native_in.unwrap_or(0u8.into());
    let txhash = web30
        .send_prepared_transaction(
            web30
                .prepare_transaction(dex_contract, payload, native_in, wallet, vec![])
                .await?,
        )
        .await?;
    web30.wait_for_transaction(txhash, timeout, None).await
}

/// Specifies a protocolCmd call to be made on the DEX contract
/// If CrocPolicy is set up then this call will fail
#[derive(Debug, Clone)]
pub struct ProtocolCmdArgs {
    pub callpath: u16,
    pub cmd: Vec<AbiToken>,
    pub sudo: bool,
}
/// Calls any protocolCmd on the DEX contract
pub async fn dex_protocol_cmd(
    web30: &Web3,
    dex_contract: EthAddress,
    wallet: PrivateKey,
    cmd_args: ProtocolCmdArgs,
    native_in: Option<Uint256>,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let timeout = timeout.unwrap_or(OPERATION_TIMEOUT);
    // ABI: protocolCmd (uint16 callpath, bytes calldata cmd, bool sudo)

    let cmd = clarity::abi::encode_tokens(&cmd_args.cmd);
    let payload = clarity::abi::encode_call(
        "protocolCmd(uint16,bytes,bool)",
        &[cmd_args.callpath.into(), cmd.into(), cmd_args.sudo.into()],
    )?;
    let native_in = native_in.unwrap_or(0u8.into());
    let txhash = web30
        .send_prepared_transaction(
            web30
                .prepare_transaction(dex_contract, payload, native_in, wallet, vec![])
                .await?,
        )
        .await?;
    web30.wait_for_transaction(txhash, timeout, None).await
}

/// Conveniently wraps dex_protocol_cmd() to invoke an authority transfer, useful when setting up CrocPolicy
pub async fn dex_authority_transfer(
    web30: &Web3,
    dex_contract: EthAddress,
    new_auth: EthAddress,
    wallet: PrivateKey,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let code: Uint256 = 20u8.into();

    // ABI: authorityTransfer (uint8 cmd_code, address auth)
    let cmd_args = vec![code.into(), new_auth.into()];

    dex_protocol_cmd(
        web30,
        dex_contract,
        wallet,
        ProtocolCmdArgs {
            callpath: COLD_PATH,
            cmd: cmd_args,
            sudo: true,
        },
        None,
        timeout,
    )
    .await
}

pub async fn dex_query_safe_mode(
    web30: &Web3,
    dex_contract: EthAddress,
    caller: Option<EthAddress>,
) -> Result<bool, Web3Error> {
    // ABI: bool internal inSafeMode_
    let caller = caller.unwrap_or(dex_contract);
    let payload = clarity::abi::encode_call("safeMode()", &[])?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, dex_contract, payload),
            None,
        )
        .await?;

    Ok(query_res[31] > 0u8)
}

pub async fn dex_query_authority(
    web30: &Web3,
    dex_contract: EthAddress,
    caller: Option<EthAddress>,
) -> Result<EthAddress, Web3Error> {
    // ABI: bool internal inSafeMode_
    let caller = caller.unwrap_or(dex_contract);
    let payload = clarity::abi::encode_call("authority()", &[])?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, dex_contract, payload),
            None,
        )
        .await?;

    Ok(EthAddress::from_slice(&query_res[12..32])?)
}
