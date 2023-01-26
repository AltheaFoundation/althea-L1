//! This crate runs the tests defined in the src/tests library crate. The bin and lib are separate
//! so that upgrades can be tested via a git dependency, using the old tag
//! as the commit and the current code as the new version.
#[macro_use]
extern crate log;

use deep_space::Contact;
use deep_space::PrivateKey;
use std::env;
use test_runner::bootstrapping::get_keys;
use test_runner::tests::lockup::lockup_test;
use test_runner::tests::microtx_fees::microtx_fees_test;
use test_runner::utils::{
    get_test_token_name, wait_for_cosmos_online, ADDRESS_PREFIX, COSMOS_NODE_GRPC,
    OPERATION_TIMEOUT, TOTAL_TIMEOUT,
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

    info!("Waiting for Cosmos chain to come online");
    wait_for_cosmos_online(&contact, TOTAL_TIMEOUT).await;
    info!("Cosmos chain is online!");

    // keys for the primary test chain
    let keys = get_keys();

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
        }
    }

    // this checks that the chain is continuing at the end of each test.
    contact
        .wait_for_next_block(TOTAL_TIMEOUT)
        .await
        .expect("Error chain has halted unexpectedly!");
}
