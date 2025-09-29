use crate::{
    args::{
        Args, ERC20AllowanceArgs, ERC20ApproveArgs, ERC20BalanceArgs, ERC20BasicArgs,
        ERC20Subcommand, ERC20TransferArgs, Erc20Args,
    },
    format_balance,
};
use clarity::{Address as EthAddress, PrivateKey as EthPrivateKey};
use num::{Bounded, ToPrimitive};
use num256::Uint256;
use std::time::Duration;
use test_runner::evm_utils::{approve_erc20_spender, get_erc20_allowance};
use web30::client::Web3;

pub async fn handle_erc20_subcommand(web30: &Web3, args: &Args, erc20_args: &Erc20Args) {
    match &erc20_args.subcmd {
        ERC20Subcommand::Balance(cmd_args) => erc20_balance(web30, cmd_args).await,
        ERC20Subcommand::Allowance(cmd_args) => erc20_allowance(web30, cmd_args).await,
        ERC20Subcommand::Supply(cmd_args) => erc20_supply(web30, cmd_args).await,
        ERC20Subcommand::Decimals(cmd_args) => erc20_decimals(web30, cmd_args).await,
        ERC20Subcommand::Approve(cmd_args) => erc20_approve(web30, args, cmd_args).await,
        ERC20Subcommand::Transfer(cmd_args) => erc20_transfer(web30, args, cmd_args).await,
    }
}
pub async fn erc20_balance(web30: &Web3, args: &ERC20BalanceArgs) {
    let erc20: EthAddress = args.erc20;
    let address: EthAddress = args.address;
    let caller: EthAddress = args.caller.unwrap_or(address);
    let decimals = web30
        .get_erc20_decimals(erc20, caller)
        .await
        .expect("Failed to get ERC20 decimals")
        .to_usize()
        .expect("Invalid decimals");
    let symbol = web30
        .get_erc20_symbol(erc20, caller)
        .await
        .expect("Failed to get ERC20 symbol");
    let balance = web30
        .get_erc20_balance_as_address(Some(caller), erc20, address)
        .await
        .expect("Failed to get ERC20 balance");
    println!("{}{}", format_balance(balance, decimals), symbol);
}

pub async fn erc20_decimals(web30: &Web3, args: &ERC20BasicArgs) {
    let erc20: EthAddress = args.erc20;
    let caller: EthAddress = args.caller;
    debug!("Querying decimals as {caller}");
    let decimals = web30
        .get_erc20_decimals(erc20, caller)
        .await
        .expect("Failed to get ERC20 decimals")
        .to_usize()
        .expect("Invalid decimals");
    println!("Decimals: {decimals}");
}

pub async fn erc20_allowance(web30: &Web3, args: &ERC20AllowanceArgs) {
    let erc20: EthAddress = args.erc20;
    let owner: EthAddress = args.owner;
    let spender: EthAddress = args.spender;
    let decimals = web30
        .get_erc20_decimals(erc20, owner)
        .await
        .expect("Failed to get ERC20 decimals")
        .to_usize()
        .expect("Invalid decimals");
    let symbol = web30
        .get_erc20_symbol(erc20, owner)
        .await
        .expect("Failed to get ERC20 symbol");

    let allowance = get_erc20_allowance(web30, erc20, owner, spender, None)
        .await
        .expect("Failed to get ERC20 allowance");
    println!("{}{}", format_balance(allowance, decimals), symbol);
}

pub async fn erc20_supply(web30: &Web3, args: &ERC20BasicArgs) {
    let erc20: EthAddress = args.erc20;
    let caller: EthAddress = args.caller;
    debug!("Querying supply as {caller}");
    let decimals = web30
        .get_erc20_decimals(erc20, caller)
        .await
        .expect("Failed to get ERC20 decimals")
        .to_usize()
        .expect("Invalid decimals");
    let balance = web30
        .get_erc20_supply(erc20, caller)
        .await
        .expect("Failed to get ERC20 balance");
    println!("{}", format_balance(balance, decimals));
}

pub async fn erc20_approve(web30: &Web3, alt_args: &Args, args: &ERC20ApproveArgs) {
    let erc20: EthAddress = args.erc20;
    let owner: EthPrivateKey = args.owner_key;
    let spender: EthAddress = args.spender;
    let amount: Uint256 = args
        .amount
        .clone()
        .map_or(Uint256::max_value(), |a| a.parse().unwrap());

    let res = approve_erc20_spender(
        web30,
        erc20,
        owner,
        spender,
        amount,
        Some(Duration::from_secs(alt_args.timeout)),
        vec![],
    )
    .await
    .expect("Failed to approve ERC20 spender");
    println!("Transaction hash: {res}");
}

pub async fn erc20_transfer(web30: &Web3, alt_args: &Args, args: &ERC20TransferArgs) {
    let erc20: EthAddress = args.erc20;
    let owner: EthPrivateKey = args.owner_key;
    let receiver: EthAddress = args.receiver;
    let amount: Uint256 = args.amount.parse().expect("Invalid transfer amount");

    let res = web30
        .erc20_send(
            amount,
            receiver,
            erc20,
            owner,
            Some(Duration::from_secs(alt_args.timeout)),
            vec![],
        )
        .await
        .expect("Failed to transfer ERC20 tokens");
    println!("Transaction hash: {res}");
}
