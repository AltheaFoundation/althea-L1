use crate::type_urls::EXECUTE_CONTRACT_PROPOSAL_TYPE_URL;
use crate::utils::{
    encode_any, get_fee, one_atom, one_eth, vote_yes_on_proposals, wait_for_proposals_to_execute,
    ValidatorKeys, ADDRESS_PREFIX, EVM_USER_KEYS, MINER_PRIVATE_KEY, OPERATION_TIMEOUT,
    STAKING_TOKEN,
};
use althea_proto::althea::nativedex::v1::{ExecuteContractMetadata, ExecuteContractProposal};
use clarity::Address as EthAddress;
use deep_space::address::get_module_account_address;
use deep_space::{Coin, Contact};
use web30::client::Web3;

pub async fn execute_contract_proposal_test(
    contact: &Contact,
    web30: &Web3,
    keys: Vec<ValidatorKeys>,
    erc20_addresses: Vec<EthAddress>,
) {
    info!("Starting execute contract proposal test");

    // Get an ERC20 contract address to use
    let erc20_contract = erc20_addresses[0];
    info!("Using ERC20 contract: {}", erc20_contract);
    let querier = EVM_USER_KEYS
        .first()
        .expect("No EVM user keys available")
        .eth_address;

    // Get the nativedex module account address
    let nativedex_module_acc =
        get_module_account_address("nativedex", Some(ADDRESS_PREFIX.as_str()))
            .expect("Failed to get module account address");
    info!(
        "Nativedex module account (Cosmos): {}",
        nativedex_module_acc
    );

    // Convert the cosmos address to an EVM address
    // Module accounts in Cosmos are 20 bytes, same as Ethereum addresses
    // We get the bytes by matching on the Address enum
    let address_bytes = nativedex_module_acc.get_bytes();
    let nativedex_evm_address = EthAddress::from_slice(address_bytes)
        .expect("Failed to convert module account to EVM address");
    info!("Nativedex module account (EVM): {}", nativedex_evm_address);

    // Create a receiver address for the transfer
    let receiver_address =
        EthAddress::parse_and_validate("0x1111111111111111111111111111111111111111")
            .expect("Invalid receiver address");

    // Amount to transfer (1000 tokens with 18 decimals)
    let transfer_amount = one_eth() * 1000u32.into();

    // Transfer some ERC20 tokens to the nativedex module account
    info!(
        "Transferring {} tokens to nativedex module",
        transfer_amount
    );

    web30
        .erc20_send(
            transfer_amount,
            nativedex_evm_address,
            erc20_contract,
            *MINER_PRIVATE_KEY,
            Some(OPERATION_TIMEOUT),
            vec![],
        )
        .await
        .expect("Failed to transfer ERC20 to module account");

    // Verify the module received the tokens
    let module_balance = web30
        .get_erc20_balance_as_address(Some(querier), erc20_contract, nativedex_evm_address, vec![])
        .await
        .expect("Failed to get module balance");
    info!("Module balance after transfer: {}", module_balance);
    assert!(
        module_balance >= transfer_amount,
        "Module should have received tokens"
    );

    // Step 1: Whitelist the ERC20 contract address via parameter change proposal
    info!("Submitting parameter change proposal to whitelist ERC20 contract");
    let deposit = Coin {
        amount: one_atom() * 100u8.into(),
        denom: STAKING_TOKEN.clone(),
    };

    use althea_proto::cosmos_sdk_proto::cosmos::params::v1beta1::{
        ParamChange, ParameterChangeProposal,
    };

    let whitelist_value = format!("[\"{}\"]", erc20_contract);
    let res = contact
        .submit_parameter_change_proposal(
            ParameterChangeProposal {
                title: "Whitelist ERC20 contract".to_string(),
                description: "Whitelist ERC20 contract for ExecuteContractProposal".to_string(),
                changes: vec![ParamChange {
                    subspace: "nativedex".to_string(),
                    key: "WhitelistedContractAddresses".to_string(),
                    value: whitelist_value,
                }],
            },
            deposit.clone(),
            get_fee(None),
            keys[0].validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    info!("Parameter change proposal submitted: {:?}", res);

    vote_yes_on_proposals(contact, &keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("Whitelist proposal executed");

    // Step 2: Create and submit ExecuteContractProposal to transfer tokens
    info!("Creating ExecuteContractProposal to transfer tokens");

    // Encode the transfer function call: transfer(address,uint256)
    let transfer_data = clarity::abi::encode_call(
        "transfer(address,uint256)",
        &[receiver_address.into(), transfer_amount.into()],
    )
    .expect("Failed to encode transfer call");

    let hex_data = format!("0x{}", clarity::utils::bytes_to_hex_str(&transfer_data));
    info!("Encoded transfer data: {}", hex_data);

    let proposal = ExecuteContractProposal {
        title: "Transfer ERC20 tokens".to_string(),
        description: "Transfer tokens from nativedex module to receiver".to_string(),
        metadata: Some(ExecuteContractMetadata {
            contract_address: erc20_contract.to_string(),
            data: hex_data,
        }),
    };

    let any = encode_any(proposal, EXECUTE_CONTRACT_PROPOSAL_TYPE_URL.to_string());
    let res = contact
        .create_legacy_gov_proposal(
            any,
            deposit,
            get_fee(None),
            keys[0].validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;
    info!("ExecuteContractProposal submitted: {:?}", res);

    vote_yes_on_proposals(contact, &keys, None).await;
    wait_for_proposals_to_execute(contact).await;
    info!("ExecuteContractProposal executed");

    // Step 3: Verify the transfer was successful
    info!("Verifying token transfer");

    let receiver_balance = web30
        .get_erc20_balance_as_address(Some(querier), erc20_contract, receiver_address, vec![])
        .await
        .expect("Failed to get receiver balance");
    info!("Receiver balance after proposal: {}", receiver_balance);

    assert_eq!(
        receiver_balance, transfer_amount,
        "Receiver should have received the transferred tokens"
    );

    let module_balance_after = web30
        .get_erc20_balance_as_address(Some(querier), erc20_contract, nativedex_evm_address, vec![])
        .await
        .expect("Failed to get module balance");
    info!("Module balance after proposal: {}", module_balance_after);

    assert_eq!(
        module_balance_after,
        module_balance - transfer_amount,
        "Module balance should be reduced by transfer amount"
    );

    info!("Execute contract proposal test successful!");
}
