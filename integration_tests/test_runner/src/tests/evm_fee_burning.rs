use std::time::Duration;

use crate::utils::{
    create_parameter_change_proposal, one_atom, vote_yes_on_proposals,
    wait_for_proposals_to_execute, EthermintUserKey, ValidatorKeys, OPERATION_TIMEOUT,
    STAKING_TOKEN,
};
use althea_proto::cosmos_sdk_proto::cosmos::bank::v1beta1::query_client::QueryClient as BankQueryClient;
use althea_proto::cosmos_sdk_proto::cosmos::bank::v1beta1::QueryAllBalancesRequest;
use althea_proto::cosmos_sdk_proto::cosmos::base::v1beta1::DecCoin;
use althea_proto::cosmos_sdk_proto::cosmos::distribution::v1beta1::{
    query_client::QueryClient as DistributionQueryClient, QueryValidatorOutstandingRewardsRequest,
};
use althea_proto::cosmos_sdk_proto::cosmos::params::v1beta1::ParamChange;
use clarity::Address as EthAddress;
use deep_space::{Coin, Contact, PrivateKey};
use futures::future::join_all;
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
    let pre_rewards = snapshot_validator_rewards(contact, &validator_keys).await;

    info!("Generating some fees");
    let gas_multiplier = web30::types::SendTxOption::GasPriceMultiplier(10.0);
    for _ in 0..35 {
        let mut fs = vec![];
        for user in &evm_user_keys {
            fs.push(web3.approve_erc20_transfers(
                erc20_contracts[0],
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
    let post_rewards = snapshot_validator_rewards(contact, &validator_keys).await;
    assert_eq!(pre_rewards, post_rewards);

    // Check that other fees are not burned, and wind up in validator accounts
    let pre_rewards = snapshot_validator_rewards(contact, &validator_keys).await;
    let pre_balances = snapshot_validator_balances(contact, &validator_keys).await;
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
    let post_rewards = snapshot_validator_rewards(contact, &validator_keys).await;
    info!("Rewards - Pre: {:?}, Post: {:?}", pre_rewards, post_rewards);
    let mut reward_increase: bool = false;
    for (pre, post) in pre_rewards.iter().zip(post_rewards.iter()) {
        for (pr, po) in pre.iter().zip(post.iter()) {
            assert_eq!(pr.denom, po.denom);
            let pree: Uint256 = pr.amount.parse().expect("Invalid integer?");
            let postt: Uint256 = po.amount.parse().expect("Invalid integer?");
            if pree < postt {
                reward_increase = true;
            }
        }
    }
    let post_balances = snapshot_validator_balances(contact, &validator_keys).await;
    info!(
        "Balances - Pre: {:?}, Post: {:?}",
        pre_balances, post_balances
    );
    let mut balance_increase: bool = false;
    for (pre, post) in pre_balances.iter().zip(post_balances.iter()) {
        for (pr, po) in pre.iter().zip(post.iter()) {
            assert_eq!(pr.denom, po.denom);
            if pr.amount < po.amount {
                balance_increase = true;
            }
        }
    }

    assert!(
        reward_increase || balance_increase,
        "Neither the rewards nor the balances increased for any of the validators"
    );
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
    let zero_fee = Coin {
        denom: STAKING_TOKEN.clone(),
        amount: 0u8.into(),
    };
    create_parameter_change_proposal(contact, proposer.validator_key, to_change, zero_fee).await;

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

pub async fn snapshot_validator_balances(
    contact: &Contact,
    validator_keys: &[ValidatorKeys],
) -> Vec<Vec<Coin>> {
    let mut grpc = BankQueryClient::connect(contact.get_url()).await.unwrap();
    let mut balances: Vec<Vec<Coin>> = Vec::new();
    for key in validator_keys {
        let balance = grpc
            .all_balances(QueryAllBalancesRequest {
                address: key.validator_key.to_address("althea").unwrap().to_string(),
                pagination: None,
            })
            .await
            .expect("Unable to get validator balances");
        balances.push(
            balance
                .into_inner()
                .balances
                .iter()
                .map(|x| Coin {
                    amount: x.amount.parse().expect("Invalid amount?"),
                    denom: x.denom.clone(),
                })
                .collect(),
        );
    }
    balances
}
