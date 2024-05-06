use std::time::Duration;

use crate::args::{
    Args, DEXAuthorityArgs, DEXMintConcentratedArgs, DEXMintConcentratedQtyArgs, DEXQueryPoolArgs,
    DEXQueryPositionArgs, DEXQueryRewardsArgs, DEXSafeModeArgs, DEXSubcommand, DEXSwapArgs,
    DexArgs,
};
use clarity::{abi::AbiToken, Address as EthAddress};
use num256::{Int256, Uint256};
use test_runner::dex_utils::{
    croc_query_ambient_position, croc_query_conc_rewards, croc_query_curve, croc_query_curve_tick,
    croc_query_liquidity, croc_query_pool_params, croc_query_price, croc_query_range_position,
    dex_query_authority, dex_query_safe_mode, dex_swap, dex_user_cmd, SwapArgs, UserCmdArgs,
    HOT_PROXY, MAX_PRICE, MIN_PRICE, WARM_PATH,
};
use web30::client::Web3;

pub async fn handle_dex_subcommand(web30: &Web3, args: &Args, dex_args: &DexArgs) {
    match &dex_args.subcmd {
        DEXSubcommand::SafeMode(cmd_args) => safe_mode(web30, args, cmd_args).await,
        DEXSubcommand::Authority(cmd_args) => authority(web30, args, cmd_args).await,
        DEXSubcommand::Curve(cmd_args) => query_curve(web30, args, cmd_args).await,
        DEXSubcommand::PoolParams(cmd_args) => query_params(web30, args, cmd_args).await,
        DEXSubcommand::Tick(cmd_args) => query_tick(web30, args, cmd_args).await,
        DEXSubcommand::Liquidity(cmd_args) => query_liquidity(web30, args, cmd_args).await,
        DEXSubcommand::Price(cmd_args) => query_price(web30, args, cmd_args).await,
        DEXSubcommand::Position(cmd_args) => query_position(web30, args, cmd_args).await,
        DEXSubcommand::Rewards(cmd_args) => query_rewards(web30, args, cmd_args).await,
        DEXSubcommand::Swap(cmd_args) => swap(web30, args, cmd_args).await,
        DEXSubcommand::MintConcentrated(cmd_args) => mint_concentrated(web30, args, cmd_args).await,
        DEXSubcommand::MintConcentratedQty(cmd_args) => {
            mint_concentrated_qty(web30, args, cmd_args).await
        }
    }
}

pub async fn safe_mode(web30: &Web3, _args: &Args, cmd_args: &DEXSafeModeArgs) {
    let dex: EthAddress = cmd_args.dex;
    let caller: EthAddress = cmd_args.caller;
    let safe_mode = dex_query_safe_mode(web30, dex, Some(caller))
        .await
        .expect("Failed to get safe mode status");
    println!("{}", safe_mode);
}

pub async fn authority(web30: &Web3, _args: &Args, cmd_args: &DEXAuthorityArgs) {
    let dex: EthAddress = cmd_args.dex;
    let caller: EthAddress = cmd_args.caller;
    let authority = dex_query_authority(web30, dex, Some(caller))
        .await
        .expect("Failed to get safe mode status");
    println!("{}", authority);
}

pub async fn query_curve(web30: &Web3, _args: &Args, cmd_args: &DEXQueryPoolArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let curve = croc_query_curve(
        web30,
        cmd_args.query_contract,
        Some(cmd_args.caller),
        cmd_args.base,
        cmd_args.quote,
        pool_index,
    )
    .await
    .expect("Failed to query curve state");
    println!("{:?}", curve);
}

pub async fn query_params(web30: &Web3, _args: &Args, cmd_args: &DEXQueryPoolArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let params = croc_query_pool_params(
        web30,
        cmd_args.query_contract,
        Some(cmd_args.caller),
        cmd_args.base,
        cmd_args.quote,
        pool_index,
    )
    .await
    .expect("Failed to query pool params");
    println!("{:?}", params);
}

pub async fn query_tick(web30: &Web3, _args: &Args, cmd_args: &DEXQueryPoolArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let tick = croc_query_curve_tick(
        web30,
        cmd_args.query_contract,
        Some(cmd_args.caller),
        cmd_args.base,
        cmd_args.quote,
        pool_index,
    )
    .await
    .expect("Failed to query tick");
    println!("{:?}", tick);
}

pub async fn query_liquidity(web30: &Web3, _args: &Args, cmd_args: &DEXQueryPoolArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let liquidity = croc_query_liquidity(
        web30,
        cmd_args.query_contract,
        Some(cmd_args.caller),
        cmd_args.base,
        cmd_args.quote,
        pool_index,
    )
    .await
    .expect("Failed to query liquidity");
    println!("{:?}", liquidity);
}

pub async fn query_price(web30: &Web3, _args: &Args, cmd_args: &DEXQueryPoolArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let price = croc_query_price(
        web30,
        cmd_args.query_contract,
        Some(cmd_args.caller),
        cmd_args.base,
        cmd_args.quote,
        pool_index,
    )
    .await
    .expect("Failed to query price");
    println!("{:?}", price);
}

pub async fn query_position(web30: &Web3, _args: &Args, cmd_args: &DEXQueryPositionArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let lower_tick: Option<Int256> = cmd_args
        .lower_tick
        .clone()
        .map(|s| s.parse().expect("Unable to parse lower tick"));
    let upper_tick: Option<Int256> = cmd_args
        .upper_tick
        .clone()
        .map(|s| s.parse().expect("Unable to parse upper tick"));
    match (lower_tick, upper_tick) {
        (Some(lower), Some(upper)) => {
            let position = croc_query_range_position(
                web30,
                cmd_args.query_contract,
                None,
                cmd_args.owner,
                cmd_args.base,
                cmd_args.quote,
                pool_index,
                lower,
                upper,
            )
            .await
            .expect("Failed to query position");
            println!("{:?}", position);
        }
        (None, None) => {
            let position = croc_query_ambient_position(
                web30,
                cmd_args.query_contract,
                cmd_args.caller,
                cmd_args.owner,
                cmd_args.base,
                cmd_args.quote,
                pool_index,
            )
            .await
            .expect("Failed to query position");
            println!("{:?}", position);
        }
        (_, _) => println!("ERROR: Provide both lower and upper ticks for ranged positions or neither for ambient positions"),
    }
}

pub async fn query_rewards(web30: &Web3, _args: &Args, cmd_args: &DEXQueryRewardsArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let lower_tick: Int256 = cmd_args
        .lower_tick
        .parse()
        .expect("Unable to parse lower tick");
    let upper_tick: Int256 = cmd_args
        .upper_tick
        .parse()
        .expect("Unable to parse upper tick");

    let rewards = croc_query_conc_rewards(
        web30,
        cmd_args.query_contract,
        None,
        cmd_args.owner,
        cmd_args.base,
        cmd_args.quote,
        pool_index,
        lower_tick,
        upper_tick,
    )
    .await
    .expect("Failed to query rewards");
    println!("{:?}", rewards);
}

pub async fn swap(web30: &Web3, args: &Args, cmd_args: &DEXSwapArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let input_amount: Option<Uint256> = cmd_args
        .input_amount
        .clone()
        .map(|s| s.parse().expect("Invalid input_amount"));
    let output_amount: Option<Uint256> = cmd_args
        .output_amount
        .clone()
        .map(|s| s.parse().expect("Invalid output_amount"));

    match (input_amount, output_amount) {
        (Some(_), Some(_)) => {
            println!("ERROR: Provide either input_amount or output_amount, not both");
            return;
        }
        (None, None) => {
            println!("ERROR: Provide one of input_amount or output_amount");
            return;
        }
        (_, _) => {}
    };
    let (base, quote, is_buy, qty, in_base_qty) = if cmd_args.input < cmd_args.output {
        if let Some(qty) = input_amount {
            (cmd_args.input, cmd_args.output, true, qty, true)
        } else {
            let qty = output_amount.unwrap();
            (cmd_args.input, cmd_args.output, true, qty, false)
        }
    } else {
        #[allow(clippy::collapsible_else_if)]
        if let Some(qty) = input_amount {
            (cmd_args.output, cmd_args.input, false, qty, false)
        } else {
            let qty = output_amount.unwrap();
            (cmd_args.output, cmd_args.input, false, qty, true)
        }
    };
    let tip = cmd_args.tip.unwrap_or(0u16);

    // If a limit price has been provided, use it. Otherwise, determine if we need to use min/max price
    let limit_price = if let Some(limit) = cmd_args.limit_price.clone() {
        limit.parse().expect("Invalid limit price")
    } else {
        #[allow(clippy::collapsible_else_if)]
        if base == cmd_args.input {
            *MAX_PRICE
        } else {
            *MIN_PRICE
        }
    };

    let native_in = if cmd_args.input == EthAddress::default() {
        input_amount
    } else {
        None
    };

    let min_out = cmd_args
        .min_output
        .clone()
        .map_or(Uint256::default(), |s| s.parse().unwrap());

    let reserve_flags = cmd_args.reserve_flags.unwrap_or(0u8);

    if cmd_args.format_as_user_cmd {
        let res = dex_user_cmd(
            web30,
            cmd_args.dex_contract,
            cmd_args.wallet,
            UserCmdArgs {
                callpath: HOT_PROXY,
                // ABI: swapEncoded(base, quote, poolIdx, isBuy, inBaseQty, poolTip, limitPrice, minOutput, reserveFlags)
                cmd: vec![
                    base.into(),
                    quote.into(),
                    pool_index.into(),
                    is_buy.into(),
                    in_base_qty.into(),
                    tip.into(),
                    limit_price.into(),
                    min_out.into(),
                    reserve_flags.into(),
                ],
            },
            native_in,
            Some(Duration::from_secs(args.timeout)),
        )
        .await
        .expect("Unable to submit swap as userCmd to dex");
        println!("Transaction result: {:?}", res);
    } else {
        let swap_args = SwapArgs {
            base,
            quote,
            pool_idx: pool_index,
            is_buy,
            in_base_qty,
            qty,
            tip,
            limit_price,
            min_out,
            reserve_flags,
        };
        info!("Swap args: {:?}", swap_args);
        let res = dex_swap(
            web30,
            cmd_args.dex_contract,
            cmd_args.wallet,
            swap_args,
            native_in,
            Some(Duration::from_secs(args.timeout)),
        )
        .await;
        println!("Transaction result: {:?}", res);
    }
}

pub async fn mint_concentrated(web30: &Web3, args: &Args, cmd_args: &DEXMintConcentratedArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let liquidity: Uint256 = cmd_args
        .liquidity
        .clone()
        .parse()
        .expect("Invalid liquidity");
    let tick_lower: Int256 = cmd_args.tick_lower.parse().expect("Invalid tick_lower");
    let tick_upper: Int256 = cmd_args.tick_upper.parse().expect("Invalid tick_upper");

    // If a limit price has been provided, use it. Otherwise, determine if we need to use min/max price
    let limit_lower = if let Some(limit) = cmd_args.limit_lower.clone() {
        limit.parse().expect("Invalid lower limit price")
    } else {
        *MIN_PRICE
    };
    let limit_upper = if let Some(limit) = cmd_args.limit_upper.clone() {
        limit.parse().expect("Invalid upper limit price")
    } else {
        *MAX_PRICE
    };
    let reserve_flags = cmd_args.reserve_flags.unwrap_or(0u8);

    let code: Uint256 = 1u8.into();
    let user_cmd_args: Vec<AbiToken> = vec![
        code.into(),
        cmd_args.base.into(),
        cmd_args.quote.into(),
        pool_index.into(),
        tick_lower.into(),
        tick_upper.into(),
        liquidity.into(),
        limit_lower.into(),
        limit_upper.into(),
        reserve_flags.into(),
        cmd_args.lp_conduit.unwrap_or_default().into(),
    ];
    info!("Mint args: {:?}", user_cmd_args);
    let res = dex_user_cmd(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        UserCmdArgs {
            callpath: WARM_PATH,
            // ABI: commitLP(uint8 code, address base, address quote, uint256 poolIdx,
            //        int24 bidTick, int24 askTick, uint128 liq, uint128 limitLower,
            //        uint128 limitHigher, uint8 reserveFlags, address lpConduit)
            cmd: user_cmd_args,
        },
        None,
        Some(Duration::from_secs(args.timeout)),
    )
    .await
    .expect("Unable to mint ranged position on dex");
    println!("Transaction result: {:?}", res);
}

pub async fn mint_concentrated_qty(
    web30: &Web3,
    args: &Args,
    cmd_args: &DEXMintConcentratedQtyArgs,
) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let qty: Uint256 = cmd_args.qty.clone().parse().expect("Invalid qty");
    let tick_lower: Int256 = cmd_args.tick_lower.parse().expect("Invalid tick_lower");
    let tick_upper: Int256 = cmd_args.tick_upper.parse().expect("Invalid tick_upper");

    // If a limit price has been provided, use it. Otherwise, determine if we need to use min/max price
    let limit_lower = if let Some(limit) = cmd_args.limit_lower.clone() {
        limit.parse().expect("Invalid lower limit price")
    } else {
        *MIN_PRICE
    };
    let limit_upper = if let Some(limit) = cmd_args.limit_upper.clone() {
        limit.parse().expect("Invalid upper limit price")
    } else {
        *MAX_PRICE
    };
    let reserve_flags = cmd_args.reserve_flags.unwrap_or(0u8);

    let code: Uint256 = if cmd_args.input_is_base {
        info!("Minting with base currency");
        11u8.into()
    } else {
        info!("Minting with quote currency");
        12u8.into()
    };
    let native_in = if cmd_args.base == EthAddress::default() && cmd_args.input_is_base {
        Some(qty)
    } else {
        None
    };

    let user_cmd_args: Vec<AbiToken> = vec![
        code.into(),
        cmd_args.base.into(),
        cmd_args.quote.into(),
        pool_index.into(),
        tick_lower.into(),
        tick_upper.into(),
        qty.into(),
        limit_lower.into(),
        limit_upper.into(),
        reserve_flags.into(),
        cmd_args.lp_conduit.unwrap_or_default().into(),
    ];
    info!("Mint Qty args: {:?}", user_cmd_args);
    let res = dex_user_cmd(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        UserCmdArgs {
            callpath: WARM_PATH,
            // ABI: commitLP(uint8 code, address base, address quote, uint256 poolIdx,
            //        int24 bidTick, int24 askTick, uint128 liq, uint128 limitLower,
            //        uint128 limitHigher, uint8 reserveFlags, address lpConduit)
            cmd: user_cmd_args,
        },
        native_in,
        Some(Duration::from_secs(args.timeout)),
    )
    .await
    .expect("Unable to mint ranged position on dex");
    println!("Transaction result: {:?}", res);
}
