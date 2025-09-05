use std::str::FromStr;

use clarity::Address;
use test_runner::{utils::one_eth};
use web30::client::Web3;

use crate::{args::{Args, ConfigArgs, ConfigCommand, ConfigSubcommand, DEXInitPoolArgs, DEXMintAmbientQtyArgs, DEXSetPoolTemplateArgs, DEXSwapArgs, DEXTransferDEXAuthorityArgs}, dex::{init_pool, mint_ambient_qty, set_pool_template, swap, transfer_dex_authority}};

pub const STABLESWAP_TEMPLATE: u32 = 36000;
pub const VOLATILESWAP_TEMPLATE: u32 = 36001;

#[allow(non_snake_case)]
pub struct CommonArgs {
    pub dex_contract: Address,
    pub croc_policy: Address,
    pub _croc_query: Address,
    pub ALTHEA: Address, // 18 decimals
    pub GRAV: Address, // 6 decimals
    pub USDC: Address, // 6 decimals
    pub sUSDS: Address, // 18 decimals
    pub USDT: Address, // 6 decimals
    pub USDS: Address, // 18 decimals
    
}
pub fn common_args() -> CommonArgs {
    CommonArgs {
        dex_contract: Address::from_str("0xd263DC98dEc57828e26F69bA8687281BA5D052E0").unwrap(),
        croc_policy: Address::from_str("0x14Ae279edb4D569BAFb98ff08299A0135Da6867a").unwrap(),
        _croc_query: Address::from_str("0xf7b59E4f71E467C0e409609A4a0688b073C56142").unwrap(),
        ALTHEA: Address::default(), // 0x0000000000000000000000000000000000000000
        GRAV: Address::from_str("0x1D54EcB8583Ca25895c512A8308389fFD581F9c9").unwrap(),
        sUSDS: Address::from_str("0x5FD55A1B9FC24967C4dB09C513C3BA0DFa7FF687").unwrap(),
        USDC: Address::from_str("0x80b5a32E4F032B2a058b4F29EC95EEfEEB87aDcd").unwrap(),
        USDS: Address::from_str("0xd567B3d7B8FE3C79a1AD8dA978812cfC4Fa05e75").unwrap(),
        USDT: Address::from_str("0xecEEEfCEE421D8062EF8d6b4D814efe4dc898265").unwrap(),
    }
}


pub async fn handle_config_subcommand(web30: &Web3, args: &Args, command_args: &ConfigCommand) {
    match &command_args.subcmd {
        // ConfigSubcommand::Config1(cmd_args) => config1(web30, args, cmd_args).await,
        // ConfigSubcommand::Config2(config_args) => config2(web30, args, config_args).await,
        // ConfigSubcommand::Config3(config_args) => config3(web30, args, config_args).await,
        // ConfigSubcommand::Config4(config_args) => config4(web30, args, config_args).await,
        ConfigSubcommand::Config5(config_args) => config5(web30, args, config_args).await,
        // ConfigSubcommand::Config6(config_args) => config6(web30, args, config_args).await,
        // ConfigSubcommand::Config7(config_args) => config7(web30, args, config_args).await,
        ConfigSubcommand::Config8(config_args) => config8(web30, args, config_args).await,
        ConfigSubcommand::Config9(config_args) => config9(web30, args, config_args).await,
        ConfigSubcommand::Config10(config_args) => config10(web30, args, config_args).await,
        ConfigSubcommand::Config11(config_args) => config11(web30, args, config_args).await,
        ConfigSubcommand::Config12(config_args) => config12(web30, args, config_args).await,
        ConfigSubcommand::Config13(config_args) => config13(web30, args, config_args).await,
        ConfigSubcommand::Config14(config_args) => config14(web30, args, config_args).await,
        ConfigSubcommand::Config15(config_args) => config15(web30, args, config_args).await,
        ConfigSubcommand::Config16(config_args) => config16(web30, args, config_args).await,
        ConfigSubcommand::Config17(config_args) => config17(web30, args, config_args).await,
        ConfigSubcommand::Config18(config_args) => config18(web30, args, config_args).await,
        _args => {
            panic!("Config subcommand not supported");
        }
    }
}

/// Runs a pool index update to fix the 36000 (stablecoin pair) pool template
#[allow(dead_code)]
pub async fn config1(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, USDS, USDC, sUSDS, USDT, GRAV, ..} = common_args();
    let wallet = cmd_args.wallet;

    let address = wallet.to_address();
    let usdc_bal = web30.get_erc20_balance(USDC, address).await.expect("Unable to get USDC balance");
    let usds_bal = web30.get_erc20_balance(USDS, address).await.expect("Unable to get USDS balance");
    let susds_bal = web30.get_erc20_balance(sUSDS, address).await.expect("Unable to get sUSDS balance");
    let usdt_bal = web30.get_erc20_balance(USDT, address).await.expect("Unable to get USDT balance");
    let grav_bal = web30.get_erc20_balance(GRAV, address).await.expect("Unable to get GRAV balance");

    // USDC/USDS at price 1.0, providing 10 USDC - will need 10 USDS
    // sUSDS/USDS at price 0.943396226415094340, providing 10 sUSDS - will need 10.6 USDS
    // USDS/USDT at price 1.0, providing 10 USDS - will need 10 USDT
    // ALTHEA/USDS at price 0.5, providing 10 USDS - will need 5 ALTHEA
    // GRAV/USDS at price 0.0002438, providing 10 USDS - will need 41017 GRAV

    // TOTAL: USDC = 10, sUSDS = 10, USDT = 10, ALTHEA = 5, GRAV = 41017, USDS = 50.6
    if usdc_bal < (10u32 * 10u32.pow(6)).into() {
        panic!("Not enough USDC balance to run all config commands, need at least 10 USDC");
    } 
    if susds_bal < (one_eth() * 10u32.into()) {
        panic!("Not enough sUSDS balance to run all config commands, need at least 10 sUSDS");
    }
    if usdt_bal < (10u32 * 10u32.pow(6)).into() {
        panic!("Not enough USDT balance to run all config commands, need at least 10 USDT");
    }
    if grav_bal < (41020u32 * 10u32.pow(6)).into() {
        panic!("Not enough GRAV balance to run all config commands, need at least 41020 GRAV");
    }
    if usds_bal < (one_eth() * 51u32.into()) {
        panic!("Not enough USDS balance to run all config commands, need at least 51 USDS");
    }

    let pool_index = STABLESWAP_TEMPLATE.to_string();
    let tick_size = 1;
    let fee_rate_basis_points = 50;
    let knockout_width = 64;
    let knockout_place_type = 3;
    let knockout_on_grid = true;
    let jit_thresh = 60;

    let pool_template_args = DEXSetPoolTemplateArgs {
        dex_contract,
        wallet,
        pool_index,
        fee_rate_basis_points,
        tick_size,
        jit_thresh,
        knockout_width,
        knockout_on_grid,
        knockout_place_type,
    };
    set_pool_template(web30, args, &pool_template_args).await;
}

#[allow(dead_code)]
pub async fn config2(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    #[allow(unused_variables)]
    let CommonArgs{dex_contract, ..} = common_args();
    let wallet = cmd_args.wallet;
    let pool_index = "36001".to_string();
    let tick_size = 4;
    let fee_rate_basis_points = 100;
    let knockout_width = 128;
    let knockout_place_type = 3;
    let knockout_on_grid = true;
    let jit_thresh = 60;

    let pool_template_args = DEXSetPoolTemplateArgs {
        dex_contract,
        wallet,
        pool_index,
        fee_rate_basis_points,
        tick_size,
        jit_thresh,
        knockout_width,
        knockout_on_grid,
        knockout_place_type,
    };
    set_pool_template(web30, args, &pool_template_args).await;
}

// Initialize the USDC/USDS pool
#[allow(dead_code)]
pub async fn config3(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, USDS, USDC, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = STABLESWAP_TEMPLATE.to_string();
    let base = USDC; // 6 decimals
    let quote = USDS; // 18 decimals
    let price = 10f64.powi(-12); // 1 USDC / 1 USDS = 10^6 / 10^18 = 10^-12

    let init_pool_args = DEXInitPoolArgs {
        dex_contract,
        wallet,
        pool_index,
        base,
        quote,
        price,
    };
    init_pool(web30, args, &init_pool_args).await;
}

// Initialize the sUSDS/USDS pool
#[allow(dead_code)]
pub async fn config4(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, USDS, sUSDS, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = STABLESWAP_TEMPLATE.to_string();
    let base = sUSDS; // 18 decimals
    let quote = USDS; // 18 decimals
    let price = 0.943396226415094340; // 1 sUSDS / 1.06 USDS = 0.943396226415094340

    let init_pool_args = DEXInitPoolArgs {
        dex_contract,
        wallet,
        pool_index,
        base,
        quote,
        price,
    };
    init_pool(web30, args, &init_pool_args).await;
}

// Initialize the USDS/USDT pool
pub async fn config5(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, USDS, USDT, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = STABLESWAP_TEMPLATE.to_string();
    let base = USDS; // 18 decimals
    let quote = USDT; // 6 decimals
    let price = 10f64.powi(12); // 1 USDS / 1 USDT = 10^18 / 10^6 = 10^12


    let init_pool_args = DEXInitPoolArgs {
        dex_contract,
        wallet,
        pool_index,
        base,
        quote,
        price,
    };
    init_pool(web30, args, &init_pool_args).await;
}

// Initialize the ALTHEA/USDS pool
#[allow(dead_code)]
pub async fn config6(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, USDS, ALTHEA, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = VOLATILESWAP_TEMPLATE.to_string();
    let base = ALTHEA; // 18 decimals
    let quote = USDS; // 18 decimals
    let price = 0.50; // 1 ALTHEA / 2 USDS = 0.50

    let init_pool_args = DEXInitPoolArgs {
        dex_contract,
        wallet,
        pool_index,
        base,
        quote,
        price,
    };
    init_pool(web30, args, &init_pool_args).await;
}

// Initialize the GRAV/USDS pool
#[allow(dead_code)]
pub async fn config7(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, USDS, GRAV, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = VOLATILESWAP_TEMPLATE.to_string();
    let base = GRAV; // 6 decimals
    let quote = USDS; // 18 decimals
    let price = 0.000000000000000244; // 0.0002438 GRAV / 1 USDS = 0.0002438 * 10^6 / 10^18 = 0.0002438 * 10^-12

    let init_pool_args = DEXInitPoolArgs {
        dex_contract,
        wallet,
        pool_index,
        base,
        quote,
        price,
    };
    init_pool(web30, args, &init_pool_args).await;
}

// Seed the USDC/USDS pool with some liquidity
pub async fn config8(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, USDS, USDC, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = STABLESWAP_TEMPLATE.to_string();
    let base = USDC; // 6 decimals
    let quote = USDS; // 18 decimals
    let qty = (8.9 * 10f64.powi(6)).to_string(); // 8.9 USDC


    let init_pool_args = DEXMintAmbientQtyArgs {
        dex_contract,
        wallet,
        pool_index,
        base,
        quote,
        input_is_base: true,
        qty,
        limit_lower: None,
        limit_upper: None,
        lp_conduit: None,
        reserve_flags: None,
    };
    mint_ambient_qty(web30, args, &init_pool_args).await;
}

// Seed the sUSDS/USDS pool with some liquidity
pub async fn config9(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, sUSDS, USDS, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = STABLESWAP_TEMPLATE.to_string();
    let base = sUSDS; // 18 decimals
    let quote = USDS; // 18 decimals
    let qty = (one_eth() * 8u32.into()).to_string(); // 8 sUSDS

    let init_pool_args = DEXMintAmbientQtyArgs {
        dex_contract,
        wallet,
        pool_index,
        base,
        quote,
        input_is_base: true,
        qty,
        limit_lower: None,
        limit_upper: None,
        lp_conduit: None,
        reserve_flags: None,
    };
    mint_ambient_qty(web30, args, &init_pool_args).await;
}

// Seed the USDS/USDT pool with some liquidity
pub async fn config10(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, USDT, USDS, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = STABLESWAP_TEMPLATE.to_string();
    let base = USDS; // 6 decimals
    let quote = USDT; // 18 decimals
    let qty = (one_eth() * 9u8.into()).to_string(); // 9 USDS

    let init_pool_args = DEXMintAmbientQtyArgs {
        dex_contract,
        wallet,
        pool_index,
        base,
        quote,
        input_is_base: true,
        qty,
        limit_lower: None,
        limit_upper: None,
        lp_conduit: None,
        reserve_flags: None,
    };
    mint_ambient_qty(web30, args, &init_pool_args).await;
}

// Seed the ALTHEA/USDS pool with some liquidity
pub async fn config11(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, ALTHEA, USDS, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = VOLATILESWAP_TEMPLATE.to_string();
    let base = ALTHEA; // 18 decimals
    let quote = USDS; // 18 decimals
    let qty = (one_eth() * 8u32.into()).to_string(); // 8 USDS

    let init_pool_args = DEXMintAmbientQtyArgs {
        dex_contract,
        wallet,
        pool_index,
        base,
        quote,
        input_is_base: false, // Use USDS instead of ALTHEA as the input
        qty,
        limit_lower: None,
        limit_upper: None,
        lp_conduit: None,
        reserve_flags: None,
    };
    mint_ambient_qty(web30, args, &init_pool_args).await;
}

// Seed the GRAV/USDS pool with some liquidity
pub async fn config12(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, GRAV, USDS, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = VOLATILESWAP_TEMPLATE.to_string();
    let base = GRAV; // 6 decimals
    let quote = USDS; // 18 decimals
    let qty = (8.9f64 * 10f64.powi(18)).to_string(); // 8.9 USDS

    let init_pool_args = DEXMintAmbientQtyArgs {
        dex_contract,
        wallet,
        pool_index,
        base,
        quote,
        input_is_base: false, // Use USDS instead of GRAV as the input
        qty,
        limit_lower: None,
        limit_upper: None,
        lp_conduit: None,
        reserve_flags: None,
    };
    mint_ambient_qty(web30, args, &init_pool_args).await;
}

// Test a swap on the USDC/USDS pool
pub async fn config13(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, USDC, USDS, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = STABLESWAP_TEMPLATE.to_string();
    let base = USDC; // 6 decimals
    let quote = USDS; // 18 decimals
    let qty = (10f64.powi(6)).to_string(); // 1 USDC

    let address = wallet.to_address();
    let balance_base = web30.get_erc20_balance(base, address).await.expect("Unable to get ERC20 base balance before swap");
    let balance_quote = web30.get_erc20_balance(quote, address).await.expect("Unable to get ERC20 quote balance before swap");

    let swap_args = DEXSwapArgs {
        dex_contract,
        wallet,
        pool_index,
        reserve_flags: None,
        input: base,
        output: quote,
        input_amount: Some(qty),
        output_amount: None,
        tip: None,
        limit_price: None,
        min_output: None,
        format_as_user_cmd: false,
    };
    swap(web30, args, &swap_args).await;

    let post_balance_base = web30.get_erc20_balance(base, address).await.expect("Unable to get ERC20 base balance after swap");
    let post_balance_quote = web30.get_erc20_balance(quote, address).await.expect("Unable to get ERC20 quote balance after swap");
    assert!(post_balance_base < balance_base, "Base balance did not decrease after swap");
    assert!(balance_base - post_balance_base != 1_000_000u64.into(), "Incorrect balance change after swap ({} -> {})", balance_base, post_balance_base);
    assert!(post_balance_quote > balance_quote, "Quote balance did not increase after swap");
    println!("Swapped {} USDC for {} USDS", post_balance_base - balance_base, post_balance_quote - balance_quote);
}

// Test a swap on the sUSDS/USDS pool
pub async fn config14(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, sUSDS, USDS, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = STABLESWAP_TEMPLATE.to_string();
    let base = sUSDS; // 18 decimals
    let quote = USDS; // 18 decimals
    let qty = one_eth().to_string(); // 1 sUSDS

    let address = wallet.to_address();
    let balance_base = web30.get_erc20_balance(base, address).await.expect("Unable to get ERC20 base balance before swap");
    let balance_quote = web30.get_erc20_balance(quote, address).await.expect("Unable to get ERC20 quote balance before swap");

    let swap_args = DEXSwapArgs {
        dex_contract,
        wallet,
        pool_index,
        reserve_flags: None,
        input: base,
        output: quote,
        input_amount: Some(qty),
        output_amount: None,
        tip: None,
        limit_price: None,
        min_output: None,
        format_as_user_cmd: false,
    };
    swap(web30, args, &swap_args).await;

    let post_balance_base = web30.get_erc20_balance(base, address).await.expect("Unable to get ERC20 base balance after swap");
    let post_balance_quote = web30.get_erc20_balance(quote, address).await.expect("Unable to get ERC20 quote balance after swap");
    assert!(post_balance_base < balance_base, "Base balance did not decrease after swap");
    assert!(balance_base - post_balance_base != one_eth(), "Incorrect balance change after swap ({} -> {})", balance_base, post_balance_base);
    assert!(post_balance_quote > balance_quote, "Quote balance did not increase after swap");
    println!("Swapped {} sUSDS for {} USDS", post_balance_base - balance_base, post_balance_quote - balance_quote);
}

// Test a swap on the USDS/USDT pool
pub async fn config15(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, USDT, USDS, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = STABLESWAP_TEMPLATE.to_string();
    let base = USDS; // 18 decimals
    let quote = USDT; // 6 decimals
    let qty = (one_eth()).to_string(); // 1 USDS


    let address = wallet.to_address();
    let balance_base = web30.get_erc20_balance(base, address).await.expect("Unable to get ERC20 base balance before swap");
    let balance_quote = web30.get_erc20_balance(quote, address).await.expect("Unable to get ERC20 quote balance before swap");

    let swap_args = DEXSwapArgs {
        dex_contract,
        wallet,
        pool_index,
        reserve_flags: None,
        input: base,
        output: quote,
        input_amount: Some(qty),
        output_amount: None,
        tip: None,
        limit_price: None,
        min_output: None,
        format_as_user_cmd: false,
    };
    swap(web30, args, &swap_args).await;

    let post_balance_base = web30.get_erc20_balance(base, address).await.expect("Unable to get ERC20 base balance after swap");
    let post_balance_quote = web30.get_erc20_balance(quote, address).await.expect("Unable to get ERC20 quote balance after swap");
    assert!(post_balance_base < balance_base, "Base balance did not decrease after swap");
    assert!(balance_base - post_balance_base != one_eth(), "Incorrect balance change after swap ({} -> {})", balance_base, post_balance_base);
    assert!(post_balance_quote > balance_quote, "Quote balance did not increase after swap");
    println!("Swapped {} USDS for {} USDT", post_balance_base - balance_base, post_balance_quote - balance_quote);
}

// Test a swap on the ALTHEA/USDS pool
pub async fn config16(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, ALTHEA, USDS, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = VOLATILESWAP_TEMPLATE.to_string();
    let base = ALTHEA; // 18 decimals
    let quote = USDS; // 18 decimals
    let qty = one_eth().to_string(); // 1 USDS (check the input token address)


    let address = wallet.to_address();
    let balance_base = web30.get_erc20_balance(base, address).await.expect("Unable to get ERC20 base balance before swap");
    let balance_quote = web30.get_erc20_balance(quote, address).await.expect("Unable to get ERC20 quote balance before swap");

    let swap_args = DEXSwapArgs {
        dex_contract,
        wallet,
        pool_index,
        reserve_flags: None,
        input: quote, // Swap with the quote token input this time
        output: base,
        input_amount: Some(qty),
        output_amount: None,
        tip: None,
        limit_price: None,
        min_output: None,
        format_as_user_cmd: false,
    };
    swap(web30, args, &swap_args).await;

    let post_balance_base = web30.get_erc20_balance(base, address).await.expect("Unable to get ERC20 base balance after swap");
    let post_balance_quote = web30.get_erc20_balance(quote, address).await.expect("Unable to get ERC20 quote balance after swap");
    assert!(post_balance_quote < balance_quote, "Quote balance did not decrease after swap");
    assert!(balance_quote - post_balance_quote != 1_000_000u64.into(), "Incorrect balance change after swap ({} -> {})", balance_quote, post_balance_quote);
    assert!(post_balance_base > balance_base, "Base balance did not increase after swap");
    println!("Swapped {} USDS for {} ALTHEA", post_balance_quote - balance_quote, post_balance_base - balance_base);
}

// Test a swap on the GRAV/USDS pool
pub async fn config17(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    let CommonArgs{dex_contract, GRAV, USDS, ..} = common_args();
    let wallet = cmd_args.wallet;

    let pool_index = VOLATILESWAP_TEMPLATE.to_string();
    let base = GRAV; // 18 decimals
    let quote = USDS; // 18 decimals
    let qty = one_eth().to_string(); // 1 USDS (check the input token address)


    let address = wallet.to_address();
    let balance_base = web30.get_erc20_balance(base, address).await.expect("Unable to get ERC20 base balance before swap");
    let balance_quote = web30.get_erc20_balance(quote, address).await.expect("Unable to get ERC20 quote balance before swap");

    let swap_args = DEXSwapArgs {
        dex_contract,
        wallet,
        pool_index,
        reserve_flags: None,
        input: quote, // Swap with the quote token input this time
        output: base,
        input_amount: Some(qty),
        output_amount: None,
        tip: None,
        limit_price: None,
        min_output: None,
        format_as_user_cmd: false,
    };
    swap(web30, args, &swap_args).await;

    let post_balance_base = web30.get_erc20_balance(base, address).await.expect("Unable to get ERC20 base balance after swap");
    let post_balance_quote = web30.get_erc20_balance(quote, address).await.expect("Unable to get ERC20 quote balance after swap");
    assert!(post_balance_quote < balance_quote, "Quote balance did not decrease after swap");
    assert!(balance_quote - post_balance_quote != 1_000_000u64.into(), "Incorrect balance change after swap ({} -> {})", balance_quote, post_balance_quote);
    assert!(post_balance_base > balance_base, "Base balance did not increase after swap");
    println!("Swapped {} USDS for {} GRAV", post_balance_quote - balance_quote, post_balance_base - balance_base);
}

// Transfer control of the DEX to the CrocPolicy contract
pub async fn config18(web30: &Web3, args: &Args, cmd_args: &ConfigArgs) {
    panic!("This safeguarding line needs to be removed before you can run this!");

    #[allow(unreachable_code)]
    let CommonArgs{dex_contract, croc_policy, ..} = common_args();
    let wallet = cmd_args.wallet;

    let transfer_authority_args = DEXTransferDEXAuthorityArgs {
        dex_contract,
        wallet,
        new_authority: croc_policy,
    };
    transfer_dex_authority(web30, args, &transfer_authority_args).await;
}


#[allow(dead_code)]
pub fn tick_from_root_price(price: f64) -> i32 {
    if price.abs() <= 0.0001 {
        return 0;
    }
    let price = price * price;
    let close_tick = price.log(1.0001f64);
    let tick_hi = close_tick as i32;
    let tick_lo = tick_hi - 1;

    // We need to check if we're on the cusp of a tick as we may differ from the solidity implementation and cause errors
    if price_from_tick(tick_hi) > price {
        // We're actually below the cusp, so we return the lower tick
        tick_lo
    } else {
        // Above the cusp, return the original tick
        tick_hi
    }
}
#[allow(dead_code)]
pub fn price_from_tick(tick: i32) -> f64 {
    let tick = tick as f64;
    1.0001f64.powf(tick)
}
