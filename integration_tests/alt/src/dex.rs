use std::time::Duration;

use crate::{
    args::{
        Args, DEXAuthorityArgs, DEXBurnKnockoutArgs, DEXInitPoolArgs, DEXInstallCallpathArgs, DEXMintAmbientArgs, DEXMintAmbientQtyArgs, DEXMintConcentratedArgs, DEXMintConcentratedQtyArgs, DEXMintKnockoutArgs, DEXQueryNonceArgs, DEXQueryPoolArgs, DEXQueryPositionArgs, DEXQueryRewardsArgs, DEXQueryTemplateArgs, DEXRecoverKnockoutArgs, DEXSafeModeArgs, DEXSetPoolTemplateArgs, DEXSubcommand, DEXSwapArgs, DEXTransferCrocPolicyArgs, DEXTransferDEXAuthorityArgs, DexArgs
    },
    utils::approve_erc20s,
};
use clarity::{
    abi::{encode_tokens, AbiToken},
    Address as EthAddress,
};

use num256::{Int256, Uint256};
use test_runner::dex_utils::{
    croc_policy_transfer_governance, croc_query_ambient_position, croc_query_conc_rewards, croc_query_curve, croc_query_curve_tick, croc_query_liquidity, croc_query_nonce, croc_query_pool_params, croc_query_pool_template, croc_query_price, croc_query_range_position, dex_authority_transfer, dex_direct_protocol_cmd, dex_query_authority, dex_query_safe_mode, dex_swap, dex_user_cmd, ProtocolCmdArgs, SwapArgs, UserCmdArgs, BOOT_PATH, COLD_PATH, HOT_PROXY, KNOCKOUT_LIQ_PATH, MAX_PRICE, MIN_PRICE, WARM_PATH
};
use web30::client::Web3;

pub async fn handle_dex_subcommand(web30: &Web3, args: &Args, dex_args: &DexArgs) {
    match &dex_args.subcmd {
        // Queries
        DEXSubcommand::SafeMode(cmd_args) => safe_mode(web30, args, cmd_args).await,
        DEXSubcommand::Authority(cmd_args) => authority(web30, args, cmd_args).await,
        DEXSubcommand::Curve(cmd_args) => query_curve(web30, args, cmd_args).await,
        DEXSubcommand::PoolParams(cmd_args) => query_params(web30, args, cmd_args).await,
        DEXSubcommand::Template(cmd_args) => query_template(web30, args, cmd_args).await,
        DEXSubcommand::Tick(cmd_args) => query_tick(web30, args, cmd_args).await,
        DEXSubcommand::Liquidity(cmd_args) => query_liquidity(web30, args, cmd_args).await,
        DEXSubcommand::Price(cmd_args) => query_price(web30, args, cmd_args).await,
        DEXSubcommand::Position(cmd_args) => query_position(web30, args, cmd_args).await,
        DEXSubcommand::Rewards(cmd_args) => query_rewards(web30, args, cmd_args).await,
        DEXSubcommand::Nonce(cmd_args) => query_nonce(web30, args, cmd_args).await,

        // Transactions
        DEXSubcommand::InitPool(cmd_args) => init_pool(web30, args, cmd_args).await,
        DEXSubcommand::Swap(cmd_args) => swap(web30, args, cmd_args).await,
        DEXSubcommand::MintAmbient(cmd_args) => mint_ambient(web30, args, cmd_args).await,
        DEXSubcommand::MintAmbientQty(cmd_args) => mint_ambient_qty(web30, args, cmd_args).await,
        DEXSubcommand::MintConcentrated(cmd_args) => mint_concentrated(web30, args, cmd_args).await,
        DEXSubcommand::MintConcentratedQty(cmd_args) => {
            mint_concentrated_qty(web30, args, cmd_args).await
        }
        DEXSubcommand::MintKnockout(cmd_args) => mint_knockout(web30, args, cmd_args).await,
        DEXSubcommand::BurnKnockout(cmd_args) => burn_knockout(web30, args, cmd_args).await,
        DEXSubcommand::RecoverKnockout(cmd_args) => recover_knockout(web30, args, cmd_args).await,

        // DEX Configuration
        DEXSubcommand::InstallCallpath(cmd_args) => install_callpath(web30, args, cmd_args).await,
        DEXSubcommand::SetPoolTemplate(cmd_args) => set_pool_template(web30, args, cmd_args).await,
        DEXSubcommand::TransferDEXAuthority(cmd_args) => transfer_dex_authority(web30, args, cmd_args).await,
        DEXSubcommand::TransferCrocPolicy(cmd_args) => transfer_croc_policy(web30, args, cmd_args).await,
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

pub async fn query_template(web30: &Web3, _args: &Args, cmd_args: &DEXQueryTemplateArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let params = croc_query_pool_template(
        web30,
        cmd_args.query_contract,
        Some(cmd_args.caller),
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

pub async fn query_nonce(web30: &Web3, _args: &Args, cmd_args: &DEXQueryNonceArgs) {
    let salt_bytes = hex::decode(&cmd_args.salt).expect("Invalid salt");
    let nonce_res = croc_query_nonce(
        web30,
        cmd_args.query_contract,
        None,
        cmd_args.client,
        salt_bytes,
    )
    .await
    .expect("Failed to query rewards");
    println!("{:?}", nonce_res);
}

pub async fn init_pool(web30: &Web3, args: &Args, cmd_args: &DEXInitPoolArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let price = (cmd_args.price.sqrt() * 2f64.powi(64)) as u128; // convert to a sqrt price in Q64.64 format

    approve_erc20s(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        cmd_args.base,
        cmd_args.quote,
        50000u32.into(),
    )
    .await;

    let code: Uint256 = 71u8.into();
    let user_cmd_args: Vec<AbiToken> = vec![
        code.into(),
        cmd_args.base.into(),
        cmd_args.quote.into(),
        pool_index.into(),
        price.into(),
    ];
    let native_in = if cmd_args.base == EthAddress::default() {
        Some(10u8.into())
    } else {
        None
    };
    info!("Init Pool args: {:?}", user_cmd_args);
    let res = dex_user_cmd(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        UserCmdArgs {
            callpath: COLD_PATH,
            // ABI: init_pool(uint8 code, address base, address quote, uint256 poolIdx, uint128 price)
            cmd: user_cmd_args,
        },
        native_in,
        Some(Duration::from_secs(args.timeout)),
    )
    .await
    .expect("Unable to create new pool on dex");
    println!("Transaction result: {:?}", res);
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

    approve_erc20s(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        base,
        quote,
        500u32.into(),
    )
    .await;

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

pub async fn mint_ambient(web30: &Web3, args: &Args, cmd_args: &DEXMintAmbientArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let liquidity: Uint256 = cmd_args
        .liquidity
        .clone()
        .parse()
        .expect("Invalid liquidity");

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

    let code: Uint256 = 3u8.into();
    let user_cmd_args: Vec<AbiToken> = vec![
        code.into(),
        cmd_args.base.into(),
        cmd_args.quote.into(),
        pool_index.into(),
        0u32.into(), // Bid tick (unused)
        0u32.into(), // Ask tick (unused)
        liquidity.into(),
        limit_lower.into(),
        limit_upper.into(),
        reserve_flags.into(),
        cmd_args.lp_conduit.unwrap_or_default().into(),
    ];

    let native_in = if cmd_args.base == EthAddress::default() {
        Some(liquidity)
    } else {
        None
    };

    approve_erc20s(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        cmd_args.base,
        cmd_args.quote,
        liquidity,
    )
    .await;

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
        native_in,
        Some(Duration::from_secs(args.timeout)),
    )
    .await
    .expect("Unable to mint ranged position on dex");
    println!("Transaction result: {:?}", res);
}

pub async fn mint_ambient_qty(web30: &Web3, args: &Args, cmd_args: &DEXMintAmbientQtyArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let qty: Uint256 = cmd_args.qty.clone().parse().expect("Invalid qty");

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
        31u8.into()
    } else {
        info!("Minting with quote currency");
        32u8.into()
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
        0u32.into(), // Bid tick (unused)
        0u32.into(), // Ask tick (unused)
        qty.into(),
        limit_lower.into(),
        limit_upper.into(),
        reserve_flags.into(),
        cmd_args.lp_conduit.unwrap_or_default().into(),
    ];

    approve_erc20s(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        cmd_args.base,
        cmd_args.quote,
        qty * 2u8.into(),
    )
    .await;

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

    let native_in = if cmd_args.base == EthAddress::default() {
        Some(liquidity)
    } else {
        None
    };

    approve_erc20s(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        cmd_args.base,
        cmd_args.quote,
        liquidity,
    )
    .await;

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
        native_in,
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

    approve_erc20s(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        cmd_args.base,
        cmd_args.quote,
        qty * 2u8.into(),
    )
    .await;

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

pub async fn mint_knockout(web30: &Web3, args: &Args, cmd_args: &DEXMintKnockoutArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let qty: Uint256 = cmd_args.qty.clone().parse().expect("Invalid liquidity");
    let tick_lower: Int256 = cmd_args.tick_lower.parse().expect("Invalid tick_lower");
    let tick_upper: Int256 = cmd_args.tick_upper.parse().expect("Invalid tick_upper");

    let reserve_flags = cmd_args.reserve_flags.unwrap_or(0u8);
    let inside_mid = cmd_args.inside_mid.unwrap_or_default();

    let code: Uint256 = 91u8.into();
    // qty and insideMid are provided as abi-encoded bytes in the "args" parameter
    let other_args: Vec<AbiToken> = vec![qty.into(), inside_mid.into()];
    let other_args_bytes = encode_tokens(&other_args);
    let user_cmd_args: Vec<AbiToken> = vec![
        code.into(),
        cmd_args.base.into(),
        cmd_args.quote.into(),
        pool_index.into(),
        tick_lower.into(),
        tick_upper.into(),
        cmd_args.is_bid.into(),
        reserve_flags.into(),
        other_args_bytes.into(),
    ];

    let native_in = if cmd_args.base == EthAddress::default() {
        Some(qty)
    } else {
        None
    };

    approve_erc20s(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        cmd_args.base,
        cmd_args.quote,
        qty,
    )
    .await;

    info!("Mint args: {:?}", user_cmd_args);

    let res = dex_user_cmd(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        UserCmdArgs {
            callpath: KNOCKOUT_LIQ_PATH,
            // ABI: (uint8 code, address base, address quote, uint256 poolIdx,
            //       int24 bidTick, int24 askTick, bool isBid,
            //       uint8 reserveFlags, bytes memory args)
            cmd: user_cmd_args,
        },
        native_in,
        Some(Duration::from_secs(args.timeout)),
    )
    .await
    .expect("Unable to mint knockout position on dex");
    println!("Transaction result: {:?}", res);
}

pub async fn burn_knockout(web30: &Web3, args: &Args, cmd_args: &DEXBurnKnockoutArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let qty: Uint256 = cmd_args.qty.clone().parse().expect("Invalid liquidity");
    let tick_lower: Int256 = cmd_args.tick_lower.parse().expect("Invalid tick_lower");
    let tick_upper: Int256 = cmd_args.tick_upper.parse().expect("Invalid tick_upper");

    let reserve_flags = cmd_args.reserve_flags.unwrap_or(0u8);
    let in_liq_qty = cmd_args.in_liq_qty.unwrap_or_default();
    let inside_mid = cmd_args.inside_mid.unwrap_or_default();

    let code: Uint256 = 92u8.into();
    // qty and insideMid are provided as abi-encoded bytes in the "args" parameter
    let other_args: Vec<AbiToken> = vec![qty.into(), in_liq_qty.into(), inside_mid.into()];
    let other_args_bytes = encode_tokens(&other_args);
    let user_cmd_args: Vec<AbiToken> = vec![
        code.into(),
        cmd_args.base.into(),
        cmd_args.quote.into(),
        pool_index.into(),
        tick_lower.into(),
        tick_upper.into(),
        cmd_args.is_bid.into(),
        reserve_flags.into(),
        other_args_bytes.into(),
    ];

    info!("Burn args: {:?}", user_cmd_args);

    let res = dex_user_cmd(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        UserCmdArgs {
            callpath: KNOCKOUT_LIQ_PATH,
            // ABI: (uint8 code, address base, address quote, uint256 poolIdx,
            //       int24 bidTick, int24 askTick, bool isBid,
            //       uint8 reserveFlags, bytes memory args)
            cmd: user_cmd_args,
        },
        None,
        Some(Duration::from_secs(args.timeout)),
    )
    .await
    .expect("Unable to mint knockout position on dex");
    println!("Transaction result: {:?}", res);
}

pub async fn recover_knockout(web30: &Web3, args: &Args, cmd_args: &DEXRecoverKnockoutArgs) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    let tick_lower: Int256 = cmd_args.tick_lower.parse().expect("Invalid tick_lower");
    let tick_upper: Int256 = cmd_args.tick_upper.parse().expect("Invalid tick_upper");
    let reserve_flags = cmd_args.reserve_flags.unwrap_or(0u8);

    let pivot_time = cmd_args.pivot_time;

    let code: Uint256 = 94u8.into();
    // qty and insideMid are provided as abi-encoded bytes in the "args" parameter
    let other_args: Vec<AbiToken> = vec![pivot_time.into()];
    let other_args_bytes = encode_tokens(&other_args);
    let user_cmd_args: Vec<AbiToken> = vec![
        code.into(),
        cmd_args.base.into(),
        cmd_args.quote.into(),
        pool_index.into(),
        tick_lower.into(),
        tick_upper.into(),
        cmd_args.is_bid.into(),
        reserve_flags.into(),
        other_args_bytes.into(),
    ];
    info!("Recover args: {:?}", user_cmd_args);

    let res = dex_user_cmd(
        web30,
        cmd_args.dex_contract,
        cmd_args.wallet,
        UserCmdArgs {
            callpath: KNOCKOUT_LIQ_PATH,
            // ABI: (uint8 code, address base, address quote, uint256 poolIdx,
            //       int24 bidTick, int24 askTick, bool isBid,
            //       uint8 reserveFlags, bytes memory args)
            cmd: user_cmd_args,
        },
        None,
        Some(Duration::from_secs(args.timeout)),
    )
    .await
    .expect("Unable to mint knockout position on dex");
    println!("Transaction result: {:?}", res);
}

pub async fn install_callpath(
    web30: &Web3,
    args: &Args,
    cmd_args: &DEXInstallCallpathArgs,
) {
    let cmd = vec![21u16.into(), cmd_args.callpath_contract.into(), cmd_args.callpath_index.into()];

    let protocol_args = ProtocolCmdArgs {
        callpath: BOOT_PATH,
        cmd,
        sudo: true,
    };

    let result = dex_direct_protocol_cmd(web30, cmd_args.dex_contract, cmd_args.wallet, protocol_args, None, Some(Duration::from_secs(args.timeout))).await;
    match result {
        Ok(r) => {
            let hash = match r {
                web30::types::TransactionResponse::Eip1559 { hash, .. } => hash,
                web30::types::TransactionResponse::Eip2930 { hash, .. } => hash,
                web30::types::TransactionResponse::Legacy { hash, .. } => hash,
            };
            println!("Successful Transaction result: {hash:?}");
        },
        Err(e) => println!("Failed Transaction result: {e:?}"),
    }
}

pub async fn set_pool_template(
    web30: &Web3,
    args: &Args,
    cmd_args: &DEXSetPoolTemplateArgs,
) {
    let pool_index: Uint256 = cmd_args.pool_index.parse().expect("Invalid pool index");
    if !cmd_args.tick_size.is_power_of_two() {
        panic!("Tick size must be a power of two");
    }
    if !cmd_args.knockout_width.is_power_of_two() {
        panic!("Knockout width must be a power of two");
    }
    if cmd_args.jit_thresh % 10 != 0 {
        panic!("JIT threshold must be a multiple of 10");
    }
    let tick_size = cmd_args.tick_size.ilog2();
    let fee_rate = cmd_args.fee_rate_basis_points * 100;
    let knockout_width = cmd_args.knockout_width.ilog2();
    let knockout_place_type: u32 = (cmd_args.knockout_place_type << 4) as u32;
    let knockout_bits = knockout_width | knockout_place_type;
    let jit_thresh = cmd_args.jit_thresh / 10;
    // u16 code, uint256 poolIdx, uint16 feeRate, uint16 tickSize, uint8 jitThresh, uint8 knockout, uint8 oracleFlags)
    let cmd = vec![110u16.into(), pool_index.into(), fee_rate.into(), tick_size.into(), jit_thresh.into(), knockout_bits.into(), 0u8.into()];
    let protocol_args = ProtocolCmdArgs {
        callpath: COLD_PATH,
        cmd,
        sudo: false,
    };

    let result = dex_direct_protocol_cmd(web30, cmd_args.dex_contract, cmd_args.wallet, protocol_args, None, Some(Duration::from_secs(args.timeout))).await;
    match result {
        Ok(r) => {
            let hash = match r {
                web30::types::TransactionResponse::Eip1559 { hash, .. } => hash,
                web30::types::TransactionResponse::Eip2930 { hash, .. } => hash,
                web30::types::TransactionResponse::Legacy { hash, .. } => hash,
            };
            println!("Successful Transaction result: {hash:?}");
        },
        Err(e) => println!("Failed Transaction result: {e:?}"),
    }

}

pub async fn transfer_dex_authority(
    web30: &Web3,
    args: &Args,
    cmd_args: &DEXTransferDEXAuthorityArgs,
) {
    let result = dex_authority_transfer(web30, cmd_args.dex_contract, cmd_args.new_authority, cmd_args.wallet, Some(Duration::from_secs(args.timeout))).await;
    match result {
        Ok(r) => {
            let hash = match r {
                web30::types::TransactionResponse::Eip1559 { hash, .. } => hash,
                web30::types::TransactionResponse::Eip2930 { hash, .. } => hash,
                web30::types::TransactionResponse::Legacy { hash, .. } => hash,
            };
            println!("Successful Transaction result: {hash:?}");
        },
        Err(e) => println!("Failed Transaction result: {e:?}"),
    }

}

pub async fn transfer_croc_policy(
    web30: &Web3,
    args: &Args,
    cmd_args: &DEXTransferCrocPolicyArgs,
) {
    let result = croc_policy_transfer_governance(web30, cmd_args.croc_policy, cmd_args.wallet, cmd_args.ops_address, cmd_args.treasury_address, cmd_args.emergency_address, Some(Duration::from_secs(args.timeout))).await;
    match result {
        Ok(r) => {
            let hash = match r {
                web30::types::TransactionResponse::Eip1559 { hash, .. } => hash,
                web30::types::TransactionResponse::Eip2930 { hash, .. } => hash,
                web30::types::TransactionResponse::Legacy { hash, .. } => hash,
            };
            println!("Successful Transaction result: {hash:?}");
        },
        Err(e) => println!("Failed Transaction result: {e:?}"),
    }

}