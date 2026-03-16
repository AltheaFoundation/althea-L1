use crate::type_urls::COMMUNITY_POOL_SPEND_PROPOSAL_TYPE_URL;
use crate::utils::{
    encode_any, get_fee, get_user_key, one_atom, vote_yes_on_proposals,
    wait_for_proposals_to_execute, ValidatorKeys, EVM_USER_KEYS, OPERATION_TIMEOUT, STAKING_TOKEN,
};
use althea_proto::cosmos_sdk_proto::cosmos::base::v1beta1::Coin as ProtoCoin;
use althea_proto::cosmos_sdk_proto::cosmos::distribution::v1beta1::CommunityPoolSpendProposal;
use clarity::Address as EthAddress;
use deep_space::address::Address as CosmosAddress;
use deep_space::{Coin, Contact};
use web30::client::Web3;
use web30::types::TransactionRequest;

pub async fn gov_spend_test(
    contact: &Contact,
    web30: &Web3,
    keys: Vec<ValidatorKeys>,
    gov_spend_test_contract: EthAddress,
) {
    info!("Starting Community Pool Spend to Contract test");
    let querier = EVM_USER_KEYS.first().unwrap();
    let querier_addr = querier.eth_address;

    // Create a new test account to receive the withdrawn funds
    let recipient_key = get_user_key(None);
    let recipient_address = recipient_key.eth_address;
    info!("Using recipient address: {}", recipient_address);

    // Get initial balance of the recipient
    let initial_recipient_balance = web30
        .eth_get_balance(recipient_address)
        .await
        .expect("Failed to get initial recipient balance");
    info!("Initial recipient balance: {}", initial_recipient_balance);

    // Convert the contract address to a Cosmos address for the proposal
    let contract_cosmos_address =
        CosmosAddress::from_slice(gov_spend_test_contract.as_bytes(), contact.get_prefix())
            .expect("Failed to convert contract address to cosmos address");

    let contract_balance_eth_before = web30
        .eth_get_balance(gov_spend_test_contract)
        .await
        .expect("Failed to get contract balance via eth_getBalance");

    let proposal_amount = one_atom() * 1000u32.into();

    info!(
        "Creating CommunintyPoolSpendProposal to send {} to contract at cosmos address {}",
        proposal_amount, contract_cosmos_address,
    );
    let proposal = CommunityPoolSpendProposal {
        title: "Community Pool Spend To Contract Test".to_string(),
        description: "Community Pool Spend To Contract Test".to_string(),
        recipient: contract_cosmos_address
            .to_bech32(contact.get_prefix())
            .unwrap(),
        amount: vec![ProtoCoin {
            denom: STAKING_TOKEN.clone(),
            amount: proposal_amount.to_string(),
        }],
    };

    let deposit = Coin {
        amount: one_atom() * 100u8.into(),
        denom: STAKING_TOKEN.clone(),
    };

    let any = encode_any(proposal, COMMUNITY_POOL_SPEND_PROPOSAL_TYPE_URL.to_string());
    let res = contact
        .create_legacy_gov_proposal(
            any,
            deposit,
            get_fee(None),
            keys[0].validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    info!("CommunityPoolSpendProposal submitted: {:?}", res);

    vote_yes_on_proposals(contact, &keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("CommunityPoolSpendProposal executed");

    let contract_balance_eth = web30
        .eth_get_balance(gov_spend_test_contract)
        .await
        .expect("Failed to get contract balance via eth_getBalance");
    info!(
        "Contract balance (eth_getBalance): {} -> {}",
        contract_balance_eth_before, contract_balance_eth
    );

    assert!(
        contract_balance_eth >= proposal_amount
            && contract_balance_eth >= contract_balance_eth_before,
        "Contract should have received funds via proposal, expected at least {}, got {}",
        proposal_amount,
        contract_balance_eth
    );

    assert!(
        contract_balance_eth == contract_balance_eth_before + proposal_amount,
        "Contract balance should have increased by proposal amount {}, expected {}, got {}",
        proposal_amount,
        contract_balance_eth_before + proposal_amount,
        contract_balance_eth
    );

    let get_balance_call =
        clarity::abi::encode_call("getBalance()", &[]).expect("Failed to encode getBalance() call");

    let contract_balance_call = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(querier_addr, gov_spend_test_contract, get_balance_call),
            vec![],
            None,
        )
        .await
        .expect("Failed to call getBalance");

    let contract_balance_from_call = clarity::Uint256::from_be_bytes(&contract_balance_call[0..32]);
    info!(
        "Contract balance (getBalance): {}",
        contract_balance_from_call
    );

    assert_eq!(
        contract_balance_eth, contract_balance_from_call,
        "eth_getBalance and getBalance() should return the same value"
    );

    let withdraw_all_payload =
        clarity::abi::encode_call("withdrawAll(address)", &[recipient_address.into()])
            .expect("Failed to encode withdrawAll call");

    let tx_id = web30
        .send_prepared_transaction(
            web30
                .prepare_transaction(
                    gov_spend_test_contract,
                    withdraw_all_payload,
                    0u32.into(),
                    querier.eth_privkey,
                    vec![],
                )
                .await
                .expect("Failed to prepare withdrawAll transaction"),
        )
        .await
        .expect("Failed to send withdrawAll transaction");
    info!(
        "withdrawAll transaction sent: 0x{:?}",
        clarity::utils::bytes_to_hex_str(&tx_id.to_be_bytes())
    );

    web30
        .wait_for_transaction(tx_id, OPERATION_TIMEOUT, None)
        .await
        .expect("withdrawAll transaction failed");

    let final_recipient_balance = web30
        .eth_get_balance(recipient_address)
        .await
        .expect("Failed to get final recipient balance");
    info!("Final recipient balance: {}", final_recipient_balance);

    assert!(
        final_recipient_balance > initial_recipient_balance,
        "Recipient balance should have increased after withdrawAll"
    );
    assert_eq!(
        final_recipient_balance,
        initial_recipient_balance + contract_balance_eth,
        "Recipient balance should have increased by the contract balance amount"
    );

    // Verify contract balance is now zero or near zero
    let final_contract_balance = web30
        .eth_get_balance(gov_spend_test_contract)
        .await
        .expect("Failed to get final contract balance");
    info!("Final contract balance: {}", final_contract_balance);

    assert!(
        final_contract_balance == 0u32.into(),
        "Contract balance should be zero after withdrawAll, got {}",
        final_contract_balance
    );

    info!("Community Pool Spend to Contract test completed successfully!");
}
