use clap::Parser;
use clarity::{Address as EthAddress, PrivateKey as EthPrivateKey};

/// The Althea L1 Tool for interacting with the Althea L1 blockchain
#[derive(Parser)]
#[clap(version = env!("CARGO_PKG_VERSION"), author = "Christian Borst <christian@althea.systems>")]
pub struct Args {
    /// Increase the logging verbosity
    #[clap(short, long)]
    pub verbose: bool,
    /// (Optional) The Ethereum RPC server that will be used
    #[clap(long, default_value = "http://localhost:8545")]
    pub ethereum_rpc: String,
    /// (Optional) The Cosmos gRPC server that will be used
    #[clap(short, long, default_value = "http://localhost:9090")]
    pub cosmos_grpc: String,
    /// (Optional) The cosmos bech32 prefix, default "althea"
    #[clap(short = 'p', long, default_value = "althea")]
    pub cosmos_prefix: String,
    /// (Optional) The query timeout in seconds
    #[clap(short, long, default_value = "30")]
    pub timeout: u64,
    #[clap(subcommand)]
    pub subcmd: SubCommand,
}

#[derive(Parser)]
pub enum SubCommand {
    Erc20(Erc20Args),
    Erc721(Erc721Args),
    Dex(DexArgs),
}

/// Interact with ERC20 tokens
#[derive(Parser)]
pub struct Erc20Args {
    #[clap(subcommand)]
    pub subcmd: ERC20Subcommand,
}

#[derive(Parser)]
pub enum ERC20Subcommand {
    Balance(ERC20BalanceArgs),
    Allowance(ERC20AllowanceArgs),
    // Query the total supply of the token
    Supply(ERC20BasicArgs),
    // Query the decimals of the token
    Decimals(ERC20BasicArgs),
    Approve(ERC20ApproveArgs),
    Transfer(ERC20TransferArgs),
}

// Query the balance of `erc20` token held by `address`
#[derive(Parser)]
pub struct ERC20BalanceArgs {
    // The ERC20 to query
    #[clap(parse(try_from_str))]
    pub erc20: EthAddress,
    /// The Ethereum address to check the balance of
    #[clap(parse(try_from_str))]
    pub address: EthAddress,
    /// (Optional) The caller to simulate the transaction through
    #[clap(short, long, parse(try_from_str))]
    pub caller: Option<EthAddress>,
}

// Query the allowance of `erc20` token held by `owner` to be spent by `spender`
#[derive(Parser)]
pub struct ERC20AllowanceArgs {
    // The ERC20 to query
    #[clap(parse(try_from_str))]
    pub erc20: EthAddress,
    /// The Ethereum address of the token owner
    #[clap(parse(try_from_str))]
    pub owner: EthAddress,
    /// The Ethereum address of the spender
    #[clap(parse(try_from_str))]
    pub spender: EthAddress,
}

// Make a basic query of `erc20`
#[derive(Parser)]
pub struct ERC20BasicArgs {
    // The ERC20 to query
    #[clap(parse(try_from_str))]
    pub erc20: EthAddress,
    /// (Optional) The Ethereum address to simulate the call from
    #[clap(parse(try_from_str))]
    pub caller: EthAddress,
}

// Approve an amount of `erc20` held by `owner_key` to be spent by `spender`
#[derive(Parser)]
pub struct ERC20ApproveArgs {
    // The ERC20 to query
    #[clap(parse(try_from_str))]
    pub erc20: Option<EthAddress>,
    /// Ethereum 0x... PrivateKey owning the tokens you would like to approve for spending
    #[clap(short, long, parse(try_from_str))]
    pub owner_key: EthPrivateKey,
    /// The Ethereum address of the spender
    #[clap(short, long, parse(try_from_str))]
    pub spender: EthAddress,
    /// The amount of tokens to approve, defaults to the maximum possible value
    #[clap(short, long)]
    pub amount: Option<String>,
}

// Transfer an amount of `erc20` held by `owner_key` to `receiver`
#[derive(Parser)]
pub struct ERC20TransferArgs {
    // The ERC20 to query
    #[clap(parse(try_from_str))]
    pub erc20: Option<EthAddress>,
    /// Ethereum 0x... PrivateKey containing the tokens you would like to send
    #[clap(short, long, parse(try_from_str))]
    pub owner_key: EthPrivateKey,
    /// The Ethereum address of the receiver
    #[clap(short, long, parse(try_from_str))]
    pub receiver: EthAddress,
    /// The amount of tokens to approve, defaults to the maximum possible value
    #[clap(short, long)]
    pub amount: String,
}

/// Interact with ERC721 tokens
#[derive(Parser)]
pub struct Erc721Args {
    #[clap(subcommand)]
    pub subcmd: ERC721Subcommand,
}

#[derive(Parser)]
pub enum ERC721Subcommand {
    OwnerOf(ERC721OwnerOfArgs),
    Approved(ERC721ApprovedArgs),
    Supply(ERC721SupplyArgs),
    Approve(ERC721ApproveArgs),
    ApproveForAll(ERC721ApproveForAllArgs),
    Transfer(ERC721TransferArgs),
}

// Query the owner of `token_id` token in `erc721`
#[derive(Parser)]
pub struct ERC721OwnerOfArgs {
    /// The ERC721 to query
    #[clap(parse(try_from_str))]
    pub erc721: EthAddress,
    /// The token ID to query
    #[clap()]
    pub token_id: String,
    /// (Optional) The Ethereum address to simulate the call from
    #[clap(short, long, parse(try_from_str))]
    pub caller: Option<EthAddress>,
}

// Query the approval status for `spender` of `token_id` held by `owner` in `erc721`
#[derive(Parser)]
pub struct ERC721ApprovedArgs {
    /// The ERC721 to query
    #[clap(parse(try_from_str))]
    pub erc721: EthAddress,
    /// The token ID to query
    #[clap(short, long)]
    pub token_id: String,
    /// The Ethereum address of the token owner
    #[clap(short, long, parse(try_from_str))]
    pub owner: EthAddress,
}

// Query the total supply of `erc721`
#[derive(Parser)]
pub struct ERC721SupplyArgs {
    /// The ERC721 to query
    #[clap(parse(try_from_str))]
    pub erc721: EthAddress,
    /// (Optional) The Ethereum address to simulate the call from
    #[clap(short, long, parse(try_from_str))]
    pub caller: Option<EthAddress>,
}

// Approve `spender` to spend `token_id` held by `owner` in `erc721`
#[derive(Parser)]
pub struct ERC721ApproveArgs {
    /// The ERC721 to query
    #[clap(parse(try_from_str))]
    pub erc721: EthAddress,
    /// The token ID to approve spending of
    #[clap(short, long)]
    pub token_id: String,
    /// The Ethereum address of the spender
    #[clap(short, long, parse(try_from_str))]
    pub spender: EthAddress,
    /// The Ethereum address of the token owner
    #[clap(short, long, parse(try_from_str))]
    pub owner_key: EthPrivateKey,
}

// Approve `spender` to spend any held by `owner` in `erc721`
#[derive(Parser)]
pub struct ERC721ApproveForAllArgs {
    /// The ERC721 to query
    #[clap(parse(try_from_str))]
    pub erc721: EthAddress,
    /// The Ethereum address of the spender
    #[clap(short, long, parse(try_from_str))]
    pub spender: EthAddress,
    /// The Ethereum address of the token owner
    #[clap(short, long, parse(try_from_str))]
    pub owner_key: EthPrivateKey,
}

// Transfer `token_id` held by `owner` in `erc721` to `receiver`
#[derive(Parser)]
pub struct ERC721TransferArgs {
    /// The ERC721 to query
    #[clap(parse(try_from_str))]
    pub erc721: EthAddress,
    /// The Ethereum address of the token owner
    #[clap(short, long, parse(try_from_str))]
    pub owner_key: EthPrivateKey,
    /// The Ethereum address of the receiver
    #[clap(short, long, parse(try_from_str))]
    pub receiver: EthAddress,
    /// The token ID to transfer
    #[clap(short, long)]
    pub token_id: String,
}

/// Interact with the Althea L1 DEX
#[derive(Parser)]
pub struct DexArgs {
    #[clap(subcommand)]
    pub subcmd: DEXSubcommand,
}

#[derive(Parser)]
pub enum DEXSubcommand {
    SafeMode(DEXSafeModeArgs),
    Authority(DEXAuthorityArgs),

    /// Use CrocQuery to fetch the curve state for a particular pool
    Curve(DEXQueryPoolArgs),
    /// Use CrocQuery to fetch the params for a particular pool
    PoolParams(DEXQueryPoolArgs),
    /// Use CrocQuery to fetch the configuration for a pool template
    Template(DEXQueryTemplateArgs),
    /// Use CrocQuery to fetch the current price tick for a particular pool
    Tick(DEXQueryPoolArgs),
    /// Use CrocQuery to fetch the current liquidity for a particular pool
    Liquidity(DEXQueryPoolArgs),
    /// Use CrocQuery to fetch the current price for a particular pool
    Price(DEXQueryPoolArgs),
    /// Use CrocQuery to fetch a particular ranged or ambient position on a pool
    Position(DEXQueryPositionArgs),
    /// Use CrocQuery to fetch the rewards earned by a ranged position
    Rewards(DEXQueryRewardsArgs),
    /// Use CrocQuery to fetch the nonce set for a user performing gasless transactions
    Nonce(DEXQueryNonceArgs),

    /// Creates a new pool
    InitPool(DEXInitPoolArgs),
    /// Perform a swap on the DEX
    Swap(DEXSwapArgs),
    /// Mint a ambient liquidity position using both tokens
    MintAmbient(DEXMintAmbientArgs),
    /// Mint a ambient liquidity position using only one token
    MintAmbientQty(DEXMintAmbientQtyArgs),
    /// Mint a concentrated liquidity position using both tokens
    MintConcentrated(DEXMintConcentratedArgs),
    /// Mint a concentrated liquidity position using only one token
    MintConcentratedQty(DEXMintConcentratedQtyArgs),
    /// Mint a knockout liquidity position
    MintKnockout(DEXMintKnockoutArgs),
    /// Burn an in-progress knockout liquidity position
    BurnKnockout(DEXBurnKnockoutArgs),
    /// Withdraw the fully-swapped liquidity from a knockout position which has been knocked out
    RecoverKnockout(DEXRecoverKnockoutArgs),
    /// Install a call path on a CrocSwapDEX instance
    InstallCallpath(DEXInstallCallpathArgs),

}

/// Query the SafeMode status of the DEX
#[derive(Parser)]
pub struct DEXSafeModeArgs {
    /// The DEX address
    #[clap(parse(try_from_str))]
    pub dex: EthAddress,
    /// The address to simulate the transaction through
    #[clap(parse(try_from_str))]
    pub caller: EthAddress,
}

/// Query the current Authority of the DEX (owner or address of CrocPolicy)
#[derive(Parser)]
pub struct DEXAuthorityArgs {
    /// The DEX address
    #[clap(parse(try_from_str))]
    pub dex: EthAddress,
    /// The address to simulate the transaction through
    #[clap(parse(try_from_str))]
    pub caller: EthAddress,
}

#[derive(Parser)]
pub struct DEXQueryPoolArgs {
    /// The CrocQuery address
    #[clap(parse(try_from_str))]
    pub query_contract: EthAddress,
    /// The address to simulate the transaction through
    #[clap(parse(try_from_str))]
    pub caller: EthAddress,
    /// The base token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub base: EthAddress,
    /// The quote token
    #[clap(parse(try_from_str))]
    pub quote: EthAddress,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
}

#[derive(Parser)]
pub struct DEXQueryTemplateArgs {
    /// The CrocQuery address
    #[clap(parse(try_from_str))]
    pub query_contract: EthAddress,
    /// The address to simulate the transaction through
    #[clap(parse(try_from_str))]
    pub caller: EthAddress,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
}


#[derive(Parser)]
pub struct DEXQueryPositionArgs {
    /// The CrocQuery address
    #[clap(parse(try_from_str))]
    pub query_contract: EthAddress,
    /// The address holding the position
    #[clap(parse(try_from_str))]
    pub owner: EthAddress,
    /// (Optional) The address to simulate the transaction through
    #[clap(short, long, parse(try_from_str))]
    pub caller: Option<EthAddress>,
    /// The base token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub base: EthAddress,
    /// The quote token
    #[clap(parse(try_from_str))]
    pub quote: EthAddress,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
    /// (Optional) The lower tick if querying a ranged position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub lower_tick: Option<String>,
    /// (Optional) The upper tick if querying a ranged position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub upper_tick: Option<String>,
}

#[derive(Parser)]
pub struct DEXQueryRewardsArgs {
    /// The CrocQuery address
    #[clap(parse(try_from_str))]
    pub query_contract: EthAddress,
    /// The address holding the position
    #[clap(parse(try_from_str))]
    pub owner: EthAddress,
    /// (Optional) The address to simulate the transaction through
    #[clap(short, long, parse(try_from_str))]
    pub caller: Option<EthAddress>,
    /// The base token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub base: EthAddress,
    /// The quote token
    #[clap(parse(try_from_str))]
    pub quote: EthAddress,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
    /// The lower tick of the ranged position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub lower_tick: String,
    /// The upper tick of the ranged position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub upper_tick: String,
}

#[derive(Parser)]
pub struct DEXQueryNonceArgs {
    /// The CrocQuery address
    #[clap(parse(try_from_str))]
    pub query_contract: EthAddress,
    /// The address whose transactions are relayed
    #[clap(parse(try_from_str))]
    pub client: EthAddress,
    /// (Optional) The address to simulate the transaction through
    #[clap(short, long, parse(try_from_str))]
    pub caller: Option<EthAddress>,
    /// The salt value used for a gasless transaction
    #[clap(parse(try_from_str))]
    pub salt: String,
}

#[derive(Parser)]
pub struct DEXInitPoolArgs {
    /// The DEX address
    #[clap(parse(try_from_str))]
    pub dex_contract: EthAddress,
    /// The wallet performing the swap
    #[clap(parse(try_from_str))]
    pub wallet: EthPrivateKey,
    /// The base token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub base: EthAddress,
    /// The quote token
    #[clap(parse(try_from_str))]
    pub quote: EthAddress,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
    /// The price to set the pool at initially, in terms of base wei for quote wei
    #[clap(parse(try_from_str))]
    pub price: f64,
}

#[derive(Parser)]
pub struct DEXSwapArgs {
    /// The DEX address
    #[clap(parse(try_from_str))]
    pub dex_contract: EthAddress,
    /// The wallet performing the swap
    #[clap(parse(try_from_str))]
    pub wallet: EthPrivateKey,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
    /// The input token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub input: EthAddress,
    /// The output token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub output: EthAddress,
    /// (Optional) The input amount (only use input_amount OR output_amount, not both)
    #[clap(short, long, parse(try_from_str))]
    pub input_amount: Option<String>,
    /// (Optional) The output amount (only use input_amount OR output_amount, not both)
    #[clap(short, long, parse(try_from_str))]
    pub output_amount: Option<String>,
    /// (Optional) The tip to give to pool liquidity providers
    #[clap(long, parse(try_from_str))]
    pub tip: Option<u16>,
    /// (Optional) Limits the acceptable price the swap is allowed to push the curve to. The swap may execute up to this limit.
    #[clap(long, parse(try_from_str))]
    pub limit_price: Option<String>,
    /// (Optional) The minimum acceptable output amount
    #[clap(long, parse(try_from_str))]
    pub min_output: Option<String>,
    /// (Optional) The reserve flags to use
    #[clap(long, parse(try_from_str))]
    pub reserve_flags: Option<u8>,
    /// (Optional) If provided, will be formatted as a userCmd call instead of calling swap directly
    #[clap(long, default_value = "false", action)]
    pub format_as_user_cmd: bool,
}

#[derive(Parser)]
pub struct DEXMintAmbientArgs {
    /// The DEX address
    #[clap(parse(try_from_str))]
    pub dex_contract: EthAddress,
    /// The wallet performing the swap
    #[clap(parse(try_from_str))]
    pub wallet: EthPrivateKey,
    /// The base token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub base: EthAddress,
    /// The quote token
    #[clap(parse(try_from_str))]
    pub quote: EthAddress,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
    /// The amount to mint in terms of sqrt(X*Y) for an equivalent constant product pool
    #[clap(parse(try_from_str))]
    pub liquidity: String,
    /// (Optional) a lower price limit to prevent minting a ranged position at an unfavorable price
    #[clap(long, parse(try_from_str))]
    pub limit_lower: Option<String>,
    /// (Optional) an upper price limit to prevent minting a ranged position at an unfavorable price
    #[clap(long, parse(try_from_str))]
    pub limit_upper: Option<String>,
    /// (Optional) an address to use as the LP Conduit argument
    #[clap(long, parse(try_from_str))]
    pub lp_conduit: Option<EthAddress>,
    /// (Optional) the reserve flags to use
    #[clap(long, parse(try_from_str))]
    pub reserve_flags: Option<u8>,
}

#[derive(Parser)]
pub struct DEXMintAmbientQtyArgs {
    /// The DEX address
    #[clap(parse(try_from_str))]
    pub dex_contract: EthAddress,
    /// The wallet performing the swap
    #[clap(parse(try_from_str))]
    pub wallet: EthPrivateKey,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
    /// The base token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub base: EthAddress,
    /// The quote token
    #[clap(parse(try_from_str))]
    pub quote: EthAddress,
    /// True to use the base token as the input token, false to use quote
    #[clap(short = 'b', long, parse(try_from_str), default_value = "true")]
    pub input_is_base: bool,
    /// The amount of input tokens to use for the mint
    #[clap(parse(try_from_str))]
    pub qty: String,
    /// (Optional) a lower price limit to prevent minting a ranged position at an unfavorable price
    #[clap(long, parse(try_from_str))]
    pub limit_lower: Option<String>,
    /// (Optional) an upper price limit to prevent minting a ranged position at an unfavorable price
    #[clap(long, parse(try_from_str))]
    pub limit_upper: Option<String>,
    /// (Optional) an address to use as the LP Conduit argument
    #[clap(long, parse(try_from_str))]
    pub lp_conduit: Option<EthAddress>,
    /// (Optional) the reserve flags to use
    #[clap(long, parse(try_from_str))]
    pub reserve_flags: Option<u8>,
}

#[derive(Parser)]
pub struct DEXMintConcentratedArgs {
    /// The DEX address
    #[clap(parse(try_from_str))]
    pub dex_contract: EthAddress,
    /// The wallet performing the swap
    #[clap(parse(try_from_str))]
    pub wallet: EthPrivateKey,
    /// The base token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub base: EthAddress,
    /// The quote token
    #[clap(parse(try_from_str))]
    pub quote: EthAddress,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
    /// The amount to mint in terms of sqrt(X*Y) for an equivalent constant product pool
    #[clap(parse(try_from_str))]
    pub liquidity: String,
    /// a lower tick limit for the ranged position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub tick_lower: String,
    /// an upper tick limit for the ranged position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub tick_upper: String,
    /// (Optional) a lower price limit to prevent minting a ranged position at an unfavorable price
    #[clap(long, parse(try_from_str))]
    pub limit_lower: Option<String>,
    /// (Optional) an upper price limit to prevent minting a ranged position at an unfavorable price
    #[clap(long, parse(try_from_str))]
    pub limit_upper: Option<String>,
    /// (Optional) an address to use as the LP Conduit argument
    #[clap(long, parse(try_from_str))]
    pub lp_conduit: Option<EthAddress>,
    /// (Optional) the reserve flags to use
    #[clap(long, parse(try_from_str))]
    pub reserve_flags: Option<u8>,
}

#[derive(Parser)]
pub struct DEXMintConcentratedQtyArgs {
    /// The DEX address
    #[clap(parse(try_from_str))]
    pub dex_contract: EthAddress,
    /// The wallet performing the swap
    #[clap(parse(try_from_str))]
    pub wallet: EthPrivateKey,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
    /// The base token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub base: EthAddress,
    /// The quote token
    #[clap(parse(try_from_str))]
    pub quote: EthAddress,
    /// True to use the base token as the input token, false to use quote
    #[clap(short = 'b', long, parse(try_from_str), default_value = "true")]
    pub input_is_base: bool,
    /// The amount of input tokens to use for the mint
    #[clap(parse(try_from_str))]
    pub qty: String,
    /// a lower tick limit for the ranged position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub tick_lower: String,
    /// an upper tick limit for the ranged position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub tick_upper: String,
    /// (Optional) a lower price limit to prevent minting a ranged position at an unfavorable price
    #[clap(long, parse(try_from_str))]
    pub limit_lower: Option<String>,
    /// (Optional) an upper price limit to prevent minting a ranged position at an unfavorable price
    #[clap(long, parse(try_from_str))]
    pub limit_upper: Option<String>,
    /// (Optional) an address to use as the LP Conduit argument
    #[clap(long, parse(try_from_str))]
    pub lp_conduit: Option<EthAddress>,
    /// (Optional) the reserve flags to use
    #[clap(long, parse(try_from_str))]
    pub reserve_flags: Option<u8>,
}

#[derive(Parser)]
pub struct DEXMintKnockoutArgs {
    /// The DEX address
    #[clap(parse(try_from_str))]
    pub dex_contract: EthAddress,
    /// The wallet performing the swap
    #[clap(parse(try_from_str))]
    pub wallet: EthPrivateKey,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
    /// The base token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub base: EthAddress,
    /// The quote token
    #[clap(parse(try_from_str))]
    pub quote: EthAddress,
    /// The amount of input tokens to provide to the position
    #[clap(parse(try_from_str))]
    pub qty: String,
    /// a lower tick limit for the knockout position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub tick_lower: String,
    /// an upper tick limit for the knockout position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub tick_upper: String,
    /// True to use the base token as the input token, false to use quote
    #[clap(parse(try_from_str))]
    pub is_bid: bool,
    /// (Optional) the reserve flags to use
    #[clap(long, parse(try_from_str))]
    pub reserve_flags: Option<u8>,
    /// If true, the mint can occur with the curve price inside the range. (This should almost always be set to false.)
    #[clap(long, parse(try_from_str))]
    pub inside_mid: Option<bool>,
}

#[derive(Parser)]
pub struct DEXBurnKnockoutArgs {
    /// The DEX address
    #[clap(parse(try_from_str))]
    pub dex_contract: EthAddress,
    /// The wallet performing the swap
    #[clap(parse(try_from_str))]
    pub wallet: EthPrivateKey,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
    /// The base token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub base: EthAddress,
    /// The quote token
    #[clap(parse(try_from_str))]
    pub quote: EthAddress,
    /// The amount of input tokens to remove from the position (or sqrt(X * Y) liquidity units if in-liq-qty is true)
    #[clap(parse(try_from_str))]
    pub qty: String,
    /// a lower tick limit for the knockout position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub tick_lower: String,
    /// an upper tick limit for the knockout position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub tick_upper: String,
    /// True to use the base token as the input token, false to use quote
    #[clap(parse(try_from_str))]
    pub is_bid: bool,
    /// (Optional) the reserve flags to use
    #[clap(long, parse(try_from_str))]
    pub reserve_flags: Option<u8>,
    /// (Optional) If true, the qty amount will be interpreted in terms of sqrt(X * Y) liquidity units instead of token amounts
    #[clap(long, parse(try_from_str))]
    pub in_liq_qty: Option<bool>,
    /// (Optional) If true, the burn can occur with the curve price inside the range, this is useful for recovering partially-filled funds.
    #[clap(long, parse(try_from_str))]
    pub inside_mid: Option<bool>,
}

#[derive(Parser)]
pub struct DEXRecoverKnockoutArgs {
    /// The DEX address
    #[clap(parse(try_from_str))]
    pub dex_contract: EthAddress,
    /// The wallet performing the swap
    #[clap(parse(try_from_str))]
    pub wallet: EthPrivateKey,
    /// The index of the pool's template
    #[clap(parse(try_from_str))]
    pub pool_index: String,
    /// The base token (0x0 if using the native token)
    #[clap(parse(try_from_str))]
    pub base: EthAddress,
    /// The quote token
    #[clap(parse(try_from_str))]
    pub quote: EthAddress,
    /// a lower tick limit for the knockout position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub tick_lower: String,
    /// an upper tick limit for the knockout position
    #[clap(parse(try_from_str), allow_hyphen_values(true))]
    pub tick_upper: String,
    /// True to use the base token as the input token, false to use quote
    #[clap(parse(try_from_str))]
    pub is_bid: bool,
    /// The block time of the initial mint of the knockout position
    #[clap(parse(try_from_str))]
    pub pivot_time: u32,
    /// (Optional) the reserve flags to use
    #[clap(long, parse(try_from_str))]
    pub reserve_flags: Option<u8>,
}

#[derive(Parser)]
pub struct DEXInstallCallpathArgs {
    /// The DEX address
    #[clap(short, long, parse(try_from_str))]
    pub dex_contract: EthAddress,
    /// The Callpath contract address to install on the DEX
    #[clap(long,parse(try_from_str))]
    pub callpath_contract: EthAddress,
    /// The wallet performing the swap
    #[clap(short,long,parse(try_from_str))]
    pub wallet: EthPrivateKey,
    /// The numerical index of the callpath to install at
    #[clap(long, parse(try_from_str))]
    pub callpath_index: u16,
}
