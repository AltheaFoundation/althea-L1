//! This crate runs the tests defined in the src/tests library crate. The bin and lib are separate
//! so that upgrades can be tested via a git dependency, using the old tag
//! as the commit and the current code as the new version.
#[macro_use]
extern crate log;

use deep_space::Contact;
use deep_space::PrivateKey;
use test_runner::tests::upgrade::upgrade_only;
use std::env;
use test_runner::bootstrapping::deploy_dex;
use test_runner::bootstrapping::deploy_multicall;
use test_runner::bootstrapping::parse_contract_addresses;
use test_runner::bootstrapping::parse_dex_contract_addresses;
use test_runner::bootstrapping::parse_ibc_validator_keys;
use test_runner::bootstrapping::send_erc20s_to_evm_users;
use test_runner::bootstrapping::start_ibc_relayer;
use test_runner::bootstrapping::{deploy_erc20_contracts, get_keys};
use test_runner::tests::dex::advanced_dex_test;
use test_runner::tests::dex::basic_dex_test;
use test_runner::tests::dex::dex_ops_proposal_test;
use test_runner::tests::dex::dex_safe_mode_test;
use test_runner::tests::dex::dex_swap_many;
use test_runner::tests::dex::dex_upgrade_test;
use test_runner::tests::erc20_conversion::erc20_conversion_test;
use test_runner::tests::evm_fee_burning::evm_fee_burning_test;
use test_runner::tests::ica_host::ica_host_happy_path;
use test_runner::tests::liquid_accounts::liquid_accounts_test;
use test_runner::tests::lockup::lockup_test;
use test_runner::tests::microtx_fees::microtx_fees_test;
use test_runner::tests::native_token::native_token_test;
use test_runner::tests::onboarding::onboarding_default_params;
use test_runner::tests::onboarding::onboarding_delist_after;
use test_runner::tests::onboarding::onboarding_disable_after;
use test_runner::tests::onboarding::onboarding_disabled_whitelisted;
use test_runner::tests::upgrade::upgrade_part_1;
use test_runner::tests::upgrade::upgrade_part_2;
use test_runner::utils::one_eth;
use test_runner::utils::ETH_NODE;
use test_runner::utils::EVM_USER_KEYS;
use test_runner::utils::IBC_ADDRESS_PREFIX;
use test_runner::utils::IBC_NODE_GRPC;
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
    let ibc_contact = Contact::new(
        IBC_NODE_GRPC.as_str(),
        OPERATION_TIMEOUT,
        IBC_ADDRESS_PREFIX.as_str(),
    )
    .unwrap();
    let web30 = web30::client::Web3::new(ETH_NODE.as_str(), OPERATION_TIMEOUT);

    if should_deploy_contracts() {
        info!("test-runner in contract deploying mode, deploying contracts, then exiting");
        deploy_erc20_contracts(&contact).await;
        info!("Deploying DEX");
        deploy_dex().await;
        info!("Deploying Multicall3");
        deploy_multicall().await;
        return;
    }

    let contracts = parse_contract_addresses();
    let dex_contracts = parse_dex_contract_addresses();
    // addresses of deployed ERC20 token contracts to be used for testing
    let erc20_addresses = contracts.erc20_addresses.clone();

    info!("Funding EVM users with ERC20s ({erc20_addresses:?})");
    send_erc20s_to_evm_users(
        &web30,
        erc20_addresses.clone(),
        EVM_USER_KEYS.clone(),
        one_eth() * 10_000_000u32.into(),
    )
    .await
    .unwrap();

    info!("Waiting for Cosmos chain to come online");
    wait_for_cosmos_online(&contact, TOTAL_TIMEOUT).await;
    info!("Cosmos chain is online!");

    // keys for the primary test chain
    let keys = get_keys();
    let (ibc_keys, _ibc_phrases) = parse_ibc_validator_keys();

    info!("Funding EVM users with the native coin");

    info!("Checking footoken balances");
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
            lockup_test(
                &contact,
                keys,
                &web30,
                EVM_USER_KEYS.clone(),
                erc20_addresses,
            )
            .await;
            return;
        } else if test_type == "MICROTX_FEES" {
            microtx_fees_test(&contact, keys).await;
            return;
        } else if test_type == "ERC20_CONVERSION" {
            erc20_conversion_test(
                &contact,
                &web30,
                keys,
                EVM_USER_KEYS.clone(),
                erc20_addresses,
            )
            .await;
            return;
        } else if test_type == "NATIVE_TOKEN" {
            native_token_test(&contact, &web30, keys).await;
            return;
        } else if test_type == "LIQUID_ACCOUNTS" {
            liquid_accounts_test(
                &contact,
                &web30,
                keys,
                erc20_addresses,
                EVM_USER_KEYS.clone(),
            )
            .await;
            return;
        } else if test_type == "ICA_HOST" {
            start_ibc_relayer(&contact, &ibc_contact, &keys, &ibc_keys).await;
            ica_host_happy_path(&contact, &ibc_contact, keys, ibc_keys).await;
            return;
        } else if test_type == "ONBOARDING_DEFAULT_PARAMS" {
            start_ibc_relayer(&contact, &ibc_contact, &keys, &ibc_keys).await;
            onboarding_default_params(
                &contact,
                &ibc_contact,
                keys,
                ibc_keys,
                erc20_addresses,
                EVM_USER_KEYS.clone(),
            )
            .await;
            return;
        } else if test_type == "ONBOARDING_DISABLED_WHITELISTED" {
            start_ibc_relayer(&contact, &ibc_contact, &keys, &ibc_keys).await;
            onboarding_disabled_whitelisted(
                &contact,
                &ibc_contact,
                keys,
                ibc_keys,
                erc20_addresses,
                EVM_USER_KEYS.clone(),
            )
            .await;
            return;
        } else if test_type == "ONBOARDING_DISABLE_AFTER" {
            start_ibc_relayer(&contact, &ibc_contact, &keys, &ibc_keys).await;
            onboarding_disable_after(
                &contact,
                &ibc_contact,
                keys,
                ibc_keys,
                erc20_addresses,
                EVM_USER_KEYS.clone(),
            )
            .await;
            return;
        } else if test_type == "ONBOARDING_DELIST_AFTER" {
            start_ibc_relayer(&contact, &ibc_contact, &keys, &ibc_keys).await;
            onboarding_delist_after(
                &contact,
                &ibc_contact,
                keys,
                ibc_keys,
                erc20_addresses,
                EVM_USER_KEYS.clone(),
            )
            .await;
            return;
        } else if test_type == "DEX" {
            basic_dex_test(
                &contact,
                &web30,
                keys,
                EVM_USER_KEYS.clone(),
                erc20_addresses,
                dex_contracts,
                contracts.walthea_address,
            )
            .await;
            return;
        } else if test_type == "DEX_ADVANCED" {
            advanced_dex_test(
                &contact,
                &web30,
                keys,
                EVM_USER_KEYS.clone(),
                erc20_addresses,
                dex_contracts,
                contracts.walthea_address,
            )
            .await;
            return;
        } else if test_type == "DEX_SWAP_MANY" {
            dex_swap_many(
                &web30,
                EVM_USER_KEYS.clone(),
                erc20_addresses,
                dex_contracts,
            )
            .await;
            return;
        } else if test_type == "DEX_UPGRADE" {
            dex_upgrade_test(
                &contact,
                &web30,
                keys,
                EVM_USER_KEYS.clone(),
                erc20_addresses,
                dex_contracts,
                contracts.walthea_address,
            )
            .await;
            return;
        } else if test_type == "DEX_SAFE_MODE" {
            dex_safe_mode_test(
                &contact,
                &web30,
                keys,
                EVM_USER_KEYS.clone(),
                erc20_addresses,
                dex_contracts,
                contracts.walthea_address,
            )
            .await;
            return;
        } else if test_type == "DEX_OPS_PROPOSAL" {
            dex_ops_proposal_test(
                &contact,
                &web30,
                keys,
                EVM_USER_KEYS.clone(),
                erc20_addresses,
                dex_contracts,
                contracts.walthea_address,
            )
            .await;
            return;
        } else if test_type == "EVM_FEES"
            || test_type == "EVM_FEE_BURNING"
            || test_type == "EVM_FEE_BURN"
        {
            evm_fee_burning_test(
                &contact,
                &web30,
                keys,
                EVM_USER_KEYS.clone(),
                erc20_addresses,
            )
            .await;
            return;
        } else if test_type == "UPGRADE_PART_1" {
            upgrade_part_1(
                &web30,
                &contact,
                &ibc_contact,
                keys,
                ibc_keys,
                erc20_addresses,
            )
            .await;
        } else if test_type == "UPGRADE_PART_2" {
            upgrade_part_2(
                &web30,
                &contact,
                &ibc_contact,
                keys,
                ibc_keys,
                erc20_addresses,
            )
            .await;
        } else if test_type == "UPGRADE_ONLY" {
            upgrade_only(
                &contact,
                keys,
            )
            .await;
        } else {
            panic!("Unknown test type: {:?}", test_type);
        }

        // this checks that the chain is continuing at the end of each test (but not for upgrade part 1, which should halt)
        if !(test_type == "UPGRADE_PART_1" || test_type == "UPGRADE_ONLY") {
            contact
                .wait_for_next_block(TOTAL_TIMEOUT)
                .await
                .expect("Error chain has halted unexpectedly!");
        }
    }
}
