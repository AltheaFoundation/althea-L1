use crate::args::{
    Args, ERC721ApproveArgs, ERC721ApproveForAllArgs, ERC721ApprovedArgs, ERC721OwnerOfArgs,
    ERC721Subcommand, ERC721SupplyArgs, ERC721TransferArgs, Erc721Args,
};
use clarity::{Address as EthAddress, PrivateKey as EthPrivateKey};
use num256::Uint256;
use std::time::Duration;
use test_runner::evm_utils::approve_erc721_for_all;
use web30::client::Web3;

pub async fn handle_erc721_subcommand(web30: &Web3, args: &Args, erc721_args: &Erc721Args) {
    match &erc721_args.subcmd {
        ERC721Subcommand::OwnerOf(cmd_args) => erc721_owner_of(web30, args, cmd_args).await,
        ERC721Subcommand::Approved(cmd_args) => erc721_approved(web30, args, cmd_args).await,
        ERC721Subcommand::Supply(cmd_args) => erc721_supply(web30, args, cmd_args).await,
        ERC721Subcommand::Approve(cmd_args) => erc721_approve(web30, args, cmd_args).await,
        ERC721Subcommand::ApproveForAll(cmd_args) => {
            erc721_approve_for_all(web30, args, cmd_args).await
        }
        ERC721Subcommand::Transfer(cmd_args) => erc721_transfer(web30, args, cmd_args).await,
    }
}

pub async fn erc721_owner_of(web30: &Web3, _args: &Args, cmd_args: &ERC721OwnerOfArgs) {
    let erc721: EthAddress = cmd_args.erc721;
    let caller: EthAddress = cmd_args.caller.unwrap_or(erc721);
    let token_id: Uint256 = cmd_args.token_id.parse().expect("Invalid token_id");
    let owner_of = web30
        .get_erc721_owner_of(erc721, caller, token_id)
        .await
        .expect("Failed to get ERC721 owner of");
    println!("{}", owner_of);
}

pub async fn erc721_approved(web30: &Web3, _args: &Args, cmd_args: &ERC721ApprovedArgs) {
    let erc721: EthAddress = cmd_args.erc721;
    let owner: EthAddress = cmd_args.owner;
    let token_id: Uint256 = cmd_args.token_id.parse().expect("Invalid token_id");
    let approved = web30
        .check_erc721_approved(erc721, owner, token_id)
        .await
        .expect("Failed to get ERC721 approval status");
    println!(
        "{}",
        approved.map_or("No spender approved".to_string(), |a| a.to_string())
    );
}

pub async fn erc721_supply(web30: &Web3, _args: &Args, cmd_args: &ERC721SupplyArgs) {
    let erc721: EthAddress = cmd_args.erc721;
    let caller: EthAddress = cmd_args.caller.unwrap_or(erc721);
    let supply = web30
        .get_erc721_supply(erc721, caller)
        .await
        .expect("Failed to get ERC721 supply");
    println!("{}", supply);
}

pub async fn erc721_approve(web30: &Web3, args: &Args, cmd_args: &ERC721ApproveArgs) {
    let erc721: EthAddress = cmd_args.erc721;
    let owner: EthPrivateKey = cmd_args.owner_key;
    let spender: EthAddress = cmd_args.spender;
    let token_id: Uint256 = cmd_args.token_id.parse().expect("Invalid token_id");
    let timeout = Some(Duration::from_secs(args.timeout));

    let res = web30
        .approve_erc721_transfers(erc721, owner, spender, token_id, timeout, vec![])
        .await
        .expect("Failed to approve ERC721 spender");
    println!("Transaction hash: {res}");
}

pub async fn erc721_approve_for_all(web30: &Web3, args: &Args, cmd_args: &ERC721ApproveForAllArgs) {
    let erc721: EthAddress = cmd_args.erc721;
    let owner: EthPrivateKey = cmd_args.owner_key;
    let spender: EthAddress = cmd_args.spender;
    let timeout = Some(Duration::from_secs(args.timeout));

    let res = approve_erc721_for_all(web30, erc721, owner, spender, timeout)
        .await
        .expect("Failed to approve ERC721 spender");
    println!("Transaction response: {res:?}");
}

pub async fn erc721_transfer(web30: &Web3, args: &Args, cmd_args: &ERC721TransferArgs) {
    let erc721: EthAddress = cmd_args.erc721;
    let owner: EthPrivateKey = cmd_args.owner_key;
    let receiver: EthAddress = cmd_args.receiver;
    let token_id: Uint256 = cmd_args.token_id.parse().expect("Invalid token_id");
    let timeout = Some(Duration::from_secs(args.timeout));

    let res = web30
        .erc721_send(receiver, erc721, token_id, owner, timeout, vec![])
        .await
        .expect("Failed to transfer ERC721 token");
    println!("Transaction hash: {res}");
}
