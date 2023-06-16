use deep_space::{Coin, Contact, PrivateKey};
use web30::client::Web3;

use crate::utils::{
    get_ethermint_key, one_eth, ValidatorKeys, ADDRESS_PREFIX, OPERATION_TIMEOUT, STAKING_TOKEN,
};

/// A simple test to assert the EVM is correctly configured with the STAKING_TOKEN (aalthea) as the EVM denom.
/// WARNING: This test is used in CI as the first test run, keeping this test simple is best for CI runtimes
pub async fn native_token_test(
    contact: &Contact,
    web30: &Web3,
    validator_keys: Vec<ValidatorKeys>,
) {
    info!("Starting Native Token test");
    let mut users = vec![];
    for ValidatorKeys {
        validator_key: key,
        validator_phrase: _,
    } in validator_keys
    {
        let user = get_ethermint_key(Some(ADDRESS_PREFIX.as_str()));
        users.push(user);

        let val_addr = key.to_address(ADDRESS_PREFIX.as_str()).unwrap();
        info!(
            "Validator {val_addr} balances: {:?}",
            contact.get_balances(val_addr).await
        );
        let send_amount = Coin {
            amount: one_eth() * 100u8.into(),
            denom: STAKING_TOKEN.to_string(),
        };
        let fee_amount = Coin {
            amount: 10u8.into(),
            denom: STAKING_TOKEN.to_string(),
        };
        contact
            .send_coins(
                send_amount,
                Some(fee_amount),
                user.ethermint_address,
                Some(OPERATION_TIMEOUT),
                key,
            )
            .await
            .expect("Failed to send tokens to user");

        let user_cos_bal = contact
            .get_balance(user.ethermint_address, STAKING_TOKEN.to_string())
            .await;
        info!(
            "User ({} aka {}) balances: {:?}",
            user.ethermint_address, user.eth_address, user_cos_bal
        );

        let user_eth_bal = web30.eth_get_balance(user.eth_address).await;
        info!(
            "User eth ({} aka {}) ethereum balances: {:?}",
            user.ethermint_address, user.eth_address, user_eth_bal
        );

        let cosmos = user_cos_bal
            .expect("Failed to get cosmos balance")
            .expect("User has no aalthea balance?");
        let ethereum = user_eth_bal.expect("Failed to get eth balance");
        assert!(
            cosmos.amount.eq(&ethereum),
            "cosmos and ethereum balances do not match!"
        );
    }
}
