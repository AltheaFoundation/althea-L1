use std::thread::sleep;
use std::time::Duration;

use crate::ibc_utils::{get_channel, get_hash_for_denom_trace};
use crate::tests::onboarding::submit_and_pass_token_proposal;
use crate::type_urls::{
    MSG_CONVERT_COIN_TYPE_URL, MSG_CONVERT_ERC20_TYPE_URL, MSG_SEND_COIN_TO_EVM_TYPE_URL,
    MSG_SEND_ERC20_TO_COSMOS_AND_IBC_TRANSFER_TYPE_URL, MSG_SEND_ERC20_TO_COSMOS_TYPE_URL,
};
use crate::utils::{
    execute_register_coin_proposal, footoken_metadata, get_fee, get_fee_option, get_ibc_chain_id,
    get_user_key, one_atom, vote_yes_on_proposals, wait_for_proposals_to_execute,
    RegisterCoinProposalParams, ValidatorKeys, COSMOS_NODE_GRPC, IBC_ADDRESS_PREFIX, IBC_NODE_GRPC,
    OPERATION_TIMEOUT, STAKING_TOKEN, TOTAL_TIMEOUT,
};
use althea_proto::althea::erc20::v1::query_client::QueryClient as Erc20QueryClient;
use althea_proto::althea::erc20::v1::{
    MsgConvertCoin, MsgConvertErc20, MsgSendCoinToEvm, MsgSendErc20ToCosmos,
    MsgSendErc20ToCosmosAndIbcTransfer, QueryTokenPairRequest,
};
use althea_proto::althea::gasfree::v1::query_client::QueryClient as GasfreeQueryClient;
use althea_proto::althea::gasfree::v1::QueryParamsRequest;
use althea_proto::cosmos_sdk_proto::cosmos::bank::v1beta1::Metadata;
use althea_proto::cosmos_sdk_proto::cosmos::base::v1beta1::Coin as ProtoCoin;
use althea_proto::cosmos_sdk_proto::cosmos::params::v1beta1::{
    ParamChange, ParameterChangeProposal,
};
use althea_proto::cosmos_sdk_proto::ibc::applications::transfer::v1::query_client::QueryClient as IbcTransferQueryClient;
use althea_proto::cosmos_sdk_proto::ibc::core::channel::v1::query_client::QueryClient as IbcChannelQueryClient;
use clarity::{Address as EthAddress, Uint256};
use deep_space::client::type_urls::MSG_MICROTX_TYPE_URL;
use deep_space::{Coin, Contact, Msg, PrivateKey};
use num::Zero;
use tonic::transport::Channel;
use web30::client::Web3;

use crate::utils::EthermintUserKey;

/// Environment variable toggle to skip governance while iterating
const SKIP_GOV: bool = false;

// Tests each of the gasfree ERC20 module messages, which are MsgSendCoinToEvm, MsgSendErc20ToCosmos, and MsgSendErc20ToCosmosAndIbcTransfer
// Happy path tests are submitted, in addition to insufficient balance tests for each message which check that the message fails when the gasfree fee cannot be collected
pub async fn gasfree_erc20_interop_test(
    althea_contact: &Contact,
    ibc_contact: &Contact,
    web3: &Web3,
    validator_keys: Vec<ValidatorKeys>,
    evm_user_keys: Vec<EthermintUserKey>,
    erc20_contracts: Vec<EthAddress>,
    expect_failure: bool,
) {
    gasfree_send_coin_to_evm_test(
        althea_contact,
        web3,
        &validator_keys,
        &evm_user_keys,
        &erc20_contracts,
        expect_failure,
    )
    .await;
    gasfree_send_erc20_to_cosmos_test(
        althea_contact,
        web3,
        &validator_keys,
        &evm_user_keys,
        &erc20_contracts,
        expect_failure,
    )
    .await;
    gasfree_send_erc20_to_cosmos_and_ibc_transfer_test(
        althea_contact,
        ibc_contact,
        web3,
        &validator_keys,
        &evm_user_keys,
        &erc20_contracts,
        expect_failure,
    )
    .await;

    gasfree_send_coin_to_evm_insufficient_balance_test(
        althea_contact,
        web3,
        &validator_keys,
        &evm_user_keys,
        &erc20_contracts,
        expect_failure,
    )
    .await;
    gasfree_send_erc20_to_cosmos_insufficient_balance_test(
        althea_contact,
        web3,
        &validator_keys,
        &evm_user_keys,
        &erc20_contracts,
        expect_failure,
    )
    .await;
    gasfree_send_erc20_to_cosmos_and_ibc_transfer_insufficient_balance_test(
        althea_contact,
        web3,
        &validator_keys,
        &evm_user_keys,
        &erc20_contracts,
        expect_failure,
    )
    .await;

    info!("Gasfree ERC20 interop tests completed successfully");
}

// Tests that a cosmos native coin and an erc20 can be sent from cosmos to evm without paying a tx fee, instead
// the gasfree fee is collected from the converted token's balance on the cosmos side.
pub async fn gasfree_send_coin_to_evm_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: &[ValidatorKeys],
    evm_user_keys: &[EthermintUserKey],
    erc20_contracts: &[EthAddress],
    expect_failure: bool,
) {
    info!("Starting gasfree MsgSendCoinToEvm test");

    let fee_basis_points: u64 = 100;
    let amounts = random_splits_uint256(300_000, 3)
        .iter()
        .map(|v| *v * one_atom())
        .collect::<Vec<Uint256>>();
    let fee_amounts = amounts
        .iter()
        .map(|v| *v * fee_basis_points.into() / 10_000u32.into())
        .collect::<Vec<Uint256>>();
    let send_total = amounts.iter().fold(Uint256::default(), |a, v| a + *v);
    let fee_total = fee_amounts.iter().fold(Uint256::default(), |a, v| a + *v);

    let user = evm_user_keys.first().expect("No EVM users provided");
    let mut erc20_qc = Erc20QueryClient::connect(contact.get_url())
        .await
        .expect("could not connect to erc20 query client");
    // 1. Register a cosmos coin (footoken) if not already registered
    let footoken_meta = footoken_metadata(contact).await;
    let registered_denom = footoken_meta.base.clone();
    let registered_erc20 = erc20_contracts.first().unwrap().to_string();
    if !SKIP_GOV {
        register_coin_if_not_registered(
            erc20_qc.clone(),
            contact,
            validator_keys,
            &registered_denom,
            footoken_meta,
        )
        .await;
        submit_and_pass_token_proposal(
            contact,
            erc20_qc.clone(),
            validator_keys,
            registered_erc20.clone(),
            true,
        )
        .await;
    } else {
        info!("Skipping coin registration governance due to SKIP_GOV");
    }
    let denom_pair = erc20_qc
        .token_pair(QueryTokenPairRequest {
            token: registered_denom.clone(),
        })
        .await
        .expect("could not query token pair")
        .into_inner()
        .token_pair
        .expect("token pair response had no token pair");
    let denom_erc20_address = denom_pair
        .erc20_address
        .parse::<EthAddress>()
        .expect("failed to parse ERC20 address");
    let erc20_pair = erc20_qc
        .token_pair(QueryTokenPairRequest {
            token: registered_erc20.clone(),
        })
        .await
        .expect("could not query token pair")
        .into_inner()
        .token_pair
        .expect("token pair response had no token pair");
    let erc20_denom = erc20_pair.denom;

    if !SKIP_GOV {
        configure_gasfree_params_if_not_configured(
            contact,
            &[registered_denom.clone(), registered_erc20.clone()],
            fee_basis_points,
            validator_keys,
            expect_failure,
        )
        .await;
    } else {
        info!("Skipping gasfree parameter change governance due to SKIP_GOV");
    }

    let begin_bank_balance = contact
        .get_balance(user.ethermint_address, registered_denom.clone())
        .await
        .unwrap()
        .unwrap();
    if begin_bank_balance.amount < send_total + fee_total {
        // Fund the user with the cosmos coin to be sent
        contact
            .send_coins(
                Coin {
                    amount: send_total + fee_total,
                    denom: registered_denom.clone(),
                },
                get_fee_option(None),
                user.ethermint_address,
                Some(OPERATION_TIMEOUT),
                validator_keys.last().unwrap().validator_key,
            )
            .await
            .expect("Unable to fund user with test coin");
    }
    let begin_evm_balance = web3
        .get_erc20_balance(denom_erc20_address, user.eth_address)
        .await
        .expect("Could not query starting ERC20 balance");
    if begin_evm_balance < send_total + fee_total {
        // Fund the user with the erc20 token's cosmos representation to be sent
        let erc20_to_cosmos_msg = MsgConvertErc20 {
            amount: (send_total + fee_total).to_string(),
            contract_address: registered_erc20.to_string(),
            receiver: user.ethermint_address.to_string(),
            sender: user.eth_address.to_string(),
        };
        info!("Planning to send this MsgConvertErc20: {erc20_to_cosmos_msg:?}");
        let msg = Msg::new(MSG_CONVERT_ERC20_TYPE_URL, erc20_to_cosmos_msg);

        let _ = contact
            .send_message(
                &[msg],
                None,
                &[get_fee(None)],
                Some(OPERATION_TIMEOUT),
                None,
                user.ethermint_key,
            )
            .await;
    }

    let start_balance_coin = contact
        .get_balance(user.ethermint_address, registered_denom.clone())
        .await
        .unwrap()
        .unwrap();
    let start_balance_erc20 = contact
        .get_balance(user.ethermint_address, erc20_denom.clone())
        .await
        .unwrap()
        .unwrap();

    assert!(
        start_balance_coin.amount >= send_total + fee_total,
        "Not enough coin balance to run test"
    );
    assert!(
        start_balance_erc20.amount >= send_total + fee_total,
        "Not enough converted ERC20 balance to run test"
    );

    // Iterate through each amount, submit a MsgSendCoinToEvm and check the balances change correctly
    for (send_amount, fee_amount) in amounts.iter().zip(fee_amounts.iter()) {
        let start_balance_coin = contact
            .get_balance(user.ethermint_address, registered_denom.clone())
            .await
            .unwrap()
            .unwrap();
        let start_balance_erc20 = contact
            .get_balance(user.ethermint_address, erc20_denom.clone())
            .await
            .unwrap()
            .unwrap();

        let msg_coin = MsgSendCoinToEvm {
            coin: Some(ProtoCoin {
                amount: send_amount.to_string(),
                denom: registered_denom.clone(),
            }),
            sender: user.ethermint_address.to_string(),
        };
        let msg_erc20 = MsgSendCoinToEvm {
            coin: Some(ProtoCoin {
                amount: send_amount.to_string(),
                denom: erc20_denom.clone(),
            }),
            sender: user.ethermint_address.to_string(),
        };

        info!("Sending gasfree MsgSendCoinToEvm: {:?}", msg_coin);
        let cosmos_msg_coin = Msg::new(MSG_SEND_COIN_TO_EVM_TYPE_URL, msg_coin);
        let res_coin = contact
            .send_message(
                &[cosmos_msg_coin],
                None,
                &[], // no fee coin intentionally
                Some(OPERATION_TIMEOUT),
                None,
                user.ethermint_key,
            )
            .await;
        if !expect_failure {
            res_coin.as_ref().expect("MsgSendCoinToEvm failed");
        } else {
            res_coin
                .as_ref()
                .expect_err("expected MsgSendCoinToEvm to fail");
            return;
        }
        let res_coin = res_coin.unwrap();
        info!(
            "Executed gasfree MsgSendCoinToEvm tx gas_used={} hash={}",
            res_coin.gas_used(),
            res_coin.txhash()
        );

        let bank_balance_coin = contact
            .get_balance(user.ethermint_address, registered_denom.clone())
            .await
            .unwrap()
            .unwrap();
        let evm_balance_coin = web3
            .get_erc20_balance(denom_erc20_address, user.eth_address)
            .await
            .expect("Could not query ERC20 balance after send");

        let expected_bank_coin = start_balance_coin.amount - *send_amount - *fee_amount;
        assert_eq!(
            bank_balance_coin.amount, expected_bank_coin,
            "Cosmos balance not reduced by send amount + fee (start {} end {} expected {})",
            start_balance_coin.amount, bank_balance_coin.amount, expected_bank_coin
        );
        assert!(
            evm_balance_coin >= *send_amount,
            "EVM balance {} too low, expected at least {}",
            evm_balance_coin,
            send_amount
        );

        info!("Sending gasfree MsgSendCoinToEvm: {:?}", msg_erc20);
        let cosmos_msg_erc20 = Msg::new(MSG_SEND_COIN_TO_EVM_TYPE_URL, msg_erc20);
        let res_erc20 = contact
            .send_message(
                &[cosmos_msg_erc20],
                None,
                &[], // no fee coin intentionally
                Some(OPERATION_TIMEOUT),
                None,
                user.ethermint_key,
            )
            .await;
        if !expect_failure {
            res_erc20.as_ref().expect("MsgSendCoinToEvm failed");
        } else {
            res_erc20
                .as_ref()
                .expect_err("expected MsgSendCoinToEvm to fail");
            return;
        }
        let res_erc20 = res_erc20.unwrap();
        info!(
            "Executed gasfree MsgSendCoinToEvm tx gas_used={} hash={}",
            res_erc20.gas_used(),
            res_erc20.txhash()
        );

        let bank_balance_erc20 = contact
            .get_balance(user.ethermint_address, erc20_denom.clone())
            .await
            .unwrap()
            .unwrap();
        let evm_balance_erc20 = web3
            .get_erc20_balance(denom_erc20_address, user.eth_address)
            .await
            .expect("Could not query ERC20 balance after send");

        let expected_bank_erc20 = start_balance_erc20.amount - *send_amount - *fee_amount;
        assert_eq!(
            bank_balance_erc20.amount, expected_bank_erc20,
            "Cosmos balance not reduced by send amount + fee (start {} end {} expected {})",
            start_balance_coin.amount, bank_balance_erc20.amount, expected_bank_erc20
        );
        assert!(
            evm_balance_erc20 >= *send_amount,
            "EVM balance {} too low, expected at least {}",
            evm_balance_erc20,
            send_amount
        );
    }

    info!("Gasfree MsgSendCoinToEvm test successful");
}

async fn register_coin_if_not_registered(
    mut erc20_qc: Erc20QueryClient<Channel>,
    contact: &Contact,
    validator_keys: &[ValidatorKeys],
    denom: &str,
    footoken_meta: Metadata,
) {
    let pair_res = erc20_qc
        .token_pair(QueryTokenPairRequest {
            token: denom.to_string(),
        })
        .await;
    if pair_res.is_ok_and(|v| v.into_inner().token_pair.is_some()) {
        info!(
            "Token pair for {} already registered, skipping coin registration proposal",
            denom
        );
    } else {
        info!(
            "Token pair for {} not registered, submitting coin registration proposal",
            denom
        );
        execute_register_coin_proposal(
            contact,
            validator_keys,
            Some(TOTAL_TIMEOUT),
            RegisterCoinProposalParams {
                coin_metadata: footoken_meta.clone(),
                proposal_desc: "Register footoken for gasfree SendCoinToEvm test".to_string(),
                proposal_title: "Register footoken".to_string(),
            },
        )
        .await;
    }
}

async fn configure_gasfree_params_if_not_configured(
    contact: &Contact,
    denoms: &[String],
    fee_basis_points: u64,
    validator_keys: &[ValidatorKeys],
    expect_failure: bool,
) {
    // Build JSON values expected by params module (strings encoded like existing tests)
    // gas_free_erc20_interop_tokens is a JSON array of strings
    let interop_tokens = denoms.to_vec();
    let interop_tokens_value = serde_json::to_string(&interop_tokens).unwrap();
    let fee_bps_value = format!("\"{}\"", fee_basis_points);
    // Add gas free message type list including the new message type (append or overwrite)
    // We include microtx existing default plus MsgSendCoinToEvm
    let gas_free_message_types = vec![
        MSG_MICROTX_TYPE_URL,
        MSG_SEND_COIN_TO_EVM_TYPE_URL,
        MSG_SEND_ERC20_TO_COSMOS_TYPE_URL,
        MSG_SEND_ERC20_TO_COSMOS_AND_IBC_TRANSFER_TYPE_URL,
    ];
    let gas_free_message_types_value = serde_json::to_string(&gas_free_message_types).unwrap();

    let mut gasfree_qc = GasfreeQueryClient::connect(contact.get_url())
        .await
        .expect("could not connect to gasfree query client");

    let current_value = gasfree_qc
        .params(QueryParamsRequest {})
        .await
        .expect("could not query gasfree params")
        .into_inner()
        .params
        .expect("params response had no params");
    let mut continue_on = false;
    if current_value.gas_free_erc20_interop_tokens != interop_tokens {
        info!(
            "Updating gasfree ERC20 interop tokens from {:?} to {}",
            current_value.gas_free_erc20_interop_tokens, interop_tokens_value
        );
        continue_on = true;
    }
    if current_value.gas_free_erc20_interop_fee_basis_points != fee_basis_points {
        info!(
            "Updating gasfree ERC20 interop fee basis points from {} to {}",
            current_value.gas_free_erc20_interop_fee_basis_points, fee_bps_value
        );
        continue_on = true;
    }
    if current_value.gas_free_message_types != gas_free_message_types {
        info!(
            "Updating gasfree message types from {:?} to {:?}",
            current_value.gas_free_message_types, gas_free_message_types
        );
        continue_on = true;
    }

    if !continue_on {
        info!("Gasfree params already configured for test, skipping parameter change proposal");
        return;
    }
    let proposal = ParameterChangeProposal {
        title: "Configure gasfree ERC20 interop params".to_string(),
        description: "Set interop tokens, fee basis points, and gas free message types for test"
            .to_string(),
        changes: vec![
            ParamChange {
                // subspace gasfree, key name guessed from struct JSON tag
                subspace: "gasfree".to_string(),
                key: "gasFreeErc20InteropTokens".to_string(),
                value: interop_tokens_value,
            },
            ParamChange {
                subspace: "gasfree".to_string(),
                key: "gasFreeErc20InteropFeeBasisPoints".to_string(),
                value: fee_bps_value,
            },
            ParamChange {
                // update gas free msg type list to include MsgSendCoinToEvm
                subspace: "gasfree".to_string(),
                key: "gasFreeMessageTypes".to_string(),
                value: gas_free_message_types_value,
            },
        ],
    };
    info!(
        "Submitting gasfree parameter change proposal: {:?}",
        proposal
    );
    let deposit = Coin {
        amount: one_atom() * 100u8.into(),
        denom: STAKING_TOKEN.clone(),
    };
    let fee = get_fee(None);
    let res = contact
        .submit_parameter_change_proposal(
            proposal,
            deposit,
            fee,
            validator_keys.first().unwrap().validator_key,
            Some(OPERATION_TIMEOUT),
        )
        .await;

    if !expect_failure {
        res.as_ref()
            .expect("parameter change proposal submission failed");
    } else {
        res.as_ref()
            .expect_err("expected parameter change proposal submission to fail");
        return;
    }
    let res = res.unwrap();
    info!(
        "Submitted gasfree param change proposal: {:?}",
        res.raw_log()
    );
    vote_yes_on_proposals(contact, validator_keys, None).await;
    wait_for_proposals_to_execute(contact).await;
}

/// Generate `num` random Uint256 values that sum to `total`.
fn random_splits_uint256(total: u64, num: usize) -> Vec<Uint256> {
    use rand::{thread_rng, Rng};
    let mut rng = thread_rng();
    let mut splits = vec![];
    let mut remaining = total;
    for i in 0..(num - 1) {
        let max = remaining - (num - i - 1) as u64;
        let val = rng.gen_range(1..=max);
        splits.push(val);
        remaining -= val;
    }
    splits.push(remaining);
    splits.into_iter().map(Uint256::from).collect()
}

// Tests that a cosmos native coin and an erc20 can be sent from evm to cosmos without paying a tx fee, instead
// the gasfree fee is added to the conversion amount and collected on the cosmos side
pub async fn gasfree_send_erc20_to_cosmos_test(
    contact: &Contact,
    web3: &Web3,
    validator_keys: &[ValidatorKeys],
    evm_user_keys: &[EthermintUserKey],
    erc20_contracts: &[EthAddress],
    expect_failure: bool,
) {
    info!("Starting gasfree MsgSendErc20ToCosmos test");

    let fee_basis_points: u64 = 100;
    let amounts = random_splits_uint256(300_000, 3)
        .iter()
        .map(|v| *v * one_atom())
        .collect::<Vec<Uint256>>();
    let fee_amounts = amounts
        .iter()
        .map(|v| *v * fee_basis_points.into() / 10_000u32.into())
        .collect::<Vec<Uint256>>();
    let send_total = amounts.iter().fold(Uint256::default(), |a, v| a + *v);
    let fee_total = fee_amounts.iter().fold(Uint256::default(), |a, v| a + *v);

    let user = evm_user_keys.first().expect("No EVM users provided");
    let mut erc20_qc = Erc20QueryClient::connect(contact.get_url())
        .await
        .expect("could not connect to erc20 query client");

    let footoken_meta = footoken_metadata(contact).await;
    let registered_denom = footoken_meta.base.clone();
    let registered_erc20 = erc20_contracts.first().unwrap();
    if !SKIP_GOV {
        register_coin_if_not_registered(
            erc20_qc.clone(),
            contact,
            validator_keys,
            &registered_denom,
            footoken_meta,
        )
        .await;
        submit_and_pass_token_proposal(
            contact,
            erc20_qc.clone(),
            validator_keys,
            registered_erc20.to_string(),
            true,
        )
        .await;
    } else {
        info!("Skipping coin registration governance due to SKIP_GOV");
    }
    let denom_pair = erc20_qc
        .token_pair(QueryTokenPairRequest {
            token: registered_denom.clone(),
        })
        .await
        .expect("could not query token pair")
        .into_inner()
        .token_pair
        .expect("token pair response had no token pair");
    let denom_erc20_address = denom_pair
        .erc20_address
        .parse::<EthAddress>()
        .expect("failed to parse ERC20 address");
    let erc20_pair = erc20_qc
        .token_pair(QueryTokenPairRequest {
            token: registered_erc20.to_string(),
        })
        .await
        .expect("could not query token pair")
        .into_inner()
        .token_pair
        .expect("token pair response had no token pair");
    let erc20_denom = erc20_pair.denom;

    if !SKIP_GOV {
        configure_gasfree_params_if_not_configured(
            contact,
            &[registered_denom.clone(), registered_erc20.to_string()],
            fee_basis_points,
            validator_keys,
            expect_failure,
        )
        .await;
    } else {
        info!("Skipping gasfree parameter change governance due to SKIP_GOV");
    }

    // Fund the user with enough of the cosmos coin to pay fees and send back to cosmos
    let begin_bank_balance = contact
        .get_balance(user.ethermint_address, registered_denom.clone())
        .await
        .unwrap_or_default()
        .unwrap_or_default();

    if begin_bank_balance.amount < send_total + fee_total {
        contact
            .send_coins(
                Coin {
                    amount: send_total + fee_total,
                    denom: registered_denom.clone(),
                },
                get_fee_option(None),
                user.ethermint_address,
                Some(OPERATION_TIMEOUT),
                validator_keys.last().unwrap().validator_key,
            )
            .await
            .expect("Unable to fund user with test coin");
    }
    let begin_evm_balance = web3
        .get_erc20_balance(denom_erc20_address, user.eth_address)
        .await
        .unwrap_or_default();
    if begin_evm_balance < send_total + fee_total {
        let msg = MsgConvertCoin {
            coin: Some(ProtoCoin {
                amount: (send_total + fee_total).to_string(),
                denom: registered_denom.clone(),
            }),
            sender: user.ethermint_address.to_string(),
            receiver: user.eth_address.to_string(),
        };
        info!("Sending MsgConvertCoin: {:?}", msg);
        let cosmos_msg = Msg::new(MSG_CONVERT_COIN_TYPE_URL, msg);
        contact
            .send_message(
                &[cosmos_msg],
                None,
                &[get_fee(None)], // no fee coin intentionally
                Some(OPERATION_TIMEOUT),
                None,
                user.ethermint_key,
            )
            .await
            .expect("MsgSendCoinToEvm failed");
    }

    // Fund the user with the erc20 token's cosmos representation to be sent
    let erc20_to_cosmos_msg = MsgConvertErc20 {
        amount: (send_total + fee_total).to_string(),
        contract_address: registered_erc20.to_string(),
        receiver: user.ethermint_address.to_string(),
        sender: user.eth_address.to_string(),
    };
    info!("Planning to send this MsgConvertErc20: {erc20_to_cosmos_msg:?}");
    let msg = Msg::new(MSG_CONVERT_ERC20_TYPE_URL, erc20_to_cosmos_msg);

    let _ = contact
        .send_message(
            &[msg],
            None,
            &[get_fee(None)],
            Some(OPERATION_TIMEOUT),
            None,
            user.ethermint_key,
        )
        .await;

    let start_evm_balance_coin = web3
        .get_erc20_balance(denom_erc20_address, user.eth_address)
        .await
        .expect("Could not query ERC20 balance before send");
    let start_evm_balance_erc20 = web3
        .get_erc20_balance(*registered_erc20, user.eth_address)
        .await
        .expect("Could not query ERC20 balance before send");

    assert!(
        start_evm_balance_coin >= send_total + fee_total,
        "Not enough coin balance to run test {:?} < {:?}",
        start_evm_balance_coin,
        send_total + fee_total
    );
    assert!(
        start_evm_balance_erc20 >= send_total + fee_total,
        "Not enough converted ERC20 balance to run test {:?} < {:?}",
        start_evm_balance_erc20,
        send_total + fee_total
    );

    // Iterate through each amount, submit a MsgSendErc20ToCosmos and check the balances change correctly
    for (send_amount, fee_amount) in amounts.iter().zip(fee_amounts.iter()) {
        let start_bank_coin = contact
            .get_balance(user.ethermint_address, registered_denom.clone())
            .await
            .unwrap()
            .unwrap_or_else(|| deep_space::Coin {
                amount: Uint256::zero(),
                denom: registered_denom.clone(),
            });
        let start_evm_coin = web3
            .get_erc20_balance(denom_erc20_address, user.eth_address)
            .await
            .expect("Could not query ERC20 balance before send");
        let start_bank_erc20 = contact
            .get_balance(user.ethermint_address, erc20_denom.clone())
            .await
            .unwrap()
            .unwrap_or_else(|| deep_space::Coin {
                amount: Uint256::zero(),
                denom: erc20_denom.clone(),
            });
        let start_evm_erc20 = web3
            .get_erc20_balance(*registered_erc20, user.eth_address)
            .await
            .expect("Could not query ERC20 balance before send");

        // Construct and send MsgSendErc20ToCosmos
        let msg_coin = MsgSendErc20ToCosmos {
            erc20: denom_erc20_address.to_string(),
            amount: send_amount.to_string(),
            sender: user.eth_address.to_string(),
        };
        let msg_erc20 = MsgSendErc20ToCosmos {
            erc20: registered_erc20.to_string(),
            amount: send_amount.to_string(),
            sender: user.eth_address.to_string(),
        };

        info!("Sending gasfree MsgSendErc20ToCosmos: {:?}", msg_coin);
        let cosmos_msg_coin = Msg::new(MSG_SEND_ERC20_TO_COSMOS_TYPE_URL, msg_coin);
        let res_coin = contact
            .send_message(
                &[cosmos_msg_coin],
                None,
                &[], // no fee coin intentionally
                Some(OPERATION_TIMEOUT),
                None,
                user.ethermint_key,
            )
            .await;
        if !expect_failure {
            res_coin.as_ref().expect("MsgSendErc20ToCosmos failed");
        } else {
            res_coin
                .as_ref()
                .expect_err("expected MsgSendErc20ToCosmos to fail");
            return;
        }
        let res_coin = res_coin.unwrap();
        info!(
            "Executed gasfree MsgSendErc20ToCosmos tx gas_used={} hash={}",
            res_coin.gas_used(),
            res_coin.txhash()
        );

        if !expect_failure {
            let bank_balance_coin = contact
                .get_balance(user.ethermint_address, registered_denom.clone())
                .await
                .unwrap()
                .unwrap_or_else(|| deep_space::Coin {
                    amount: Uint256::zero(),
                    denom: registered_denom.clone(),
                });
            let evm_balance_coin = web3
                .get_erc20_balance(denom_erc20_address, user.eth_address)
                .await
                .expect("Could not query ERC20 balance after send");

            let expected_evm_coin = start_evm_coin - *send_amount - *fee_amount;
            let expected_bank_coin = start_bank_coin.amount + *send_amount;
            assert_eq!(
                evm_balance_coin, expected_evm_coin,
                "EVM balance not reduced by send amount and fee (start {} end {} expected {})",
                start_evm_coin, evm_balance_coin, expected_evm_coin
            );
            assert_eq!(
                bank_balance_coin.amount, expected_bank_coin,
                "Cosmos balance not increased by send amount (start {} end {} expected {})",
                start_bank_coin.amount, bank_balance_coin.amount, expected_bank_coin
            );
        }

        // -------------------------------

        info!("Sending gasfree MsgSendErc20ToCosmos: {:?}", msg_erc20);
        let cosmos_msg_erc20 = Msg::new(MSG_SEND_ERC20_TO_COSMOS_TYPE_URL, msg_erc20);
        let res_erc20 = contact
            .send_message(
                &[cosmos_msg_erc20],
                None,
                &[], // no fee coin intentionally
                Some(OPERATION_TIMEOUT),
                None,
                user.ethermint_key,
            )
            .await;
        if !expect_failure {
            res_erc20.as_ref().expect("MsgSendErc20ToCosmos failed");
        } else {
            res_erc20
                .as_ref()
                .expect_err("expected MsgSendErc20ToCosmos to fail");
            return;
        }
        let res_erc20 = res_erc20.unwrap();

        info!(
            "Executed gasfree MsgSendErc20ToCosmos tx gas_used={} hash={}",
            res_erc20.gas_used(),
            res_erc20.txhash()
        );

        if !expect_failure {
            let bank_balance_erc20 = contact
                .get_balance(user.ethermint_address, erc20_denom.to_string())
                .await
                .unwrap()
                .unwrap_or_else(|| deep_space::Coin {
                    amount: Uint256::zero(),
                    denom: erc20_denom.to_string(),
                });
            let evm_balance_erc20 = web3
                .get_erc20_balance(*registered_erc20, user.eth_address)
                .await
                .expect("Could not query ERC20 balance after send");

            let expected_evm_erc20 = start_evm_erc20 - *send_amount - *fee_amount;
            let expected_bank_erc20 = start_bank_erc20.amount + *send_amount;
            assert_eq!(
                evm_balance_erc20, expected_evm_erc20,
                "EVM balance not reduced by send amount - fee (start {} end {} expected {})",
                start_evm_erc20, evm_balance_erc20, expected_evm_erc20
            );
            assert_eq!(
                bank_balance_erc20.amount, expected_bank_erc20,
                "Cosmos balance not increased by send amount (start {} end {} expected {})",
                start_bank_erc20.amount, bank_balance_erc20.amount, expected_bank_erc20
            );
        }
    }

    info!("Gasfree MsgSendErc20ToCosmos test successful");
}

// Tests that a cosmos native coin and an erc20 can be sent from evm to an IBC connected chain without paying a tx fee, instead
// the gasfree fee is added to the conversion amount and collected on the cosmos side, before the sent coin is forwarded over IBC
pub async fn gasfree_send_erc20_to_cosmos_and_ibc_transfer_test(
    althea_contact: &Contact,
    ibc_contact: &Contact,
    web3: &Web3,
    validator_keys: &[ValidatorKeys],
    evm_user_keys: &[EthermintUserKey],
    erc20_contracts: &[EthAddress],
    expect_failure: bool,
) {
    info!("Starting gasfree MsgSendERC20ToCosmosAndIBCTransfer test");

    let fee_basis_points: u64 = 100;
    let amounts = random_splits_uint256(300_000, 3)
        .iter()
        .map(|v| *v * one_atom())
        .collect::<Vec<Uint256>>();
    let fee_amounts = amounts
        .iter()
        .map(|v| *v * fee_basis_points.into() / 10_000u32.into())
        .collect::<Vec<Uint256>>();
    let amount_total = amounts.iter().fold(Uint256::default(), |a, v| a + *v);
    let fee_total = fee_amounts.iter().fold(Uint256::default(), |a, v| a + *v);

    let althea_channel_qc = IbcChannelQueryClient::connect(COSMOS_NODE_GRPC.as_str())
        .await
        .expect("Could not connect channel query client");

    // Wait for the ibc channel to be created and find the channel ids
    info!("Waiting for IBC channel creation:");
    let channel_id_timeout = Duration::from_secs(60 * 5);
    let althea_channel = get_channel(
        althea_channel_qc,
        get_ibc_chain_id(),
        Some(channel_id_timeout),
    )
    .await
    .expect("Could not find ibc-test-1 <-> althea_6633438-1 channel");
    let ibc_ibc_transfer_qc = IbcTransferQueryClient::connect(IBC_NODE_GRPC.as_str())
        .await
        .expect("Could not connect to ibc transfer query client");

    let mut althea_erc20_qc = Erc20QueryClient::connect(COSMOS_NODE_GRPC.as_str())
        .await
        .expect("Could not connect erc20 query client");

    let registered_erc20 = erc20_contracts.first().unwrap();
    let footoken_meta = footoken_metadata(althea_contact).await;
    let registered_denom = footoken_meta.base.clone();

    if !SKIP_GOV {
        register_coin_if_not_registered(
            althea_erc20_qc.clone(),
            althea_contact,
            validator_keys,
            &registered_denom,
            footoken_meta,
        )
        .await;
        submit_and_pass_token_proposal(
            althea_contact,
            althea_erc20_qc.clone(),
            validator_keys,
            registered_erc20.to_string(),
            true,
        )
        .await;
    }
    let denom_pair = althea_erc20_qc
        .token_pair(QueryTokenPairRequest {
            token: registered_denom.clone(),
        })
        .await
        .expect("could not query token pair")
        .into_inner()
        .token_pair
        .expect("token pair response had no token pair");
    let denom_erc20_address = denom_pair
        .erc20_address
        .parse::<EthAddress>()
        .expect("failed to parse ERC20 address");
    let erc20_pair = althea_erc20_qc
        .token_pair(QueryTokenPairRequest {
            token: registered_erc20.to_string(),
        })
        .await
        .expect("could not query token pair")
        .into_inner()
        .token_pair
        .expect("token pair response had no token pair");
    let erc20_denom = erc20_pair.denom;

    if !SKIP_GOV {
        configure_gasfree_params_if_not_configured(
            althea_contact,
            &[registered_denom.clone(), registered_erc20.to_string()],
            fee_basis_points,
            validator_keys,
            expect_failure,
        )
        .await;
    } else {
        info!("Skipping gasfree parameter change governance due to SKIP_GOV");
    }

    let user = evm_user_keys.first().expect("No EVM users provided");
    let ibc_address = user.ethermint_key.to_address(&IBC_ADDRESS_PREFIX).unwrap();

    let begin_bank_balance = althea_contact
        .get_balance(user.ethermint_address, registered_denom.clone())
        .await
        .unwrap_or_default()
        .unwrap_or_default();
    if begin_bank_balance.amount < amount_total + fee_total {
        // Fund the user with the cosmos coin to pay fees and send over IBC
        althea_contact
            .send_coins(
                Coin {
                    amount: amount_total + fee_total,
                    denom: registered_denom.clone(),
                },
                get_fee_option(None),
                user.ethermint_address,
                Some(OPERATION_TIMEOUT),
                validator_keys.last().unwrap().validator_key,
            )
            .await
            .expect("Unable to fund user with test coin");
    }
    let begin_evm_balance = web3
        .get_erc20_balance(denom_erc20_address, user.eth_address)
        .await
        .unwrap_or_default();
    if begin_evm_balance < amount_total + fee_total {
        let msg = MsgConvertCoin {
            coin: Some(ProtoCoin {
                amount: (amount_total + fee_total).to_string(),
                denom: registered_denom.clone(),
            }),
            sender: user.ethermint_address.to_string(),
            receiver: user.eth_address.to_string(),
        };

        info!("Sending MsgConvertCoin: {:?}", msg);
        let cosmos_msg = Msg::new(MSG_CONVERT_COIN_TYPE_URL, msg);
        althea_contact
            .send_message(
                &[cosmos_msg],
                None,
                &[get_fee(None)],
                Some(OPERATION_TIMEOUT),
                None,
                user.ethermint_key,
            )
            .await
            .expect("MsgSendCoinToEvm failed");
    }
    let start_evm_balance_erc20 = web3
        .get_erc20_balance(*registered_erc20, user.eth_address)
        .await
        .expect("Could not query ERC20 balance before send");
    let start_althea_balance_erc20 = althea_contact
        .get_balance(user.ethermint_address, erc20_denom.clone())
        .await
        .unwrap()
        .unwrap();
    let start_evm_balance_denom = web3
        .get_erc20_balance(denom_erc20_address, user.eth_address)
        .await
        .expect("Could not query ERC20 balance before send");
    let start_althea_balance_denom = althea_contact
        .get_balance(user.ethermint_address, registered_denom.clone())
        .await
        .unwrap()
        .unwrap();

    for send_amount in amounts.iter() {
        let msg_denom = MsgSendErc20ToCosmosAndIbcTransfer {
            erc20: denom_erc20_address.to_string(),
            amount: send_amount.to_string(),
            sender: user.eth_address.to_string(),
            destination_port: althea_channel.port_id.clone(),
            destination_channel: althea_channel.channel_id.clone(),
            destination_receiver: ibc_address.to_string(),
        };
        let msg_erc20 = MsgSendErc20ToCosmosAndIbcTransfer {
            erc20: registered_erc20.to_string(),
            amount: send_amount.to_string(),
            sender: user.eth_address.to_string(),
            destination_port: althea_channel.port_id.clone(),
            destination_channel: althea_channel.channel_id.clone(),
            destination_receiver: ibc_address.to_string(),
        };
        info!(
            "Sending gasfree MsgSendErc20ToCosmosAndIBCTransfer: {:?}",
            msg_erc20
        );
        let cosmos_msg_erc20 = Msg::new(
            MSG_SEND_ERC20_TO_COSMOS_AND_IBC_TRANSFER_TYPE_URL,
            msg_erc20,
        );
        let res_erc20 = althea_contact
            .send_message(
                &[cosmos_msg_erc20],
                None,
                &[], // no fee coin intentionally
                Some(OPERATION_TIMEOUT),
                None,
                user.ethermint_key,
            )
            .await;
        if !expect_failure {
            res_erc20
                .as_ref()
                .expect("MsgSendErc20ToCosmosAndIBCTransfer failed");
        } else {
            res_erc20
                .as_ref()
                .expect_err("expected MsgSendErc20ToCosmosAndIBCTransfer to fail");
            return;
        }
        let res_erc20 = res_erc20.unwrap();
        info!(
            "Executed gasfree MsgSendErc20ToCosmosAndIBCTransfer tx gas_used={} hash={}",
            res_erc20.gas_used(),
            res_erc20.txhash()
        );

        // -------------------------------
        info!(
            "Sending gasfree MsgSendErc20ToCosmosAndIBCTransfer: {:?}",
            msg_denom
        );
        let cosmos_msg_denom = Msg::new(
            MSG_SEND_ERC20_TO_COSMOS_AND_IBC_TRANSFER_TYPE_URL,
            msg_denom,
        );
        let res_denom = althea_contact
            .send_message(
                &[cosmos_msg_denom],
                None,
                &[], // no fee coin intentionally
                Some(OPERATION_TIMEOUT),
                None,
                user.ethermint_key,
            )
            .await;
        if !expect_failure {
            res_denom
                .as_ref()
                .expect("MsgSendErc20ToCosmosAndIBCTransfer failed");
        } else {
            res_denom
                .as_ref()
                .expect_err("expected MsgSendErc20ToCosmosAndIBCTransfer to fail");
            return;
        }
        let res_denom = res_denom.unwrap();
        info!(
            "Executed gasfree MsgSendErc20ToCosmosAndIBCTransfer tx gas_used={} hash={}",
            res_denom.gas_used(),
            res_denom.txhash()
        );
    }

    if !expect_failure {
        sleep(Duration::from_secs(60));

        let ibc_erc20 = get_hash_for_denom_trace(
            erc20_denom.clone(),
            "channel-0".to_string(),
            None,
            ibc_ibc_transfer_qc.clone(),
        )
        .await
        .expect("Could not get erc20 coin denom on ibc chain");
        let ibc_denom = get_hash_for_denom_trace(
            registered_denom.clone(),
            "channel-0".to_string(),
            None,
            ibc_ibc_transfer_qc.clone(),
        )
        .await
        .expect("Could not get althea bank coin denom on ibc chain");

        let ibc_balance_erc20 = ibc_contact
            .get_balance(ibc_address, ibc_erc20.clone())
            .await
            .expect("Could not get ibc erc20 balance")
            .unwrap_or(Coin {
                amount: 0u8.into(),
                denom: ibc_erc20.clone(),
            });
        let end_evm_balance_erc20 = web3
            .get_erc20_balance(*registered_erc20, user.eth_address)
            .await
            .expect("Could not query ERC20 balance before send");
        let end_althea_balance_erc20 = althea_contact
            .get_balance(user.ethermint_address, erc20_denom.clone())
            .await
            .unwrap()
            .unwrap();

        // Aggregate checks
        let expected_evm_erc20 = start_evm_balance_erc20 - amount_total - fee_total;
        let expected_althea_erc20 = start_althea_balance_erc20.amount;
        assert_eq!(
            end_evm_balance_erc20, expected_evm_erc20,
            "EVM balance not reduced by total sent - total fees (start {} end {} expected {})",
            start_evm_balance_erc20, end_evm_balance_erc20, expected_evm_erc20
        );
        assert_eq!(
            end_althea_balance_erc20.amount,
            expected_althea_erc20,
            "Cosmos balance should have been unchanged (start {} end {} expected {})",
            start_althea_balance_erc20.amount,
            end_althea_balance_erc20.amount,
            expected_althea_erc20
        );
        assert_eq!(
            ibc_balance_erc20.amount, amount_total,
            "IBC balance not increased by total sent (expected {})",
            amount_total
        );

        // -------------------------------
        let ibc_balance_denom = ibc_contact
            .get_balance(ibc_address, ibc_denom.clone())
            .await
            .expect("Could not get ibc denom balance")
            .unwrap_or(Coin {
                amount: 0u8.into(),
                denom: ibc_denom.clone(),
            });
        let end_evm_balance_denom = web3
            .get_erc20_balance(denom_erc20_address, user.eth_address)
            .await
            .expect("Could not query ERC20 balance before send");
        let end_althea_balance_denom = althea_contact
            .get_balance(user.ethermint_address, registered_denom.clone())
            .await
            .unwrap()
            .unwrap();

        // Aggregate checks
        let expected_evm_denom = start_evm_balance_denom - amount_total - fee_total;
        let expected_althea_denom = start_althea_balance_denom.amount;
        assert_eq!(
            end_evm_balance_denom, expected_evm_denom,
            "EVM balance not reduced by total sent and total fees (start {} end {} expected {})",
            start_evm_balance_denom, end_evm_balance_denom, expected_evm_denom
        );
        assert_eq!(
            end_althea_balance_denom.amount,
            expected_althea_denom,
            "Cosmos balance should have been unchanged (start {} end {} expected {})",
            start_althea_balance_denom.amount,
            end_althea_balance_denom.amount,
            expected_althea_denom
        );
        assert_eq!(
            ibc_balance_denom.amount, amount_total,
            "IBC balance not increased by total sent (expected {})",
            amount_total
        );
    }

    info!("Gasfree MsgSendErc20ToCosmos test successful");
}

/// Test MsgSendCoinToEvm fails when the user has insufficient balance to pay the fee
pub async fn gasfree_send_coin_to_evm_insufficient_balance_test(
    contact: &Contact,
    web30: &Web3,
    validator_keys: &[ValidatorKeys],
    evm_user_keys: &[EthermintUserKey],
    erc20_contracts: &[EthAddress],
    expect_failure: bool,
) {
    info!("Starting gasfree MsgSendCoinToEvm insufficient balance test");

    let fee_basis_points: u64 = 100;

    let erc20_qc = Erc20QueryClient::connect(contact.get_url())
        .await
        .expect("could not connect to erc20 query client");
    let footoken_meta = footoken_metadata(contact).await;
    let registered_denom = footoken_meta.base.clone();
    let registered_erc20 = erc20_contracts.first().unwrap();

    let funder = evm_user_keys.first().expect("No EVM users provided");
    let user = get_user_key(None);
    web30
        .erc20_send(
            one_atom(),
            user.eth_address,
            *registered_erc20,
            funder.eth_privkey,
            Some(OPERATION_TIMEOUT),
            vec![],
        )
        .await
        .expect("Failed to send ERC20");
    contact
        .send_coins(
            Coin {
                amount: one_atom(),
                denom: STAKING_TOKEN.clone(),
            },
            get_fee_option(None),
            user.ethermint_address,
            Some(OPERATION_TIMEOUT),
            validator_keys.last().unwrap().validator_key,
        )
        .await
        .expect("Failed to send gas token");

    if !SKIP_GOV {
        register_coin_if_not_registered(
            erc20_qc.clone(),
            contact,
            validator_keys,
            &registered_denom,
            footoken_meta,
        )
        .await;
        submit_and_pass_token_proposal(
            contact,
            erc20_qc.clone(),
            validator_keys,
            registered_erc20.to_string(),
            true,
        )
        .await;
    } else {
        info!("Skipping coin registration governance due to SKIP_GOV");
    }

    if !SKIP_GOV {
        // Match the other gasfree params so this test does not trigger a gov proposal
        configure_gasfree_params_if_not_configured(
            contact,
            &[registered_denom.clone(), registered_erc20.to_string()],
            fee_basis_points,
            validator_keys,
            expect_failure,
        )
        .await;
    } else {
        info!("Skipping gasfree parameter change governance due to SKIP_GOV");
    }

    let start_balance = contact
        .get_balance(user.ethermint_address, registered_denom.clone())
        .await
        .unwrap()
        .unwrap();

    // Attempt MsgSendCoinToEvm, should fail
    let msg = MsgSendCoinToEvm {
        coin: Some(ProtoCoin {
            amount: start_balance.amount.to_string(),
            denom: registered_denom.clone(),
        }),
        sender: user.ethermint_address.to_string(),
    };
    info!(
        "Sending gasfree MsgSendCoinToEvm with insufficient balance: {:?}",
        msg
    );
    let cosmos_msg = Msg::new(MSG_SEND_COIN_TO_EVM_TYPE_URL, msg);
    let res = contact
        .send_message(
            &[cosmos_msg],
            None,
            &[], // no fee coin intentionally
            Some(OPERATION_TIMEOUT),
            None,
            user.ethermint_key,
        )
        .await;

    if !expect_failure {
        assert!(
            res.is_err(),
            "MsgSendCoinToEvm should fail due to insufficient balance"
        );

        let end_balance = contact
            .get_balance(user.ethermint_address, registered_denom.clone())
            .await
            .unwrap()
            .unwrap();

        assert!(
            end_balance.amount.eq(&start_balance.amount),
            "User balance should be unchanged after failed MsgSendCoinToEvm"
        );
    } else {
        res.expect_err("expected MsgSendCoinToEvm to fail");
    }

    info!("Gasfree MsgSendCoinToEvm insufficient balance test successful");
}

/// Test MsgSendErc20ToCosmos fails when the user has insufficient balance to pay the fee
pub async fn gasfree_send_erc20_to_cosmos_insufficient_balance_test(
    contact: &Contact,
    web30: &Web3,
    validator_keys: &[ValidatorKeys],
    evm_user_keys: &[EthermintUserKey],
    erc20_contracts: &[EthAddress],
    expect_failure: bool,
) {
    info!("Starting gasfree MsgSendErc20ToCosmos insufficient balance test");

    let fee_basis_points: u64 = 100;

    let mut erc20_qc = Erc20QueryClient::connect(contact.get_url())
        .await
        .expect("could not connect to erc20 query client");
    let footoken_meta = footoken_metadata(contact).await;
    let registered_denom = footoken_meta.base.clone();
    let registered_erc20 = erc20_contracts.first().unwrap();

    let funder = evm_user_keys.first().expect("No EVM users provided");
    let user = get_user_key(None);
    web30
        .erc20_send(
            one_atom(),
            user.eth_address,
            *registered_erc20,
            funder.eth_privkey,
            Some(OPERATION_TIMEOUT),
            vec![],
        )
        .await
        .expect("Failed to send ERC20");
    contact
        .send_coins(
            Coin {
                amount: one_atom(),
                denom: STAKING_TOKEN.clone(),
            },
            get_fee_option(None),
            user.ethermint_address,
            Some(OPERATION_TIMEOUT),
            validator_keys.last().unwrap().validator_key,
        )
        .await
        .expect("Failed to send gas token");

    if !SKIP_GOV {
        register_coin_if_not_registered(
            erc20_qc.clone(),
            contact,
            validator_keys,
            &registered_denom,
            footoken_meta,
        )
        .await;
        submit_and_pass_token_proposal(
            contact,
            erc20_qc.clone(),
            validator_keys,
            registered_erc20.to_string(),
            true,
        )
        .await;
    } else {
        info!("Skipping coin registration governance due to SKIP_GOV");
    }
    let erc20_pair = erc20_qc
        .token_pair(QueryTokenPairRequest {
            token: registered_erc20.to_string(),
        })
        .await
        .expect("could not query token pair")
        .into_inner()
        .token_pair
        .expect("token pair response had no token pair");
    let erc20_denom = erc20_pair.denom;

    if !SKIP_GOV {
        // Match the other gasfree params so this test does not trigger a gov proposal
        configure_gasfree_params_if_not_configured(
            contact,
            &[registered_denom.clone(), registered_erc20.to_string()],
            fee_basis_points,
            validator_keys,
            expect_failure,
        )
        .await;
    } else {
        info!("Skipping gasfree parameter change governance due to SKIP_GOV");
    }

    let check_balance = contact
        .get_balance(user.ethermint_address, erc20_denom.clone())
        .await
        .unwrap()
        .unwrap();

    if check_balance.amount < one_atom() {
        contact
            .send_coins(
                Coin {
                    amount: one_atom(),
                    denom: registered_denom.clone(),
                },
                get_fee_option(None),
                user.ethermint_address,
                Some(OPERATION_TIMEOUT),
                validator_keys.last().unwrap().validator_key,
            )
            .await
            .expect("Unable to fund user with test coin");
        let msg = MsgConvertCoin {
            coin: Some(ProtoCoin {
                amount: one_atom().to_string(),
                denom: registered_denom.clone(),
            }),
            sender: user.ethermint_address.to_string(),
            receiver: user.eth_address.to_string(),
        };
        let cosmos_msg = Msg::new(MSG_CONVERT_COIN_TYPE_URL, msg);
        contact
            .send_message(
                &[cosmos_msg],
                None,
                &[get_fee(None)], // no fee coin intentionally
                Some(OPERATION_TIMEOUT),
                None,
                user.ethermint_key,
            )
            .await
            .expect("MsgSendCoinToEvm failed");
    }
    let start_balance = contact
        .get_balance(user.ethermint_address, erc20_denom.clone())
        .await
        .unwrap()
        .unwrap();

    // Attempt MsgSendErc20ToCosmos, should fail
    let msg = MsgSendErc20ToCosmos {
        erc20: registered_erc20.to_string(),
        amount: start_balance.to_string(),
        sender: user.eth_address.to_string(),
    };
    info!(
        "Sending gasfree MsgSendErc20ToCosmos with insufficient balance: {:?}",
        msg
    );
    let cosmos_msg = Msg::new(MSG_SEND_ERC20_TO_COSMOS_TYPE_URL, msg);
    let res = contact
        .send_message(
            &[cosmos_msg],
            None,
            &[], // no fee coin intentionally
            Some(OPERATION_TIMEOUT),
            None,
            user.ethermint_key,
        )
        .await;
    if !expect_failure {
        assert!(
            res.is_err(),
            "MsgSendErc20ToCosmos should fail due to insufficient balance"
        );

        let end_balance = contact
            .get_balance(user.ethermint_address, erc20_denom.clone())
            .await
            .unwrap()
            .unwrap();

        assert!(
            end_balance.amount.eq(&start_balance.amount),
            "User ERC20 balance should be unchanged after failed MsgSendErc20ToCosmos"
        );
    } else {
        res.expect_err("expected MsgSendErc20ToCosmos to fail");
    }

    info!("Gasfree MsgSendErc20ToCosmos insufficient balance test successful");
}

/// Test MsgSendErc20ToCosmosAndIBCTransfer fails when the user has insufficient balance to pay the fee
pub async fn gasfree_send_erc20_to_cosmos_and_ibc_transfer_insufficient_balance_test(
    contact: &Contact,
    web30: &Web3,
    validator_keys: &[ValidatorKeys],
    evm_user_keys: &[EthermintUserKey],
    erc20_contracts: &[EthAddress],
    expect_failure: bool,
) {
    info!("Starting gasfree MsgSendErc20ToCosmos insufficient balance test");

    let fee_basis_points: u64 = 100;

    let mut erc20_qc = Erc20QueryClient::connect(contact.get_url())
        .await
        .expect("could not connect to erc20 query client");
    let footoken_meta = footoken_metadata(contact).await;
    let registered_denom = footoken_meta.base.clone();
    let registered_erc20 = erc20_contracts.first().unwrap();

    let funder = evm_user_keys.first().expect("No EVM users provided");
    let user = get_user_key(None);
    let ibc_address = user.ethermint_key.to_address(&IBC_ADDRESS_PREFIX).unwrap();
    web30
        .erc20_send(
            one_atom(),
            user.eth_address,
            *registered_erc20,
            funder.eth_privkey,
            Some(OPERATION_TIMEOUT),
            vec![],
        )
        .await
        .expect("Failed to send ERC20");
    contact
        .send_coins(
            Coin {
                amount: one_atom(),
                denom: STAKING_TOKEN.clone(),
            },
            get_fee_option(None),
            user.ethermint_address,
            Some(OPERATION_TIMEOUT),
            validator_keys.last().unwrap().validator_key,
        )
        .await
        .expect("Failed to send gas token");

    if !SKIP_GOV {
        register_coin_if_not_registered(
            erc20_qc.clone(),
            contact,
            validator_keys,
            &registered_denom,
            footoken_meta,
        )
        .await;
        submit_and_pass_token_proposal(
            contact,
            erc20_qc.clone(),
            validator_keys,
            registered_erc20.to_string(),
            true,
        )
        .await;
    } else {
        info!("Skipping coin registration governance due to SKIP_GOV");
    }
    let erc20_pair = erc20_qc
        .token_pair(QueryTokenPairRequest {
            token: registered_erc20.to_string(),
        })
        .await
        .expect("could not query token pair")
        .into_inner()
        .token_pair
        .expect("token pair response had no token pair");
    let erc20_denom = erc20_pair.denom;

    if !SKIP_GOV {
        // Match the other gasfree params so this test does not trigger a gov proposal
        configure_gasfree_params_if_not_configured(
            contact,
            &[registered_denom.clone(), registered_erc20.to_string()],
            fee_basis_points,
            validator_keys,
            expect_failure,
        )
        .await;
    } else {
        info!("Skipping gasfree parameter change governance due to SKIP_GOV");
    }
    let althea_channel_qc = IbcChannelQueryClient::connect(COSMOS_NODE_GRPC.as_str())
        .await
        .expect("Could not connect channel query client");

    // Wait for the ibc channel to be created and find the channel ids
    info!("Waiting for IBC channel creation:");
    let channel_id_timeout = Duration::from_secs(60 * 5);
    let althea_channel = get_channel(
        althea_channel_qc,
        get_ibc_chain_id(),
        Some(channel_id_timeout),
    )
    .await
    .expect("Could not find ibc-test-1 <-> althea_6633438-1 channel");

    let start_balance = contact
        .get_balance(user.ethermint_address, erc20_denom.clone())
        .await
        .unwrap()
        .unwrap();

    let msg = MsgSendErc20ToCosmosAndIbcTransfer {
        erc20: registered_erc20.to_string(),
        amount: start_balance.amount.to_string(),
        sender: user.eth_address.to_string(),
        destination_port: althea_channel.port_id.clone(),
        destination_channel: althea_channel.channel_id.clone(),
        destination_receiver: ibc_address.to_string(),
    };
    info!(
        "Sending gasfree MsgSendErc20ToCosmosAndIBCTransfer with insufficient balance: {:?}",
        msg
    );
    let cosmos_msg = Msg::new(MSG_SEND_ERC20_TO_COSMOS_AND_IBC_TRANSFER_TYPE_URL, msg);
    let res = contact
        .send_message(
            &[cosmos_msg],
            None,
            &[], // no fee coin intentionally
            Some(OPERATION_TIMEOUT),
            None,
            user.ethermint_key,
        )
        .await;

    if !expect_failure {
        assert!(
            res.is_err(),
            "MsgSendErc20ToCosmosAndIBCTransfer should fail due to insufficient balance"
        );

        let end_balance = contact
            .get_balance(user.ethermint_address, erc20_denom.clone())
            .await
            .unwrap()
            .unwrap();

        assert!(
        end_balance.amount.eq(&start_balance.amount),
        "User ERC20 balance should be unchanged after failed MsgSendErc20ToCosmosAndIBCTransfer"
    );
    } else {
        res.expect_err("expected MsgSendErc20ToCosmosAndIBCTransfer to fail");
    }

    info!("Gasfree MsgSendErc20ToCosmosAndIBCTransfer insufficient balance test successful");
}
