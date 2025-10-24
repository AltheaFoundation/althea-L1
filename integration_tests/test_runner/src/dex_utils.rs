use std::{convert::TryInto, str::FromStr, time::Duration};

use clarity::{
    abi::{encode_tokens, AbiToken},
    Address as EthAddress, PrivateKey, Uint256,
};
use num256::Int256;

use num::{ToPrimitive, Zero};
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

/// This query returns the stored ranged position information, which is not an accurate reflection of
/// the tokens that the position represents. Calling croc_query_ranged_tokens may be more useful since
/// it will return the token amounts which would be paid out on burning BUT NOT THE ACCUMULATED FEE REWARDS
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
pub struct CrocQueryTokens {
    pub liq: Uint256,
    pub base_qty: Uint256,
    pub quote_qty: Uint256,
}

/// This query returns the liquidity and tokens that the ranged position represents.
/// If you instead need the stored position information, use croc_query_range_position
#[allow(clippy::too_many_arguments)]
pub async fn croc_query_range_tokens(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    owner: EthAddress,
    base: EthAddress,   // The base token, must be lexically smaller than quote
    quote: EthAddress,  // The quote token, must be lexically larger than base
    pool_idx: Uint256,  // The index of the pool's template
    lower_tick: Int256, // The lower tick boundary of the position
    upper_tick: Int256, // The lower tick boundary of the position
) -> Result<CrocQueryTokens, Web3Error> {
    if base.gt(&quote) {
        return Err(Web3Error::ContractCallError(
            "croc_query_range_tokens: base must be lexically smaller than quote".to_string(),
        ));
    }

    // ABI: queryRangeTokens(address owner, address base, address quote, uint256 poolIdx, int24 lowerTick, int24 upperTick)
    // returns (uint128 liq, uint128 baseQty, uint128 quoteQty)
    let caller = caller.unwrap_or(owner);
    let payload = clarity::abi::encode_call(
        "queryRangeTokens(address,address,address,uint256,int24,int24)",
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
    let base_qty = Uint256::from_be_bytes(&query_res[i..i + 32]);
    i += 32;
    let quote_qty = Uint256::from_be_bytes(&query_res[i..i + 32]);

    Ok(CrocQueryTokens {
        liq,
        base_qty,
        quote_qty,
    })
}

#[derive(Debug, Clone)]
pub struct CrocQueryKnockoutPivot {
    pub lots: Uint256, // Multiply by 1024 to get the amount of sqrt(X*Y) liquidity at the pivot
    pub pivot: u32,    // The pivot time used in later referring to the knockout pivot
    pub range: u16,    // The width of the knockout range liquidity in ticks at the pivot
}

#[allow(clippy::too_many_arguments)]
pub async fn croc_query_knockout_pivot(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: EthAddress,
    base: EthAddress,  // The base token, must be lexically smaller than quote
    quote: EthAddress, // The quote token, must be lexically larger than base
    pool_idx: Uint256, // The index of the pool's template
    is_bid: bool, // If true then the liquidity knocks out when the price moves below tick, otherwise above
    tick: Int256, // The tick of the knockout pivot
) -> Result<CrocQueryKnockoutPivot, Web3Error> {
    if base.gt(&quote) {
        return Err(Web3Error::ContractCallError(
            "croc_query_knockout_pivot: base must be lexically smaller than quote".to_string(),
        ));
    }

    // ABI: queryKnockoutPivot(address base, address quote, uint256 poolIdx, bool isBid, int24 tick)
    // returns (uint96 lots, uint32 pivot, uint16 range)
    let payload = clarity::abi::encode_call(
        "queryKnockoutPivot(address,address,uint256,bool,int24)",
        &[
            base.into(),
            quote.into(),
            pool_idx.into(),
            is_bid.into(),
            tick.into(),
        ],
    )?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;

    let mut i: usize = 0;
    let lots = Uint256::from_be_bytes(&query_res[i..i + 32]);
    i += 32;
    let pivot = u32::from_be_bytes(query_res[i + 28..i + 32].try_into().unwrap());
    i += 32;
    let range = u16::from_be_bytes(query_res[i + 30..i + 32].try_into().unwrap());

    Ok(CrocQueryKnockoutPivot { lots, pivot, range })
}

#[derive(Debug, Clone)]
pub struct CrocQueryKnockoutTokens {
    pub liq: Uint256,
    pub base_qty: Uint256,
    pub quote_qty: Uint256,
    pub knocked_out: bool,
}

#[allow(clippy::too_many_arguments)]
pub async fn croc_query_knockout_tokens(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    owner: EthAddress,
    base: EthAddress,   // The base token, must be lexically smaller than quote
    quote: EthAddress,  // The quote token, must be lexically larger than base
    pool_idx: Uint256,  // The index of the pool's template
    pivot: u32, // The block timestamp associated with the minting of the knockout position (cast to a u32, which will allegedly work until the year 2106)
    lower_tick: Int256, // The lower tick boundary of the position
    upper_tick: Int256, // The lower tick boundary of the position
    is_bid: bool,
) -> Result<CrocQueryKnockoutTokens, Web3Error> {
    if base.gt(&quote) {
        return Err(Web3Error::ContractCallError(
            "croc_query_knockout_tokens: base must be lexically smaller than quote".to_string(),
        ));
    }

    // ABI: queryKnockoutTokens (address owner, address base, address quote, uint256 poolIdx, uint32 pivot, bool isBid, int24 lowerTick, int24 upperTick)
    // returns (uint128 liq, uint128 baseQty, uint128 quoteQty, bool knockedOut)
    let caller = caller.unwrap_or(owner);
    let payload = clarity::abi::encode_call(
        "queryKnockoutTokens(address,address,address,uint256,uint32,bool,int24,int24)",
        &[
            owner.into(),
            base.into(),
            quote.into(),
            pool_idx.into(),
            pivot.into(),
            is_bid.into(),
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
    let base_qty = Uint256::from_be_bytes(&query_res[i..i + 32]);
    i += 32;
    let quote_qty = Uint256::from_be_bytes(&query_res[i..i + 32]);
    let knocked_out = query_res.last().unwrap() > &0u8;

    Ok(CrocQueryKnockoutTokens {
        liq,
        base_qty,
        quote_qty,
        knocked_out,
    })
}

#[derive(Debug, Clone)]
pub struct CrocQueryAmbientPosition {
    pub seeds: Uint256,
    pub timestamp: u32,
}

/// This query returns the stored ambient position information, which is not an accurate reflection of
/// the tokens that the position represents. Calling croc_query_ambient_tokens may be more useful since
/// it will return the "inflated" liquidity as the base and quote tokens which would be paid out on burning.
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

/// This query returns the liquidity and tokens that the ambient position represents.
/// If you instead need the stored position information, use croc_query_ambient_position
#[allow(clippy::too_many_arguments)]
pub async fn croc_query_ambient_tokens(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    owner: EthAddress,
    base: EthAddress,  // The base token, must be lexically smaller than quote
    quote: EthAddress, // The quote token, must be lexically larger than base
    pool_idx: Uint256, // The index of the pool's template
) -> Result<CrocQueryTokens, Web3Error> {
    if base.gt(&quote) {
        return Err(Web3Error::ContractCallError(
            "croc_query_ambient_tokens: base must be lexically smaller than quote".to_string(),
        ));
    }

    // ABI: queryAmbientTokens(address owner, address base, address quote, uint256 poolIdx)
    // returns (uint128 liq, uint128 baseQty, uint128 quoteQty)
    let caller = caller.unwrap_or(owner);
    let payload = clarity::abi::encode_call(
        "queryAmbientTokens(address,address,address,uint256)",
        &[owner.into(), base.into(), quote.into(), pool_idx.into()],
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
    let base_qty = Uint256::from_be_bytes(&query_res[i..i + 32]);
    i += 32;
    let quote_qty = Uint256::from_be_bytes(&query_res[i..i + 32]);

    Ok(CrocQueryTokens {
        liq,
        base_qty,
        quote_qty,
    })
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

#[derive(Debug, Clone)]
pub struct UserBalance {
    pub surplus_collateral: u128,
    pub nonce: u32,
    pub agent_calls_left: u32,
}
#[allow(clippy::too_many_arguments)]
pub async fn croc_query_nonce(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: Option<EthAddress>,
    client: EthAddress,
    salt: Vec<u8>, // The arbitrary salt bytes used for multidimensional nonces
) -> Result<UserBalance, Web3Error> {
    // ABI: queryRelayNonce (address client, bytes32 nonceSalt)
    // returns (uint128 surplus, uint32 nonce, uint32 agentCallsLeft)
    let caller = caller.unwrap_or(client);
    let payload = clarity::abi::encode_call(
        "queryRelayNonce(address,bytes32)",
        &[client.into(), salt.into()],
    )?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;

    let mut i: usize = 8; // Left are 8 bytes unused
                          // The next 16 hold the surplus collateral
    let surplus_collateral = u128::from_be_bytes(query_res[i..i + 16].try_into().unwrap());
    i += 16;
    // The next 4 hold nonce
    let nonce = u32::from_be_bytes(query_res[i..i + 4].try_into().unwrap());
    i += 4;
    // Remaining 4 hold agent_calls_left
    let agent_calls_left = u32::from_be_bytes(query_res[i..i + 4].try_into().unwrap());

    Ok(UserBalance {
        surplus_collateral,
        nonce,
        agent_calls_left,
    })
}

#[derive(Debug, Clone)]
pub struct LiquidityCurveLevel {
    pub bid_lots: Uint256,
    pub ask_lots: Uint256,
    pub odometer: Uint256,
}
#[allow(clippy::too_many_arguments)]
pub async fn croc_query_level(
    web30: &Web3,
    croc_query_contract: EthAddress,
    caller: EthAddress,
    base: EthAddress,
    quote: EthAddress,
    pool_idx: Uint256,
    tick: Int256,
) -> Result<LiquidityCurveLevel, Web3Error> {
    // ABI: queryLevel (address base, address quote, uint256 poolIdx, int24 tick)
    // returns (uint96 bidLots, uint96 askLots, uint64 odometer)
    let payload = clarity::abi::encode_call(
        "queryLevel(address,address,uint256,int24)",
        &[base.into(), quote.into(), pool_idx.into(), tick.into()],
    )?;

    let query_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, croc_query_contract, payload),
            None,
        )
        .await?;

    let mut i: usize = 0;
    let bid_lots = Uint256::from_be_bytes(query_res[i..i + 32].try_into().unwrap());
    i += 32;
    let ask_lots = Uint256::from_be_bytes(query_res[i..i + 32].try_into().unwrap());
    i += 32;
    let odometer = Uint256::from_be_bytes(query_res[i..i + 32].try_into().unwrap());

    Ok(LiquidityCurveLevel {
        bid_lots,
        ask_lots,
        odometer,
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
/// NOTE: if CrocPolicy is set up, this call will fail and CrocPolicy must be used instead
pub async fn dex_direct_protocol_cmd(
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

    dex_direct_protocol_cmd(
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

/// Specifies an opsResolution call to be made on the CrocPolicy contract
#[derive(Debug, Clone)]
pub struct OpsResolutionArgs {
    pub minion: EthAddress, // Use the DEX address
    pub callpath: u16,
    pub cmd: Vec<AbiToken>,
    // sudo: bool, // Ops resolutions are not allowed to invoke sudo commands
}
/// Calls a non-sudo protocolCmd via CrocPolicy
/// NOTE: CrocPolicy MUST be the DEX `authority_` for this to work, and the wallet must be one of the Ops/Treasury/Emergency roles
pub async fn croc_policy_ops_resolution(
    web30: &Web3,
    croc_policy_contract: EthAddress,
    wallet: PrivateKey,
    cmd_args: OpsResolutionArgs,
    native_in: Option<Uint256>,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let timeout = timeout.unwrap_or(OPERATION_TIMEOUT);
    // ABI: opsResolution (address minion, uint16 proxyPath, bytes cmd)

    let cmd = clarity::abi::encode_tokens(&cmd_args.cmd);
    let payload = clarity::abi::encode_call(
        "opsResolution(address,uint16,bytes)",
        &[cmd_args.minion.into(), cmd_args.callpath.into(), cmd.into()],
    )?;
    let native_in = native_in.unwrap_or(0u8.into());
    let txhash = web30
        .send_prepared_transaction(
            web30
                .prepare_transaction(croc_policy_contract, payload, native_in, wallet, vec![])
                .await?,
        )
        .await?;
    web30.wait_for_transaction(txhash, timeout, None).await
}

/// Executes a treasuryResolution directly on the CrocPolicy contract (i.e. not using nativedex governance)
/// NOTE: CrocPolicy MUST be the DEX `authority_` for this to work, and the wallet must be the Treasury role
pub async fn croc_policy_treasury_resolution(
    web30: &Web3,
    croc_policy_contract: EthAddress,
    dex_contract: EthAddress,
    wallet: PrivateKey,
    cmd_args: ProtocolCmdArgs,
    native_in: Option<Uint256>,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let timeout = timeout.unwrap_or(OPERATION_TIMEOUT);
    // ABI: treasuryResolution (address minion, uint16 proxyPath, bytes cmd, bool sudo)

    let cmd = clarity::abi::encode_tokens(&cmd_args.cmd);
    let payload = clarity::abi::encode_call(
        "treasuryResolution(address,uint16,bytes,bool)",
        &[
            dex_contract.into(),
            cmd_args.callpath.into(),
            cmd.into(),
            cmd_args.sudo.into(),
        ],
    )?;
    let native_in = native_in.unwrap_or(0u8.into());
    let txhash = web30
        .send_prepared_transaction(
            web30
                .prepare_transaction(croc_policy_contract, payload, native_in, wallet, vec![])
                .await?,
        )
        .await?;
    web30.wait_for_transaction(txhash, timeout, None).await
}

/// Transfers the control of the CrocPolicy contract to a new set of governance addresses
pub async fn croc_policy_transfer_governance(
    web30: &Web3,
    croc_policy_contract: EthAddress,
    wallet: PrivateKey,
    ops_address: EthAddress,
    treasury_address: EthAddress,
    emergency_address: EthAddress,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let timeout = timeout.unwrap_or(OPERATION_TIMEOUT);
    // ABI: transferGovernance (address ops, address treasury, address emergency)

    let payload = clarity::abi::encode_call(
        "transferGovernance(address,address,address)",
        &[
            ops_address.into(),
            treasury_address.into(),
            emergency_address.into(),
        ],
    )?;
    let txhash = web30
        .send_prepared_transaction(
            web30
                .prepare_transaction(croc_policy_contract, payload, 0u8.into(), wallet, vec![])
                .await?,
        )
        .await?;
    web30.wait_for_transaction(txhash, timeout, None).await
}
#[allow(clippy::too_many_arguments)]
pub async fn dex_mint_ranged_pos(
    web3: &Web3,
    dex: EthAddress,
    query: EthAddress,
    evm_privkey: PrivateKey,
    evm_address: EthAddress,
    base: EthAddress,
    quote: EthAddress,
    pool_idx: Uint256,
    bid_tick: Int256,
    ask_tick: Int256,
    liq: Uint256, // The liquidity to mint (must be a multiple of 1024)
) {
    assert!(base.lt(&quote), "base must be lexically smaller than quote");
    let start_pos = croc_query_range_position(
        web3,
        query,
        None,
        evm_address,
        base,
        quote,
        pool_idx,
        bid_tick,
        ask_tick,
    )
    .await
    .expect("Could not query position");

    let mint_ranged_pos_args = UserCmdArgs {
        callpath: WARM_PATH, // Warm Path index
        cmd: vec![
            Uint256::from(1u8).into(),    // Mint Ranged Liq code
            base.into(),                  // base
            quote.into(),                 // quote
            pool_idx.into(),              // poolIdx
            bid_tick.into(),              // bid (lower) tick
            ask_tick.into(),              // ask (upper) tick
            liq.into(), // liq (in liquidity units, which must be a multiple of 1024)
            (*MIN_PRICE).into(), // limitLower
            (*MAX_PRICE).into(), // limitHigher
            Uint256::from(0u8).into(), // reserveFlags
            EthAddress::default().into(), // lpConduit
        ],
    };
    info!("Minting position in both tokens: {mint_ranged_pos_args:?}");
    dex_user_cmd(web3, dex, evm_privkey, mint_ranged_pos_args, None, None)
        .await
        .expect("Failed to mint position in pool");
    let range_pos = croc_query_range_position(
        web3,
        query,
        None,
        evm_address,
        base,
        quote,
        pool_idx,
        bid_tick,
        ask_tick,
    )
    .await
    .expect("Could not query position");
    assert_eq!(range_pos.liq - start_pos.liq, liq);
}

#[allow(clippy::too_many_arguments)]
pub async fn dex_mint_ranged_in_amount(
    web3: &Web3,
    dex: EthAddress,
    evm_privkey: PrivateKey,
    base: EthAddress,
    quote: EthAddress,
    pool_idx: Uint256,
    bid_tick: Int256,
    ask_tick: Int256,
    qty: Uint256,  // The amount of a token to mint a position with
    in_base: bool, // Whether to mint in the base token or the quote token
) {
    assert!(base.lt(&quote), "base must be lexically smaller than quote");

    let code = if in_base {
        Uint256::from(11u8) // Mint in base amount code
    } else {
        Uint256::from(12u8) // Mint in quote amount code
    };
    let native_in = if base == EthAddress::default() {
        Some(qty)
    } else {
        None
    };
    let mint_ranged_pos_args = UserCmdArgs {
        callpath: WARM_PATH, // Warm Path index
        cmd: vec![
            code.into(),
            base.into(),                  // base
            quote.into(),                 // quote
            pool_idx.into(),              // poolIdx
            bid_tick.into(),              // bid (lower) tick
            ask_tick.into(),              // ask (upper) tick
            qty.into(), // liq (in liquidity units, which must be a multiple of 1024)
            (*MIN_PRICE).into(), // limitLower
            (*MAX_PRICE).into(), // limitHigher
            Uint256::from(0u8).into(), // reserveFlags
            EthAddress::default().into(), // lpConduit
        ],
    };
    info!("Minting position in single token: {mint_ranged_pos_args:?}");
    dex_user_cmd(
        web3,
        dex,
        evm_privkey,
        mint_ranged_pos_args,
        native_in,
        None,
    )
    .await
    .expect("Failed to mint position in pool");
}

#[allow(clippy::too_many_arguments)]
pub async fn dex_burn_ranged_pos(
    web3: &Web3,
    dex: EthAddress,
    query: EthAddress,
    evm_privkey: PrivateKey,
    evm_address: EthAddress,
    base: EthAddress,
    quote: EthAddress,
    pool_idx: Uint256,
    bid_tick: Int256,
    ask_tick: Int256,
    liq: Uint256, // The liquidity to burn (must be a multiple of 1024)
) {
    assert!(base.lt(&quote), "base must be lexically smaller than quote");
    let pos_start = croc_query_range_position(
        web3,
        query,
        None,
        evm_address,
        base,
        quote,
        pool_idx,
        bid_tick,
        ask_tick,
    )
    .await
    .expect("Could not query position");

    let burn_ranged_pos_args = UserCmdArgs {
        callpath: WARM_PATH, // Warm Path index
        cmd: vec![
            Uint256::from(2u8).into(),    // Burn Ranged Liq code
            base.into(),                  // base
            quote.into(),                 // quote
            (pool_idx).into(),            // poolIdx
            bid_tick.into(),              // bid (lower) tick
            ask_tick.into(),              // ask (upper) tick
            liq.into(), // liq (in liquidity units, which must be a multiple of 1024)
            (*MIN_PRICE).into(), // limitLower
            (*MAX_PRICE).into(), // limitHigher
            Uint256::from(0u8).into(), // reserveFlags
            EthAddress::default().into(), // lpConduit
        ],
    };
    info!("Burning position: {burn_ranged_pos_args:?}");
    dex_user_cmd(web3, dex, evm_privkey, burn_ranged_pos_args, None, None)
        .await
        .expect("Failed to burn position in pool");
    let range_pos = croc_query_range_position(
        web3,
        query,
        None,
        evm_address,
        base,
        quote,
        pool_idx,
        bid_tick,
        ask_tick,
    )
    .await
    .expect("Could not query position");

    assert_eq!(pos_start.liq - range_pos.liq, liq);
}

#[allow(clippy::too_many_arguments)]
pub async fn dex_mint_knockout_pos(
    web3: &Web3,
    dex: EthAddress,
    _query: EthAddress,
    evm_privkey: PrivateKey,
    _evm_address: EthAddress,
    base: EthAddress,
    quote: EthAddress,
    pool_idx: Uint256,
    bid_tick: Int256,
    ask_tick: Int256,
    is_bid: bool,
    reserve_flags: u8, // Controls what happens with "surplus" values held by the dex
    qty: Uint256,
    inside_mid: bool,
) {
    assert!(base.lt(&quote), "base must be lexically smaller than quote");
    let arg_bytes = encode_tokens(&[qty.into(), inside_mid.into()]);
    let mint_ko_pos_args = UserCmdArgs {
        callpath: KNOCKOUT_LIQ_PATH, // KnockoutLiqPath index
        cmd: vec![
            Uint256::from(91u8).into(), // Mint Knockout code
            base.into(),                // base
            quote.into(),               // quote
            pool_idx.into(),            // poolIdx
            bid_tick.into(),            // bid (lower) tick
            ask_tick.into(),            // ask (upper) tick
            is_bid.into(),
            reserve_flags.into(),
            arg_bytes.into(),
        ],
    };
    info!("Minting knockout position: {mint_ko_pos_args:?}");
    dex_user_cmd(web3, dex, evm_privkey, mint_ko_pos_args, None, None)
        .await
        .expect("Failed to mint knockout position in pool");

    // let pivot_tick = if is_bid { bid_tick } else { ask_tick };
    // info!(
    //     "Querying knockout position: {} {} {} {} {}",
    //     base, quote, pool_idx, is_bid, pivot_tick
    // );
    // // Querying the knockout position requires the "pivotTime", which is the timestamp of the block that the position was minted in
    // let pivot = croc_query_knockout_pivot(
    //     web3,
    //     dex,
    //     evm_address,
    //     base,
    //     quote,
    //     pool_idx,
    //     is_bid,
    //     pivot_tick,
    // )
    // .await
    // .expect("Could not query pivot");
    // info!("Queried pivot: {pivot:?}");
    // let pivot = pivot.pivot;

    // let ko_pos = croc_query_knockout_tokens(
    //     web3,
    //     query,
    //     None,
    //     evm_address,
    //     base,
    //     quote,
    //     pool_idx,
    //     pivot,
    //     bid_tick,
    //     ask_tick,
    //     is_bid,
    // )
    // .await
    // .expect("Could not query position");
    // info!("Minted knockout position: {ko_pos:?}");
    // assert!(ko_pos.base_qty > 0u8.into() || ko_pos.quote_qty > 0u8.into());
}

#[allow(clippy::too_many_arguments)]
pub async fn dex_burn_knockout_pos(
    web3: &Web3,
    dex: EthAddress,
    query: EthAddress,
    evm_privkey: PrivateKey,
    evm_address: EthAddress,
    base: EthAddress,
    quote: EthAddress,
    pool_idx: Uint256,
    bid_tick: Int256,
    ask_tick: Int256,
    is_bid: bool,
    reserve_flags: u8, // Controls what happens with "surplus" values held by the dex
    qty: Uint256,
    in_liq_qty: bool,
    inside_mid: bool,
    pivot: Option<u32>,
) {
    assert!(base.lt(&quote), "base must be lexically smaller than quote");
    let start_pos = if let Some(pivot) = pivot {
        Some(
            croc_query_knockout_tokens(
                web3,
                query,
                None,
                evm_address,
                base,
                quote,
                pool_idx,
                pivot,
                bid_tick,
                ask_tick,
                is_bid,
            )
            .await
            .expect("Could not query position"),
        )
    } else {
        None
    };

    let arg_bytes = encode_tokens(&[qty.into(), in_liq_qty.into(), inside_mid.into()]);
    let burn_ko_pos_args = UserCmdArgs {
        callpath: KNOCKOUT_LIQ_PATH, // KnockoutLiqPath index
        cmd: vec![
            Uint256::from(92u8).into(), // Burn Knockout code
            base.into(),                // base
            quote.into(),               // quote
            (pool_idx).into(),          // poolIdx
            bid_tick.into(),            // bid (lower) tick
            ask_tick.into(),            // ask (upper) tick
            is_bid.into(),
            reserve_flags.into(),
            arg_bytes.into(),
        ],
    };
    info!("Burning position: {burn_ko_pos_args:?}");
    dex_user_cmd(web3, dex, evm_privkey, burn_ko_pos_args, None, None)
        .await
        .expect("Failed to burn position in pool");

    // Unfortunately there is no way to compare the `qty` we put into the position with the `liq` result we can query
    // but we can determine if the position's liquidity has changed
    if let Some(start_pos) = start_pos {
        let pivot = pivot.unwrap();
        let ko_pos = croc_query_knockout_tokens(
            web3,
            query,
            None,
            evm_address,
            base,
            quote,
            pool_idx,
            pivot,
            bid_tick,
            ask_tick,
            is_bid,
        )
        .await
        .expect("Could not query position");
        assert!(ko_pos.liq < start_pos.liq);
    }
}

#[allow(clippy::too_many_arguments)]
pub async fn dex_mint_ambient_pos(
    web3: &Web3,
    dex: EthAddress,
    query: EthAddress,
    evm_privkey: PrivateKey,
    evm_address: EthAddress,
    base: EthAddress,
    quote: EthAddress,
    pool_idx: Uint256,
    liq: Uint256, // The liquidity to mint (must be a multiple of 1024)
) {
    assert!(base.lt(&quote), "base must be lexically smaller than quote");
    let start_pos =
        croc_query_ambient_tokens(web3, query, None, evm_address, base, quote, pool_idx)
            .await
            .expect("Could not query position");

    let mint_ambient_pos_args = UserCmdArgs {
        callpath: WARM_PATH, // Warm Path index
        cmd: vec![
            Uint256::from(3u8).into(),    // Mint Ambient Liq code
            base.into(),                  // base
            quote.into(),                 // quote
            (pool_idx).into(),            // poolIdx
            Uint256::zero().into(),       // bid (lower) tick
            Uint256::zero().into(),       // ask (upper) tick
            liq.into(), // liq (in liquidity units, which must be a multiple of 1024)
            (*MIN_PRICE).into(), // limitLower
            (*MAX_PRICE).into(), // limitHigher
            Uint256::zero().into(), // reserveFlags
            EthAddress::default().into(), // lpConduit
        ],
    };
    info!("Minting ambient position: {mint_ambient_pos_args:?}");
    dex_user_cmd(web3, dex, evm_privkey, mint_ambient_pos_args, None, None)
        .await
        .expect("Failed to mint position in pool");
    let amb_pos = croc_query_ambient_tokens(web3, query, None, evm_address, base, quote, pool_idx)
        .await
        .expect("Could not query position");
    assert!(liq - (amb_pos.liq - start_pos.liq) < 1000u32.into());
}

#[allow(clippy::too_many_arguments)]
pub async fn dex_mint_ambient_in_amount(
    web3: &Web3,
    dex: EthAddress,
    query: EthAddress,
    evm_privkey: PrivateKey,
    evm_address: EthAddress,
    base: EthAddress,
    quote: EthAddress,
    pool_idx: Uint256,
    qty: Uint256,  // The amount of a token to mint a position with
    in_base: bool, // Whether to mint in the base token or the quote token
) {
    assert!(base.lt(&quote), "base must be lexically smaller than quote");

    let code = if in_base {
        Uint256::from(31u8) // Mint in base amount code
    } else {
        Uint256::from(32u8) // Mint in quote amount code
    };

    let start_pos =
        croc_query_ambient_tokens(web3, query, None, evm_address, base, quote, pool_idx)
            .await
            .expect("Could not query position");

    let native_in = if base == EthAddress::default() {
        Some(qty)
    } else {
        None
    };

    let mint_ambient_pos_args = UserCmdArgs {
        callpath: WARM_PATH, // Warm Path index
        cmd: vec![
            code.into(),
            base.into(),                  // base
            quote.into(),                 // quote
            (pool_idx).into(),            // poolIdx
            Uint256::zero().into(),       // bid (lower) tick
            Uint256::zero().into(),       // ask (upper) tick
            qty.into(),                   // qty of tokens to supply
            (*MIN_PRICE).into(),          // limitLower
            (*MAX_PRICE).into(),          // limitHigher
            Uint256::zero().into(),       // reserveFlags
            EthAddress::default().into(), // lpConduit
        ],
    };
    info!("Minting ambient position: {mint_ambient_pos_args:?}");
    dex_user_cmd(
        web3,
        dex,
        evm_privkey,
        mint_ambient_pos_args,
        native_in,
        None,
    )
    .await
    .expect("Failed to mint position in pool");
    let amb_pos = croc_query_ambient_tokens(web3, query, None, evm_address, base, quote, pool_idx)
        .await
        .expect("Could not query position");
    assert!(amb_pos.liq > start_pos.liq);
}

#[allow(clippy::too_many_arguments)]
pub async fn dex_burn_ambient_pos(
    web3: &Web3,
    dex: EthAddress,
    query: EthAddress,
    evm_privkey: PrivateKey,
    evm_address: EthAddress,
    base: EthAddress,
    quote: EthAddress,
    pool_idx: Uint256,
    liq: Uint256, // The liquidity to mint (must be a multiple of 1024)
) {
    assert!(base.lt(&quote), "base must be lexically smaller than quote");
    let start_pos =
        croc_query_ambient_tokens(web3, query, None, evm_address, base, quote, pool_idx)
            .await
            .expect("Could not query position");

    let burn_ambient_pos_args = UserCmdArgs {
        callpath: WARM_PATH, // Warm Path index
        cmd: vec![
            Uint256::from(4u8).into(),    // Burn Ambient Liq code
            base.into(),                  // base
            quote.into(),                 // quote
            (pool_idx).into(),            // poolIdx
            Uint256::zero().into(),       // bid (lower) tick
            Uint256::zero().into(),       // ask (upper) tick
            liq.into(), // liq (in liquidity units, which must be a multiple of 1024)
            (*MIN_PRICE).into(), // limitLower
            (*MAX_PRICE).into(), // limitHigher
            Uint256::zero().into(), // reserveFlags
            EthAddress::default().into(), // lpConduit
        ],
    };
    info!("Burning ambient position: {burn_ambient_pos_args:?}");
    dex_user_cmd(web3, dex, evm_privkey, burn_ambient_pos_args, None, None)
        .await
        .expect("Failed to burn position in pool");
    let amb_pos = croc_query_ambient_tokens(web3, query, None, evm_address, base, quote, pool_idx)
        .await
        .expect("Could not query position");
    assert_eq!(start_pos.liq - amb_pos.liq, liq);
}

// Mimics the logic in the Ambient sizeAmbientLiq() function
pub fn size_ambient_liq(collateral: u128, is_add: bool, price_root: u128, in_base: bool) -> u128 {
    let buffered = buffer_collateral(collateral, is_add);
    let liq = liquidity_supported(buffered, price_root, in_base);

    if is_add {
        liq
    } else {
        liq + 1
    }
}

// Mimics the logic in the Ambient sizeConcentratedLiq() function
pub fn size_concentrated_liq(
    collateral: u128,
    is_add: bool,
    price_root: u128,
    lower_tick: i32,
    upper_tick: i32,
    in_base: bool,
) -> u128 {
    let (mut bid_price, mut ask_price) =
        determine_price_range(price_root, lower_tick, upper_tick, in_base).unwrap();
    let buffered = buffer_collateral(collateral, is_add);
    if !in_base {
        bid_price = recip_q64(bid_price);
        ask_price = recip_q64(ask_price);
    }
    let price_delta = bid_price.abs_diff(ask_price);
    let liq = liquidity_supported(buffered, price_delta, in_base);

    if is_add {
        shave_round_lots(liq)
    } else {
        shave_round_lots_up(liq)
    }
}

pub fn buffer_collateral(collateral: u128, is_add: bool) -> u128 {
    const BUFFER: u128 = 4;

    if is_add {
        collateral.saturating_sub(BUFFER)
    } else {
        #[allow(clippy::collapsible_if)]
        if collateral > u128::MAX - BUFFER {
            u128::MAX
        } else {
            collateral + BUFFER
        }
    }
}

pub fn liquidity_supported(buffered_collateral: u128, price_root: u128, in_base: bool) -> u128 {
    let bc_f = buffered_collateral.to_f64().unwrap();
    let pr_f = price_root.to_f64().unwrap();

    if in_base {
        (bc_f * 2f64.powf(64f64) / pr_f).to_u128()
    } else {
        (bc_f * pr_f / 2f64.powf(64f64)).to_u128()
    }
    .unwrap()
}

pub fn determine_price_range(
    curve_price: u128,
    lower_tick: i32,
    upper_tick: i32,
    in_base: bool,
) -> Result<(u128, u128), String> {
    let mut bid_price = tick_to_sqrt_ratio(lower_tick);
    let mut ask_price = tick_to_sqrt_ratio(upper_tick);

    if curve_price <= bid_price {
        if in_base {
            return Err("Price is below the bid price".to_string());
        }
    } else if curve_price >= ask_price {
        if !in_base {
            return Err("Price is above the ask price".to_string());
        }
    } else if in_base {
        ask_price = curve_price;
    } else {
        bid_price = curve_price;
    }

    Ok((bid_price, ask_price))
}

pub fn tick_to_sqrt_ratio(tick: i32) -> u128 {
    (1.0001f64.powf(tick as f64).sqrt() * 2f64.powf(64f64))
        .to_u128()
        .unwrap()
}

fn recip_q64(x: u128) -> u128 {
    let q128 = Uint256::from_str("340282366920938463463374607431768211456").unwrap();
    (q128 / x.into()).to_u128().unwrap()
}

fn shave_round_lots(liq: u128) -> u128 {
    (liq >> 11) << 11
}

fn shave_round_lots_up(liq: u128) -> u128 {
    ((liq >> 11) + 1) << 11
}

pub fn ambient_liq_to_flows(liq: u128, price_root: u128) -> (i128, i128) {
    let base = (price_root * liq) >> 64;
    let quote = (price_root << 64) / liq;

    (base as i128, quote as i128)
}
