//! This crate runs the tests defined in the src/tests library crate. The bin and lib are separate
//! so that upgrades can be tested via a git dependency, using the old tag
//! as the commit and the current code as the new version.
#[macro_use]
extern crate log;

use deep_space::Coin;
use deep_space::Contact;
use deep_space::PrivateKey;
use std::env;
use test_runner::bootstrapping::parse_contract_addresses;
use test_runner::bootstrapping::send_erc20s_to_evm_users;
use test_runner::bootstrapping::{deploy_contracts, get_keys};
use test_runner::tests::erc20_conversion::erc20_conversion_test;
use test_runner::tests::lockup::lockup_test;
use test_runner::tests::microtx_fees::microtx_fees_test;
use test_runner::utils::one_atom;
use test_runner::utils::one_hundred_eth;
use test_runner::utils::send_funds_bulk;
use test_runner::utils::ETH_NODE;
use test_runner::utils::EVM_USER_KEYS;
use test_runner::utils::STAKING_TOKEN;
use test_runner::utils::{
    get_test_token_name, should_deploy_contracts, wait_for_cosmos_online, ADDRESS_PREFIX,
    COSMOS_NODE_GRPC, OPERATION_TIMEOUT, TOTAL_TIMEOUT,
};

#[actix_rt::main]
pub async fn main() {
    env_logger::init();
    info!("Starting Althea test-runner");
    let contact = Contact::new(
        COSMOS_NODE_GRPC.as_str(),
        OPERATION_TIMEOUT,
        ADDRESS_PREFIX.as_str(),
    )
    .unwrap();
    let web30 = web30::client::Web3::new(ETH_NODE.as_str(), OPERATION_TIMEOUT);

    if should_deploy_contracts() {
        info!("test-runner in contract deploying mode, deploying contracts, then exiting");
        deploy_contracts(&contact).await;
        return;
    }

    let contracts = parse_contract_addresses();
    // addresses of deployed ERC20 token contracts to be used for testing
    let erc20_addresses = contracts.erc20_addresses.clone();

    send_erc20s_to_evm_users(
        &web30,
        erc20_addresses.clone(),
        EVM_USER_KEYS.clone(),
        one_hundred_eth(),
    )
    .await
    .unwrap();

    info!("Waiting for Cosmos chain to come online");
    wait_for_cosmos_online(&contact, TOTAL_TIMEOUT).await;
    info!("Cosmos chain is online!");

    // keys for the primary test chain
    let keys = get_keys();

    // Send the EVM users some althea token
    send_funds_bulk(
        &contact,
        keys.get(0).expect("No validator keys?").validator_key,
        &EVM_USER_KEYS
            .clone()
            .into_iter()
            .map(|euk| euk.ethermint_address)
            .collect::<Vec<_>>(),
        Coin {
            amount: one_atom(),
            denom: STAKING_TOKEN.to_string(),
        },
        Some(OPERATION_TIMEOUT),
    )
    .await
    .unwrap();

    // assert that the validators have a balance of the footoken we use
    // for test transfers
    assert!(contact
        .get_balance(
            keys[0]
                .validator_key
                .to_address(&contact.get_prefix())
                .unwrap(),
            get_test_token_name(),
        )
        .await
        .unwrap()
        .is_some());

    let test_type = env::var("TEST_TYPE");

    info!("Starting tests with {:?}", test_type);
    if let Ok(test_type) = test_type {
        if test_type == "LOCKUP" {
            info!("Starting Lockup test");
            lockup_test(&contact, keys).await;
            return;
        } else if test_type == "MICROTX_FEES" {
            info!("Starting microtx fees test");
            microtx_fees_test(&contact, keys).await;
            return;
        } else if test_type == "ERC20_CONVERSION" {
            info!("Starting erc20 conversion test");
            erc20_conversion_test(
                &contact,
                &web30,
                keys,
                EVM_USER_KEYS.clone(),
                erc20_addresses.clone(),
            )
            .await;
            return;
        }
    }

    // this checks that the chain is continuing at the end of each test.
    contact
        .wait_for_next_block(TOTAL_TIMEOUT)
        .await
        .expect("Error chain has halted unexpectedly!");
}
