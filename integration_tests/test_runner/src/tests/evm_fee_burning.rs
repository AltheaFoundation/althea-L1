use std::time::Duration;

use crate::utils::{
    create_parameter_change_proposal, get_fee, one_atom, vote_yes_on_proposals,
    wait_for_proposals_to_execute, EthermintUserKey, ValidatorKeys, OPERATION_TIMEOUT,
    STAKING_TOKEN,
};
use althea_proto::cosmos_sdk_proto::cosmos::base::v1beta1::DecCoin;
use althea_proto::cosmos_sdk_proto::cosmos::distribution::v1beta1::{
    query_client::QueryClient as DistributionQueryClient, QueryValidatorOutstandingRewardsRequest,
};
use althea_proto::cosmos_sdk_proto::cosmos::params::v1beta1::ParamChange;
use clarity::Address as EthAddress;
use deep_space::{Coin, Contact, PrivateKey};
use futures::future::join_all;
use num::Bounded;
use num256::Uint256;
use tokio::time::sleep;
use web30::client::Web3;

pub async fn evm_fee_burning_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: Vec<ValidatorKeys>,
    evm_user_keys: Vec<EthermintUserKey>,
    erc20_contracts: Vec<EthAddress>,
) {
    info!("Start evm fee burning test");
    info!("Set inflation to 0");
    set_inflation_to_zero(contact, &validator_keys).await;

    let pre_supply = contact
        .query_supply_of(STAKING_TOKEN.clone())
        .await
        .expect("Unable to get aalthea supply")
        .expect("No supply of aalthea?");
    let pre_balances = snapshot_validator_rewards(contact, &validator_keys).await;

    info!("Generating some fees");
    let gas_multiplier = web30::types::SendTxOption::GasPriceMultiplier(10.0);
    for _ in 0..35 {
        let mut fs = vec![];
        for user in &evm_user_keys {
            fs.push(web3.erc20_approve(
                erc20_contracts[0],
                Uint256::max_value(),
                user.eth_privkey,
                erc20_contracts[1],
                Some(Duration::from_secs(15)),
                vec![gas_multiplier.clone()],
            ));
        }

        join_all(fs).await;
    }

    sleep(Duration::from_secs(10)).await;
    let post_supply = contact
        .query_supply_of(STAKING_TOKEN.clone())
        .await
        .expect("Unable to get aalthea supply")
        .expect("No supply of aalthea?");
    assert!(pre_supply.amount > post_supply.amount);
    info!(
        "Supply decreased by: {}",
        pre_supply.amount - post_supply.amount
    );
    let post_balances = snapshot_validator_rewards(contact, &validator_keys).await;
    assert_eq!(pre_balances, post_balances);

    // Check that other fees are not burned, and wind up in validator accounts
    let pre_balances = snapshot_validator_rewards(contact, &validator_keys).await;
    let mut fs = vec![];
    for user in &evm_user_keys {
        fs.push(contact.send_coins(
            Coin {
                amount: one_atom(),
                denom: STAKING_TOKEN.clone(),
            },
            Some(Coin {
                amount: one_atom() * 100u8.into(),
                denom: STAKING_TOKEN.clone(),
            }),
            evm_user_keys[0].ethermint_address,
            Some(Duration::from_secs(20)),
            user.ethermint_key,
        ));
    }
    join_all(fs).await;
    // info!("Results: {:?}", results);
    sleep(Duration::from_secs(10)).await;
    let post_balances = snapshot_validator_rewards(contact, &validator_keys).await;
    let althea_pre: Vec<Option<&DecCoin>> = pre_balances.iter().map(|v| v.iter().find(|d| &d.denom == &*STAKING_TOKEN)).collect();
    let althea_post: Vec<Option<&DecCoin>> = post_balances.iter().map(|v| v.iter().find(|d| &d.denom == &*STAKING_TOKEN)).collect();
    info!("Pre: {althea_pre:?}, Post: {althea_post:?}");
    for (pre, post) in althea_pre.iter().zip(althea_post.iter()) {
        match (pre, post) {
            (Some(althea_rewards_pre), Some(althea_rewards_post)) => {
                let rewards_pre: Uint256 = althea_rewards_pre.amount.parse().expect("Invalid integer?");
                let rewards_post: Uint256 = althea_rewards_post.amount.parse().expect("Invalid integer?");
                assert!(rewards_pre < rewards_post, "Validator staking token rewards did not increase");
            },
            _ => panic!("Validator missing staking token rewards"),
        }
    }

    info!("Successfully tested EVM fee burning");
}

pub async fn set_inflation_to_zero(contact: &Contact, validator_keys: &[ValidatorKeys]) {
    let to_change = vec![
        ParamChange {
            subspace: "mint".to_string(),
            key: "InflationMax".to_string(),
            value: "\"0.0\"".to_string(),
        },
        ParamChange {
            subspace: "mint".to_string(),
            key: "InflationMin".to_string(),
            value: "\"0.0\"".to_string(),
        },
    ];

    let proposer = validator_keys.first().unwrap();
    create_parameter_change_proposal(contact, proposer.validator_key, to_change, get_fee(None))
        .await;

    vote_yes_on_proposals(contact, validator_keys, Some(OPERATION_TIMEOUT)).await;
    wait_for_proposals_to_execute(contact).await;
}

pub async fn snapshot_validator_rewards(
    contact: &Contact,
    validator_keys: &[ValidatorKeys],
) -> Vec<Vec<DecCoin>> {
    let mut grpc = DistributionQueryClient::connect(contact.get_url())
        .await
        .unwrap();
    let mut rewards = Vec::new();
    for key in validator_keys {
        let reward = grpc
            .validator_outstanding_rewards(QueryValidatorOutstandingRewardsRequest {
                validator_address: key
                    .validator_key
                    .to_address("altheavaloper")
                    .unwrap()
                    .to_string(),
            })
            .await
            .expect("Unable to get rewards");
        rewards.push(reward.into_inner().rewards.expect("No rewards").rewards);
    }
    rewards
}
