use crate::utils::{
    execute_upgrade_proposal, wait_for_block, UpgradeProposalParams, ValidatorKeys, EVM_USER_KEYS,
};
use clarity::Address as EthAddress;
use deep_space::client::ChainStatus;
use deep_space::{Contact, CosmosPrivateKey};
use std::time::Duration;
use tokio::time::sleep as delay_for;
use web30::client::Web3;

use super::erc20_conversion::erc20_conversion_test;
use super::microtx_fees::microtx_fees_test;
use super::native_token::native_token_test;

pub const UPGRADE_NAME: &str = "example";
const UPGRADE_BLOCK_DELTA: u64 = 30;

/// Perform a series of integration tests to seed the system with data, then submit and pass a chain
/// upgrade proposal
/// NOTE: To run this test, use the tests/run-upgrade-test.sh command with an old binary
#[allow(clippy::too_many_arguments)]
pub async fn upgrade_part_1(
    web30: &Web3,
    althea_contact: &Contact,
    ibc_contact: &Contact,
    keys: Vec<ValidatorKeys>,
    ibc_keys: Vec<CosmosPrivateKey>,
    erc20_addresses: Vec<EthAddress>,
) {
    info!("Starting upgrade test part 1");

    run_all_recoverable_tests(web30, althea_contact, keys.clone(), erc20_addresses.clone()).await;
    run_upgrade_specific_tests(
        web30,
        althea_contact,
        ibc_contact,
        keys.clone(),
        ibc_keys.clone(),
        erc20_addresses.clone(),
        false,
    )
    .await;

    let upgrade_height = run_upgrade(althea_contact, keys, UPGRADE_NAME.to_string(), false).await;

    info!(
        "Ready to run the new binary, waiting for chain panic at upgrade height of {upgrade_height}!"
    );
    // Wait for the block before the upgrade height, we won't get a response from the chain
    let res = wait_for_block(althea_contact, (upgrade_height - 1) as u64).await;
    if res.is_err() {
        panic!("Unable to wait for upgrade! {}", res.err().unwrap());
    }

    delay_for(Duration::from_secs(10)).await; // wait for the new block to halt the chain
    let status = althea_contact.get_chain_status().await;
    info!("Done waiting, chain should be halted, status response: {status:?}");
}

/// Perform a series of integration tests after an upgrade has executed
/// NOTE: To run this test, follow the instructions for v2_upgrade_part_1 and WAIT FOR CHAIN HALT,
/// then finally run tests/run-tests.sh with V2_UPGRADE_PART_2 as the test type.
#[allow(clippy::too_many_arguments)]
pub async fn upgrade_part_2(
    web30: &Web3,
    althea_contact: &Contact,
    ibc_contact: &Contact,
    keys: Vec<ValidatorKeys>,
    ibc_keys: Vec<CosmosPrivateKey>,
    erc20_addresses: Vec<EthAddress>,
) {
    info!("Starting upgrade_part_2 test");

    run_all_recoverable_tests(web30, althea_contact, keys.clone(), erc20_addresses.clone()).await;
    run_upgrade_specific_tests(
        web30,
        althea_contact,
        ibc_contact,
        keys.clone(),
        ibc_keys,
        erc20_addresses.clone(),
        true,
    )
    .await;
}

/// Submit and pass a chain upgrade proposal immediately
#[allow(clippy::too_many_arguments)]
pub async fn upgrade_only(
    althea_contact: &Contact,
    keys: Vec<ValidatorKeys>,
) {
    info!("Initiating {UPGRADE_NAME} upgrade");

    let upgrade_height = run_upgrade(althea_contact, keys, UPGRADE_NAME.to_string(), false).await;

    info!(
        "Ready to run the new binary, waiting for chain panic at upgrade height of {}!",
        upgrade_height
    );
    // Wait for the block before the upgrade height, we won't get a response from the chain
    let res = wait_for_block(althea_contact, (upgrade_height - 1) as u64).await;
    if res.is_err() {
        panic!("Unable to wait for upgrade! {}", res.err().unwrap());
    }

    delay_for(Duration::from_secs(10)).await; // wait for the new block to halt the chain
    let status = althea_contact.get_chain_status().await;
    info!(
        "Done waiting, chain should be halted, status response: {:?}",
        status
    );
}

pub async fn run_upgrade(
    contact: &Contact,
    keys: Vec<ValidatorKeys>,
    plan_name: String,
    wait_for_upgrade: bool,
) -> i64 {
    let curr_height = contact.get_chain_status().await.unwrap();
    let curr_height = if let ChainStatus::Moving { block_height } = curr_height {
        block_height
    } else {
        panic!("Chain is not moving!");
    };
    let upgrade_height = (curr_height + UPGRADE_BLOCK_DELTA) as i64;
    let upgrade_prop_params = UpgradeProposalParams {
        upgrade_height,
        plan_name,
        plan_info: "upgrade info here".to_string(),
        proposal_title: "proposal title here".to_string(),
        proposal_desc: "proposal description here".to_string(),
    };
    info!(
        "Starting upgrade vote with params name: {}, height: {}",
        upgrade_prop_params.plan_name.clone(),
        upgrade_height
    );
    execute_upgrade_proposal(contact, &keys, None, upgrade_prop_params).await;

    if wait_for_upgrade {
        info!(
            "Ready to run the new binary, waiting for chain panic at upgrade height of {upgrade_height}!"
        );
        // Wait for the block before the upgrade height, we won't get a response from the chain
        let res = wait_for_block(contact, (upgrade_height - 1) as u64).await;
        if res.is_err() {
            panic!("Unable to wait for upgrade! {}", res.err().unwrap());
        }

        delay_for(Duration::from_secs(10)).await; // wait for the new block to halt the chain
        let status = contact.get_chain_status().await;
        info!("Done waiting, chain should be halted, status response: {status:?}");
    }
    upgrade_height
}

/// Runs many integration tests, but only the ones which DO NOT corrupt state
pub async fn run_all_recoverable_tests(
    web30: &Web3,
    contact: &Contact,
    keys: Vec<ValidatorKeys>,
    erc20_addresses: Vec<EthAddress>,
) {
    native_token_test(contact, web30, keys.clone()).await;
    erc20_conversion_test(
        contact,
        web30,
        keys.clone(),
        EVM_USER_KEYS.clone(),
        erc20_addresses,
    )
    .await;
    microtx_fees_test(contact, keys.clone()).await;
}

// These tests should fail in upgrade_part_1() but pass in upgrade_part_2()
#[allow(clippy::too_many_arguments)]
pub async fn run_upgrade_specific_tests(
    _web30: &Web3,
    _althea_contact: &Contact,
    _ibc_contact: &Contact,
    _keys: Vec<ValidatorKeys>,
    _ibc_keys: Vec<CosmosPrivateKey>,
    _erc20_addresses: Vec<EthAddress>,
    _post_upgrade: bool,
) {
}
