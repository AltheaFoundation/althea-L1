#[macro_use]
extern crate log;

mod args;
mod dex;
mod erc20;
mod erc721;
mod utils;
mod config;

use args::{Args, SubCommand};
use clap::Parser;
use deep_space::Contact;
use dex::handle_dex_subcommand;
use env_logger::Env;
use erc20::handle_erc20_subcommand;
use erc721::handle_erc721_subcommand;
use num256::Uint256;
use std::time::Duration;
use web30::client::Web3;

use crate::config::handle_config_subcommand;

#[actix_rt::main]
async fn main() {
    let args: Args = Args::parse();
    let log_level = match args.verbose {
        true => "debug",
        false => "info",
    };
    env_logger::Builder::from_env(Env::default().default_filter_or(log_level)).init();
    // On Linux static builds we need to probe ssl certs path to be able to
    // do TLS stuff.
    unsafe {
        openssl_probe::init_openssl_env_vars();
    }

    // parse the arguments
    let timeout_secs = args.timeout;
    let timeout = Duration::from_secs(timeout_secs);
    let ethereum_rpc = args.ethereum_rpc.clone();
    let web30 = Web3::new(&ethereum_rpc, timeout);
    let cosmos_grpc = args.cosmos_grpc.clone();
    let _contact = Contact::new(&cosmos_grpc, timeout, &args.cosmos_prefix)
        .expect("Unable to connect to Cosmos gRPC");

    // control flow for the command structure
    match &args.subcmd {
        SubCommand::Erc20(erc20_args)=>handle_erc20_subcommand(&web30, &args,erc20_args).await,
        SubCommand::Erc721(erc721_args)=>{handle_erc721_subcommand(&web30, &args,erc721_args).await}
        SubCommand::Dex(dex_args)=>handle_dex_subcommand(&web30, &args,dex_args).await,
        SubCommand::Config(config_command)=>handle_config_subcommand(&web30, &args,config_command).await,
    }
}

pub fn format_balance(amount: Uint256, decimals: usize) -> String {
    if amount.is_zero() {
        return "0".to_string();
    }
    let s_amount = amount.to_string();
    format!(
        "{}.{}",
        &s_amount[..s_amount.len().saturating_sub(decimals)],
        &s_amount[s_amount.len().saturating_sub(decimals)..]
    )
}
