//! Contains ERC20 <> Cosmos Coin tests for the Canto erc20 module over at https://github.com/Canto-Network/Canto

use std::str::FromStr;

use crate::bootstrapping::STAKING_TOKEN;
use crate::type_urls::{MSG_CONVERT_COIN_TYPE_URL, MSG_CONVERT_ERC20_TYPE_URL};
use crate::utils::{
    cosmos_address_to_eth_address, execute_register_coin_proposal, execute_register_erc20_proposal,
    footoken_metadata, get_module_account_address, one_atom, one_eth, EthermintUserKey,
    RegisterCoinProposalParams, RegisterErc20ProposalParams, ValidatorKeys, OPERATION_TIMEOUT,
    TOTAL_TIMEOUT,
};
use althea_proto::canto::erc20::v1::query_client::QueryClient as Erc20QueryClient;
use althea_proto::canto::erc20::v1::{MsgConvertCoin, MsgConvertErc20, QueryTokenPairRequest};
use althea_proto::cosmos_sdk_proto::cosmos::bank::v1beta1::Metadata;
use althea_proto::cosmos_sdk_proto::cosmos::base::v1beta1::Coin as ProtoCoin;
use clarity::{Address as EthAddress, Uint256};
use deep_space::error::CosmosGrpcError;
use deep_space::{Coin, Contact, Msg};
use tonic::transport::Channel;
use web30::client::Web3;

const SKIP_GOV: bool = false; // skip the gov proposal if debugging the test

/// Tests the hidden links that the erc20 module provides between the Ethermint EVM and Cosmos coins by
/// executing a RegisterErc20 gov proposal for one of the test contracts, migrating the test ERC20 out of the
/// EVM-space into the x/bank-space, performing a funds transfer via x/bank MsgSend, then sending those funds back
/// into the EVM-space
pub async fn erc20_conversion_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: Vec<ValidatorKeys>,
    evm_user_keys: Vec<EthermintUserKey>,
    erc20_contracts: Vec<EthAddress>,
) {
    erc20_register_and_round_trip_test(
        contact,
        web3,
        validator_keys.clone(),
        evm_user_keys.clone(),
        erc20_contracts.clone(),
    )
    .await;
    coin_register_and_round_trip_test(
        contact,
        web3,
        validator_keys.clone(),
        evm_user_keys.clone(),
        footoken_metadata(contact).await,
    )
    .await;
    info!("The whole ERC20_CONVERSION test was successful!");
}

/// Tests EVM-only ERC20 conversion, including RegisterERC20 proposal execution and EVM -> Cosmos -> EVM round trip transfers
pub async fn erc20_register_and_round_trip_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: Vec<ValidatorKeys>,
    evm_user_keys: Vec<EthermintUserKey>,
    erc20_contracts: Vec<EthAddress>,
) {
    // Register an ERC20 in the EVM to a new Cosmos Coin controlled by the bank module
    let registered_erc20 = erc20_contracts
        .get(0)
        .expect("No ERC20 contracts passed to erc20 happy path test?");

    // TODO: Test unregistered conversion failure
    let _unregistered_erc20 = erc20_contracts.get(1).expect(
        "Not enough ERC20 contracts passed to erc20 happy path test? Need at least 2 contracts.",
    );
    let recvr = evm_user_keys.get(0).expect("No EVM users?");
    let transfer_recvr = evm_user_keys.get(1).expect("fewer than 2 evm users?");
    let start_evm_balance = web3
        .get_erc20_balance(*registered_erc20, transfer_recvr.eth_address)
        .await;

    let erc20_params = RegisterErc20ProposalParams {
        erc20_address: registered_erc20.to_string(),
        proposal_desc: "Register ERC20 Proposal Description".to_string(),
        proposal_title: "Register ERC20 Proposal Title".to_string(),
    };
    if !SKIP_GOV {
        execute_register_erc20_proposal(
            contact,
            &validator_keys,
            Some(TOTAL_TIMEOUT),
            erc20_params,
        )
        .await;
    } else {
        info!("Skipping RegisterErc20 governance process because of SKIP_GOV constant value ({SKIP_GOV})");
    }

    // Test out using the new Cosmos Coin by converting some ERC20, checking the balance, and sending it to another address,
    // finally converting it back into the original ERC20

    // Give the evm user some aalthea tokens to avoid errors
    contact
        .send_coins(
            Coin {
                amount: one_atom(),
                denom: STAKING_TOKEN.to_string(),
            },
            Some(Coin {
                amount: 0u8.into(),
                denom: STAKING_TOKEN.to_string(),
            }),
            recvr.ethermint_address,
            Some(OPERATION_TIMEOUT),
            validator_keys.get(0).unwrap().validator_key,
        )
        .await
        .expect("Unable to send evm user any {STAKING_TOKEN}");

    // First try to send far too many tokens to cosmos and expect error
    let bad_amount: Uint256 = one_eth() * 10000u32.into();
    let bad_erc20_to_cosmos_msg = MsgConvertErc20 {
        amount: bad_amount.to_string(),
        contract_address: registered_erc20.to_string(),
        receiver: recvr.ethermint_address.to_string(),
        sender: recvr.eth_address.to_string(),
    };
    info!("Planning to fail this MsgConvertErc20: {bad_erc20_to_cosmos_msg:?}");
    let msg = Msg::new(MSG_CONVERT_ERC20_TYPE_URL, bad_erc20_to_cosmos_msg);

    let res = contact
        .send_message(
            &[msg],
            None,
            &[Coin {
                // fee coin
                amount: 100u8.into(),
                denom: STAKING_TOKEN.clone(),
            }],
            Some(OPERATION_TIMEOUT),
            recvr.ethermint_key,
        )
        .await;
    res.expect_err("Expected failure when sending too many ERC20 tokens to x/bank");

    // The evm user has some erc20 tokens already, they are going to convert them and send to their cosmos account
    let convert_amount: Uint256 = 100u8.into();
    let erc20_to_cosmos_msg = MsgConvertErc20 {
        amount: convert_amount.to_string(),
        contract_address: registered_erc20.to_string(),
        receiver: recvr.ethermint_address.to_string(),
        sender: recvr.eth_address.to_string(),
    };
    info!("Planning to send this MsgConvertErc20: {erc20_to_cosmos_msg:?}");
    let msg = Msg::new(MSG_CONVERT_ERC20_TYPE_URL, erc20_to_cosmos_msg);

    let res = contact
        .send_message(
            &[msg],
            None,
            &[Coin {
                // fee coin
                amount: 100u8.into(),
                denom: STAKING_TOKEN.clone(),
            }],
            Some(OPERATION_TIMEOUT),
            recvr.ethermint_key,
        )
        .await;
    res.expect("Expected success when sending ERC20 to x/bank");

    // Check that the balance changed
    info!("Successfully executed ERC20 conversion message");
    let balances = contact.get_balances(recvr.ethermint_address).await;
    info!(
        "Account balances after ERC20 conversion execution: {:?}",
        balances
    );
    let expected_erc20_denom = format!("erc20/{}", registered_erc20.to_string());

    // The linter is annoying here, I cannot actually inline those format args into the string
    assert!(
        balances_contains_coin(
            balances.unwrap(),
            Coin {
                amount: convert_amount,
                denom: expected_erc20_denom.clone()
            },
            true
        ),
        "expected at least {}{} but did not find it",
        convert_amount,
        expected_erc20_denom,
    );

    // Set up a user to receive these tokens via the bank module's transfer process
    let res = contact
        .send_coins(
            Coin {
                amount: convert_amount,
                denom: expected_erc20_denom.clone(),
            },
            Some(Coin {
                // fee coin
                amount: 100u8.into(),
                denom: STAKING_TOKEN.clone(),
            }),
            transfer_recvr.ethermint_address,
            Some(OPERATION_TIMEOUT),
            recvr.ethermint_key,
        )
        .await;
    info!("Sent {expected_erc20_denom} to transfer receiver, got response {res:?}");

    let balances = contact.get_balances(transfer_recvr.ethermint_address).await;
    info!(
        "Account balances after sending ERC20 via x/bank: {:?}",
        balances
    );

    // The linter is annoying here, I cannot actually inline those format args into the string
    assert!(
        balances_contains_coin(
            balances.unwrap(),
            Coin {
                amount: convert_amount,
                denom: expected_erc20_denom.clone()
            },
            true
        ),
        "expected at least {}{} but did not find it",
        convert_amount,
        expected_erc20_denom,
    );

    // First, try to transfer far too many tokens the account does not have
    let bad_balance = one_eth() * 1000u32.into();
    let bad_cosmos_to_erc20_msg = MsgConvertCoin {
        coin: Some(ProtoCoin {
            amount: bad_balance.to_string(),
            denom: expected_erc20_denom.clone(),
        }),
        receiver: transfer_recvr.eth_address.to_string(),
        sender: transfer_recvr.ethermint_address.to_string(),
    };
    info!("Planning to fail this MsgConvertCoin: {bad_cosmos_to_erc20_msg:?}");
    let msg = Msg::new(MSG_CONVERT_COIN_TYPE_URL, bad_cosmos_to_erc20_msg);

    let res = contact
        .send_message(
            &[msg],
            None,
            &[Coin {
                // fee coin
                amount: 100u8.into(),
                denom: STAKING_TOKEN.clone(),
            }],
            Some(OPERATION_TIMEOUT),
            transfer_recvr.ethermint_key,
        )
        .await;
    res.expect_err("Expected failure when sending too many tokens into the EVM");

    // Have the transfer_recvr user send them into the EVM and check the balance changes
    let cosmos_to_erc20_msg = MsgConvertCoin {
        coin: Some(ProtoCoin {
            amount: convert_amount.to_string(),
            denom: expected_erc20_denom,
        }),
        receiver: transfer_recvr.eth_address.to_string(),
        sender: transfer_recvr.ethermint_address.to_string(),
    };
    info!("Planning to send this MsgConvertCoin: {cosmos_to_erc20_msg:?}");
    let msg = Msg::new(MSG_CONVERT_COIN_TYPE_URL, cosmos_to_erc20_msg);

    let res = contact
        .send_message(
            &[msg],
            None,
            &[Coin {
                // fee coin
                amount: 100u8.into(),
                denom: STAKING_TOKEN.clone(),
            }],
            Some(OPERATION_TIMEOUT),
            transfer_recvr.ethermint_key,
        )
        .await;
    res.expect("Expected success when sending ERC20 back to EVM");

    let end_evm_balance = web3
        .get_erc20_balance(*registered_erc20, transfer_recvr.eth_address)
        .await;
    let evm_balance = match (start_evm_balance, end_evm_balance) {
        (Err(_), Ok(v)) => v,
        (Ok(s), Ok(e)) => {
            assert!(e >= s, "user {} somehow lost the registered erc20 over the erc20 conversion test (end {} < start {})", transfer_recvr.eth_address, e, s);
            e - s
        }
        (_, _) => {
            panic!(
                "Unexpected balance changes for user {} during erc20 conversion test",
                transfer_recvr.eth_address
            );
        }
    };
    // The linter is annoying here, I cannot actually inline those format args into the string
    assert!(
        evm_balance >= convert_amount,
        "unexpected balance {} after MsgConvertCoin, expected at least {}",
        evm_balance,
        convert_amount,
    );

    info!("Successful x/evm Convert ERC20 -> x/bank MsgSend -> x/evm Convert Coin execution sequence!");
}

/// Returns true if `balances` contains `coin`, optionally greater than too if gte = true
pub fn balances_contains_coin(balances: Vec<Coin>, coin: Coin, gte: bool) -> bool {
    for c in balances {
        if c.denom == coin.denom && (c.amount == coin.amount || (gte && c.amount >= coin.amount)) {
            return true;
        }
    }
    if coin.amount == 0u8.into() {
        // Not finding a 0 coin is desired
        return true;
    }

    false
}

/// Tests Cosmos Coin -> EVM ERC20 conversion, including RegisterCoinProposal execution and Cosmos -> EVM -> Cosmos
/// round trip transfers
pub async fn coin_register_and_round_trip_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: Vec<ValidatorKeys>,
    evm_user_keys: Vec<EthermintUserKey>,
    cosmos_coin: Metadata,
) {
    let grpc = Erc20QueryClient::connect(contact.get_url())
        .await
        .expect("could not connect to ERC20 query client");
    let registered_denom = cosmos_coin.base.clone();
    let recvr = evm_user_keys.get(0).expect("No EVM users?");
    let transfer_recvr = evm_user_keys.get(1).expect("fewer than 2 evm users?");
    let erc20_module_eth_address = cosmos_address_to_eth_address(
        get_module_account_address("erc20", None).expect("unable to get erc20 module address"),
    )
    .expect("unable to convert erc20 module address to ethereum address");
    let start_coin_balance = contact
        .get_balance(
            transfer_recvr.ethermint_address,
            registered_denom.to_string(),
        )
        .await;
    let start_coin_balance = start_coin_balance
        .unwrap()
        .expect("EVM user key does not have any {registered_denom}!");

    let coin_params = RegisterCoinProposalParams {
        coin_metadata: cosmos_coin.clone(),
        proposal_desc: "Register Coin Proposal Description".to_string(),
        proposal_title: "Register Coin Proposal Title".to_string(),
    };
    if !SKIP_GOV {
        execute_register_coin_proposal(contact, &validator_keys, Some(TOTAL_TIMEOUT), coin_params)
            .await;
    } else {
        info!("Skipping RegisterCoin governance process because of SKIP_GOV constant value ({SKIP_GOV})");
    }

    // Test out using the new ERC20 by converting some Coin, checking the balance, and sending it to another address,
    // finally converting it back into the original Coin

    // Give the evm user some of the coin for the test
    contact
        .send_coins(
            Coin {
                amount: one_atom(),
                denom: registered_denom.to_string(),
            },
            Some(Coin {
                amount: 0u8.into(),
                denom: STAKING_TOKEN.to_string(),
            }),
            recvr.ethermint_address,
            Some(OPERATION_TIMEOUT),
            validator_keys.get(0).unwrap().validator_key,
        )
        .await
        .expect("Unable to send evm user any {STAKING_TOKEN}");

    // Send to the EVM via x/erc20 MsgConvertCoin
    let convert_amount: Uint256 = 100u8.into();
    let cosmos_to_erc20_msg = MsgConvertCoin {
        coin: Some(ProtoCoin {
            amount: convert_amount.to_string(),
            denom: registered_denom.to_string(),
        }),
        receiver: recvr.eth_address.to_string(),
        sender: recvr.ethermint_address.to_string(),
    };
    info!("Planning to send this MsgConvertCoin: {cosmos_to_erc20_msg:?}");
    let msg = Msg::new(MSG_CONVERT_COIN_TYPE_URL, cosmos_to_erc20_msg);

    let res = contact
        .send_message(
            &[msg],
            None,
            &[Coin {
                // fee coin
                amount: 100u8.into(),
                denom: STAKING_TOKEN.clone(),
            }],
            Some(OPERATION_TIMEOUT),
            recvr.ethermint_key,
        )
        .await;
    res.expect("Expected success when sending Coin to x/evm");
    info!("Successfully executed Coin conversion message");

    // Learn the generated ERC20's address
    let generated_erc20 =
        find_register_coin_generated_erc20_address(grpc, registered_denom.clone())
            .await
            .unwrap();

    // Check that the balance changed
    let balance = web3
        .get_erc20_balance(generated_erc20, recvr.eth_address)
        .await;
    info!(
        "Account balances after Coin conversion execution: {:?}",
        balance
    );

    // The linter is annoying here, I cannot actually inline those format args into the string
    assert!(
        balance.unwrap() >= convert_amount,
        "expected at least {}{} but did not find it",
        convert_amount,
        generated_erc20,
    );

    // Transfer from recvr->transfer_recvr via ERC20 transfer() process
    let res = web3
        .erc20_send(
            convert_amount,
            transfer_recvr.eth_address,
            generated_erc20,
            recvr.eth_privkey,
            Some(OPERATION_TIMEOUT),
            vec![],
        )
        .await;
    res.expect("Expected success when sending ERC20 within the EVM");

    // Check that the balance changed
    info!("Successfully called erc20 transfer()");
    let balance = web3
        .get_erc20_balance(generated_erc20, transfer_recvr.eth_address)
        .await;
    info!("Account balance after erc20 transfer(): {:?}", balance);

    // The linter is annoying here, I cannot actually inline those format args into the string
    assert!(
        balance.unwrap() >= convert_amount,
        "expected at least {}{} but did not find it",
        convert_amount,
        generated_erc20,
    );

    // Send the tokens back to cosmos via the module's ethereum address
    // There is a special hook registered to detect any erc20 transfers and automatically
    // convert these ERC20s if they are registered and send them to the originator's cosmos account

    let res = web3
        .erc20_send(
            convert_amount,
            erc20_module_eth_address,
            generated_erc20,
            transfer_recvr.eth_privkey,
            Some(OPERATION_TIMEOUT),
            vec![],
        )
        .await;

    info!("Sent {convert_amount}{generated_erc20} to erc20 module {erc20_module_eth_address}, got response {res:?}");

    let balances = contact.get_balances(transfer_recvr.ethermint_address).await;
    info!(
        "Account balances after sending ERC20 via x/bank: {:?}",
        balances
    );

    // The linter is annoying here, I cannot actually inline those format args into the string
    assert!(
        balances_contains_coin(balances.unwrap(), start_coin_balance.clone(), true),
        "expected to end with starting balance",
    );

    info!("Successful x/evm Convert ERC20 -> x/bank MsgSend -> x/evm Convert Coin execution sequence!");
}

/// Queries the Canto x/erc20 module's TokenPair endpoint, returning a parsed EthAddress from a successful response
pub async fn find_register_coin_generated_erc20_address(
    grpc: Erc20QueryClient<Channel>,
    token: String,
) -> Result<EthAddress, CosmosGrpcError> {
    let mut grpc = grpc;
    let pair = grpc
        .token_pair(QueryTokenPairRequest {
            token: token.clone(),
        })
        .await?
        .into_inner()
        .token_pair;
    let pair = pair.ok_or(CosmosGrpcError::BadResponse(format!(
        "no pair for token {token}"
    )))?;
    if !pair.enabled {
        return Err(CosmosGrpcError::BadResponse(
            "pair is not enabled!".to_string(),
        ));
    }
    if pair.denom != token {
        return Err(CosmosGrpcError::BadResponse(format!(
            "pair returned does not have queried denom ({token} != {})!",
            pair.denom
        )));
    }

    EthAddress::from_str(&pair.erc20_address)
        .map_err(|e| CosmosGrpcError::BadResponse(format!("invalid erc20 address returned: {e}")))
}
