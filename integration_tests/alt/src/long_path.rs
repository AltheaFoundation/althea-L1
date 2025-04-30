use clarity::{abi::encode_tokens, Address as EthAddress};
use num256::Uint256;
use num_traits::ToPrimitive;

#[derive(Debug)]
pub struct OrderDirective {
    pub open: SettlementChannel,
    pub hops: Vec<HopDirective>,
}

#[derive(Debug)]
pub struct SettlementChannel {
    pub token: EthAddress,
    pub limit_qty: i128,
    pub dust_thresh: u128,
    pub use_surplus: bool,
}

#[derive(Debug)]
pub struct HopDirective {
    pub pools: Vec<PoolDirective>,
    pub settle: SettlementChannel,
    pub improve: PriceImproveReq,
}

#[derive(Debug)]
pub struct PoolDirective {
    pub pool_idx: u128,
    pub ambient: AmbientDirective,
    pub conc: Vec<ConcentratedDirective>,
    pub swap: SwapDirective,
    pub chain: ChainingFlags,
}

#[derive(Debug)]
pub struct PriceImproveReq {
    pub is_enabled: bool,
    pub use_base_side: bool,
}

#[derive(Debug)]
pub struct AmbientDirective {
    pub is_add: bool,
    pub roll_type: u8,
    pub liquidity: u128,
}

#[derive(Debug)]
pub struct ConcentratedDirective {
    pub low_tick: i32,
    pub high_tick: i32,
    pub is_add: bool,
    pub is_tick_rel: bool,
    pub roll_type: u8,
    pub liquidity: u128,
}

#[derive(Debug)]
pub struct SwapDirective {
    pub is_buy: bool,
    pub in_base_qty: bool,
    pub roll_type: u8,
    pub qty: u128,
    pub limit_price: u128,
}

#[derive(Debug)]
pub struct ChainingFlags {
    pub roll_exit: u8,
    pub swap_defer: bool,
    pub offset_surplus: bool,
}

pub(crate) fn pack_order(
    base: EthAddress,
    quote: EthAddress,
    pool_idx: Uint256,
    is_buy: bool,
    qty: Uint256,
    in_base_qty: bool,
    limit_price: Uint256,
    min_out: u128,
    surplus_in: bool,
    surplus_out: bool,
) -> OrderDirective {
    let pool = PoolDirective {
        pool_idx: pool_idx.to_u128().unwrap(),
        ambient: AmbientDirective {
            is_add: false,
            roll_type: 0,
            liquidity: 0,
        },
        conc: vec![],
        swap: SwapDirective {
            is_buy,
            in_base_qty,
            roll_type: 0,
            qty: qty.to_u128().unwrap(),
            limit_price: limit_price.to_u128().unwrap(),
        },
        chain: ChainingFlags {
            roll_exit: 0,
            swap_defer: false,
            offset_surplus: false,
        },
    };
    let hop = HopDirective {
        settle: SettlementChannel {
            token: quote,
            limit_qty: -(min_out as i128),
            dust_thresh: 0,
            use_surplus: surplus_out,
        },
        pools: vec![pool],
        improve: PriceImproveReq {
            is_enabled: false,
            use_base_side: false,
        },
    };

    OrderDirective {
        open: SettlementChannel {
            token: base,
            limit_qty: 2i128.pow(125),
            dust_thresh: 0,
            use_surplus: surplus_in,
        },
        hops: vec![hop],
    }
}

impl OrderDirective {
    pub fn encode_bytes(&self) -> Vec<u8> {
        let schema = encode_tokens(&[1u8.into()]);
        let open = encode_tokens(&[
            self.open.token.into(),
            self.open.limit_qty.into(),
            self.open.dust_thresh.into(),
            self.open.use_surplus.into(),
        ]);
        let hops = self
            .hops
            .iter()
            .map(|h| hop_to_bytes(h))
            .collect::<Vec<_>>();
        let hops_len = encode_tokens(&[(hops.len() as u64).into()]);

        let mut res = Vec::new();
        res.extend(schema);
        res.extend(open);
        res.extend(hops_len);
        for hop in hops {
            res.extend(hop);
        }
        res
    }
}

fn hop_to_bytes(hop: &HopDirective) -> Vec<u8> {
    let pools: Vec<Vec<u8>> = hop.pools.iter().map(|p| pool_to_bytes(p)).collect();
    let pools_len = encode_tokens(&[(pools.len() as u64).into()]);
    let settle = encode_tokens(&[
        hop.settle.token.into(),
        hop.settle.limit_qty.into(),
        hop.settle.dust_thresh.into(),
        hop.settle.use_surplus.into(),
    ]);
    let improve = encode_tokens(&[
        hop.improve.is_enabled.into(),
        hop.improve.use_base_side.into(),
    ]);

    let mut res = vec![];
    res.extend(pools_len);
    for pool in pools {
        res.extend(pool);
    }
    res.extend(settle);
    res.extend(improve);
    res
}

fn pool_to_bytes(pool: &PoolDirective) -> Vec<u8> {
    let pool_idx: Vec<u8> = encode_tokens(&[pool.pool_idx.into()]);
    let ambient = encode_tokens(&[
        pool.ambient.is_add.into(),
        pool.ambient.roll_type.into(),
        pool.ambient.liquidity.into(),
    ]);

    let concs: Vec<Vec<u8>> = pool.conc.iter().map(|c| conc_to_bytes(c)).collect();
    let concs_len = encode_tokens(&[(concs.len() as u64).into()]);

    let swap = encode_tokens(&[
        pool.swap.is_buy.into(),
        pool.swap.in_base_qty.into(),
        pool.swap.roll_type.into(),
        pool.swap.qty.into(),
        pool.swap.limit_price.into(),
    ]);
    let chain = encode_tokens(&[
        pool.chain.roll_exit.into(),
        pool.chain.swap_defer.into(),
        pool.chain.offset_surplus.into(),
    ]);

    let mut res = Vec::new();
    res.extend(pool_idx);
    res.extend(ambient);
    res.extend(concs_len);
    for conc in concs {
        res.extend(conc);
    }
    res.extend(swap);
    res.extend(chain);
    res
}

fn conc_to_bytes(conc: &ConcentratedDirective) -> Vec<u8> {
    encode_tokens(&[
        conc.low_tick.into(),
        conc.high_tick.into(),
        conc.is_add.into(),
        conc.is_tick_rel.into(),
        conc.roll_type.into(),
        conc.liquidity.into(),
    ])
}
