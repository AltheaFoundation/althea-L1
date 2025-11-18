//! Tests to ensure that deprecated Canto proposal types are not submittable
//! This test verifies that the old canto::erc20::v1 proposal types (RegisterCoinProposal and RegisterErc20Proposal)
//! are rejected by the chain, while the new althea::erc20::v1 types work correctly.

use crate::utils::{
    get_deposit, get_fee, ValidatorKeys, ADDRESS_PREFIX, OPERATION_TIMEOUT, STAKING_TOKEN,
    TOTAL_TIMEOUT,
};
use althea_proto::canto::erc20::v1::{
    RegisterCoinProposal as CantoRegisterCoinProposal,
    RegisterErc20Proposal as CantoRegisterErc20Proposal,
};
use althea_proto::cosmos_sdk_proto::cosmos::bank::v1beta1::{DenomUnit, Metadata};
use clarity::Address as EthAddress;
use deep_space::utils::encode_any;
use deep_space::{Coin, Contact, PrivateKey};

// Canto type URLs (old, deprecated)
pub const CANTO_REGISTER_COIN_PROPOSAL_TYPE_URL: &str = "/canto.erc20.v1.RegisterCoinProposal";
pub const CANTO_REGISTER_ERC20_PROPOSAL_TYPE_URL: &str = "/canto.erc20.v1.RegisterERC20Proposal";

/// Tests that the deprecated Canto proposal types are rejected by the chain
pub async fn deprecated_proposals_test(
    contact: &Contact,
    validator_keys: Vec<ValidatorKeys>,
    erc20_contracts: Vec<EthAddress>,
) {
    info!("Starting deprecated proposals test");

    // Try to submit a Canto RegisterErc20Proposal and expect it to fail
    test_canto_register_erc20_proposal_fails(contact, &validator_keys, &erc20_contracts).await;

    // Try to submit a Canto RegisterCoinProposal and expect it to fail
    test_canto_register_coin_proposal_fails(contact, &validator_keys).await;

    // Verify the chain still works with a basic transaction
    test_chain_still_functional(contact, &validator_keys).await;

    info!("Deprecated proposals test completed successfully!");
}

/// Attempts to submit a Canto RegisterErc20Proposal and expects it to fail
async fn test_canto_register_erc20_proposal_fails(
    contact: &Contact,
    validator_keys: &[ValidatorKeys],
    erc20_contracts: &[EthAddress],
) {
    info!("Testing that Canto RegisterErc20Proposal is rejected");

    let erc20_address = erc20_contracts
        .first()
        .expect("No ERC20 contracts provided");

    let proposal = CantoRegisterErc20Proposal {
        title: "Deprecated Canto RegisterERC20 Proposal".to_string(),
        description: "This proposal should fail because it uses the deprecated Canto type"
            .to_string(),
        erc20address: erc20_address.to_string(),
    };

    let proposal_any = encode_any(proposal, CANTO_REGISTER_ERC20_PROPOSAL_TYPE_URL.to_string());

    let deposit = get_deposit(Some(STAKING_TOKEN.to_string()));
    let fee = get_fee(None);
    let key = validator_keys[0].validator_key;

    info!("Attempting to submit Canto RegisterErc20Proposal (should fail)");
    let result = contact
        .create_legacy_gov_proposal(proposal_any, deposit, fee, key, Some(OPERATION_TIMEOUT))
        .await;

    match result {
        Ok(_) => {
            panic!("Expected Canto RegisterErc20Proposal to be rejected, but it was accepted!");
        }
        Err(e) => {
            info!(
                "Canto RegisterErc20Proposal was correctly rejected: {:?}",
                e
            );
            assert!(
                format!("{:?}", e).contains("unrecognized proposal type")
                    || format!("{:?}", e).contains("no handler exists for proposal type"),
                "Error message did not indicate unknown proposal type"
            );
        }
    }
}

/// Attempts to submit a Canto RegisterCoinProposal and expects it to fail
async fn test_canto_register_coin_proposal_fails(
    contact: &Contact,
    validator_keys: &[ValidatorKeys],
) {
    info!("Testing that Canto RegisterCoinProposal is rejected");

    let metadata = Metadata {
        base: "atestcoin".to_string(),
        description: "Test coin for deprecated proposal".to_string(),
        denom_units: vec![
            DenomUnit {
                aliases: vec![],
                denom: "atestcoin".to_string(),
                exponent: 0,
            },
            DenomUnit {
                aliases: vec![],
                denom: "testcoin".to_string(),
                exponent: 18,
            },
        ],
        display: "testcoin".to_string(),
        name: "Test Coin".to_string(),
        symbol: "TEST".to_string(),
        ..Default::default()
    };

    let proposal = CantoRegisterCoinProposal {
        title: "Deprecated Canto RegisterCoin Proposal".to_string(),
        description: "This proposal should fail because it uses the deprecated Canto type"
            .to_string(),
        metadata: Some(metadata),
    };

    let proposal_any = encode_any(proposal, CANTO_REGISTER_COIN_PROPOSAL_TYPE_URL.to_string());

    let deposit = get_deposit(Some(STAKING_TOKEN.to_string()));
    let fee = get_fee(None);
    let key = validator_keys[0].validator_key;

    info!("Attempting to submit Canto RegisterCoinProposal (should fail)");
    let result = contact
        .create_legacy_gov_proposal(proposal_any, deposit, fee, key, Some(OPERATION_TIMEOUT))
        .await;

    match result {
        Ok(_) => {
            panic!("Expected Canto RegisterCoinProposal to be rejected, but it was accepted!");
        }
        Err(e) => {
            info!("Canto RegisterCoinProposal was correctly rejected: {:?}", e);
            assert!(
                format!("{:?}", e).contains("unrecognized proposal type")
                    || format!("{:?}", e).contains("no handler exists for proposal type"),
                "Error message did not indicate unknown proposal type"
            );
        }
    }
}

/// Verifies the chain is still functional after attempting to submit deprecated proposals
async fn test_chain_still_functional(contact: &Contact, validator_keys: &[ValidatorKeys]) {
    info!("Verifying chain is still functional with a basic transaction");

    let sender = validator_keys[0].validator_key;
    let receiver = validator_keys[1]
        .validator_key
        .to_address(&ADDRESS_PREFIX)
        .unwrap();

    let send_amount = Coin {
        amount: 1000u64.into(),
        denom: STAKING_TOKEN.to_string(),
    };

    let fee = get_fee(None);

    let result = contact
        .send_coins(
            send_amount.clone(),
            Some(fee),
            receiver,
            Some(OPERATION_TIMEOUT),
            sender,
        )
        .await;

    match result {
        Ok(_) => {
            info!(
                "Successfully sent {} to verify chain functionality",
                send_amount
            );
        }
        Err(e) => {
            panic!(
                "Chain appears to be non-functional after deprecated proposal test: {:?}",
                e
            );
        }
    }

    // Wait for the next block to ensure the chain is progressing
    contact
        .wait_for_next_block(TOTAL_TIMEOUT)
        .await
        .expect("Chain failed to produce next block");

    info!("Chain is still functional!");
}
