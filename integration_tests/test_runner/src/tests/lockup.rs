use std::time::SystemTime;

use crate::type_urls::{
    GENERIC_AUTHORIZATION_TYPE_URL, MSG_EXEC_TYPE_URL, MSG_GRANT_TYPE_URL, MSG_MICROTX_TYPE_URL,
    MSG_MULTI_SEND_TYPE_URL, MSG_SEND_TYPE_URL, MSG_TRANSFER_TYPE_URL,
};
use crate::utils::{
    create_parameter_change_proposal, encode_any, footoken_metadata, get_user_key, one_atom,
    send_funds_bulk, vote_yes_on_proposals, wait_for_proposals_to_execute, EthermintUserKey,
    ValidatorKeys, ADDRESS_PREFIX, OPERATION_TIMEOUT, STAKING_TOKEN,
};
use althea_proto::althea::microtx::v1::MsgMicrotx;
use althea_proto::cosmos_sdk_proto::cosmos::authz::v1beta1::{
    GenericAuthorization, Grant, MsgExec, MsgGrant,
};
use althea_proto::cosmos_sdk_proto::cosmos::bank::v1beta1::{Input, MsgMultiSend, MsgSend, Output};
use althea_proto::cosmos_sdk_proto::cosmos::base::v1beta1::Coin as ProtoCoin;
use althea_proto::cosmos_sdk_proto::cosmos::params::v1beta1::ParamChange;
use clarity::Uint256;
use deep_space::error::CosmosGrpcError;
use deep_space::{Address, Coin, Contact, Msg, PrivateKey};

/// These *_PARAM_KEY constants are defined in x/lockup/types/types.go and must match those values exactly
pub const LOCKED_PARAM_KEY: &str = "locked";
pub const LOCK_EXEMPT_PARAM_KEY: &str = "lockExempt";
pub const LOCKED_MSG_TYPES_PARAM_KEY: &str = "lockedMessageTypes";
pub const LOCKED_TOKEN_DENOMS_PARAM_KEY: &str = "lockedTokenDenoms";

/// Simulates the launch lockup process by setting the lockup module params via governance,
/// attempting to transfer tokens a variety of ways, and finally clearing the lockup module params
/// and asserting that balances can successfully be transferred
pub async fn lockup_test(contact: &Contact, validator_keys: Vec<ValidatorKeys>) {
    let lock_exempt = get_user_key(None);
    let msg_send_authorized = get_user_key(None);
    let msg_multi_send_authorized = get_user_key(None);
    let msg_microtx_authorized = get_user_key(None);
    fund_lock_exempt_user(contact, &validator_keys, lock_exempt).await;
    fund_authorized_users(
        contact,
        &validator_keys,
        msg_send_authorized,
        msg_multi_send_authorized,
    )
    .await;
    lockup_the_chain(contact, &validator_keys, &lock_exempt).await;

    // TODO: Add ibc transfer and check that transfers are blocked outbound for aalthea
    fail_to_send(
        contact,
        &validator_keys,
        [
            msg_send_authorized,
            msg_multi_send_authorized,
            msg_microtx_authorized,
        ],
    )
    .await;
    send_from_lock_exempt(contact, lock_exempt).await;
    send_unlocked_token(contact, &validator_keys).await;

    unlock_the_chain(contact, &validator_keys).await;
    successfully_send(contact, &validator_keys, lock_exempt).await;
}

async fn fund_lock_exempt_user(
    contact: &Contact,
    validator_keys: &[ValidatorKeys],
    lock_exempt: EthermintUserKey,
) {
    let sender = validator_keys.first().unwrap().validator_key;
    let amount = Coin {
        denom: STAKING_TOKEN.clone(),
        amount: one_atom() * 100u16.into(),
    };

    info!("Funding lock exempt user {}", lock_exempt.ethermint_address);
    contact
        .send_coins(
            amount.clone(),
            Some(amount),
            lock_exempt.ethermint_address,
            Some(OPERATION_TIMEOUT),
            sender,
        )
        .await
        .expect("Unable to send funds to lock exempt user!");
}

async fn fund_authorized_users(
    contact: &Contact,
    validator_keys: &[ValidatorKeys],
    auth_1: EthermintUserKey,
    auth_2: EthermintUserKey,
) {
    let sender = validator_keys.first().unwrap().validator_key;
    let amount = Coin {
        denom: STAKING_TOKEN.clone(),
        amount: one_atom(),
    };
    info!("Funding auth_1 user {}", auth_1.ethermint_address);
    contact
        .send_coins(
            amount.clone(),
            Some(amount.clone()),
            auth_1.ethermint_address,
            Some(OPERATION_TIMEOUT),
            sender,
        )
        .await
        .expect("Unable to send funds to auth_1 user!");
    info!("Funding auth_2 user {}", auth_2.ethermint_address);
    contact
        .send_coins(
            amount.clone(),
            Some(amount),
            auth_2.ethermint_address,
            Some(OPERATION_TIMEOUT),
            sender,
        )
        .await
        .expect("Unable to send funds to auth_2 user!");
}

pub async fn lockup_the_chain(
    contact: &Contact,
    validator_keys: &[ValidatorKeys],
    lock_exempt: &EthermintUserKey,
) {
    let to_change = create_lockup_param_changes(lock_exempt.ethermint_address);
    let proposer = validator_keys.first().unwrap();
    let zero_fee = Coin {
        denom: STAKING_TOKEN.clone(),
        amount: 0u8.into(),
    };
    create_parameter_change_proposal(contact, proposer.validator_key, to_change, zero_fee).await;

    vote_yes_on_proposals(contact, validator_keys, Some(OPERATION_TIMEOUT)).await;
    wait_for_proposals_to_execute(contact).await;
}

pub fn create_lockup_param_changes(exempt_user: Address) -> Vec<ParamChange> {
    // Params{lock_exempt:, locked: false, locked_message_types: Vec::new() };
    let lockup_param = ParamChange {
        subspace: "lockup".to_string(),
        key: String::new(),
        value: String::new(),
    };
    let mut locked = lockup_param.clone();
    locked.key = LOCKED_PARAM_KEY.to_string();
    locked.value = format!("{}", true);

    let mut lock_exempt = lockup_param.clone();
    lock_exempt.key = LOCK_EXEMPT_PARAM_KEY.to_string();
    lock_exempt.value = serde_json::to_string(&vec![exempt_user.to_string()]).unwrap();

    let locked_msgs = vec![
        MSG_SEND_TYPE_URL.to_string(),
        MSG_MULTI_SEND_TYPE_URL.to_string(),
        MSG_MICROTX_TYPE_URL.to_string(),
        MSG_TRANSFER_TYPE_URL.to_string(),
    ];
    let mut locked_msg_types = lockup_param.clone();
    locked_msg_types.key = LOCKED_MSG_TYPES_PARAM_KEY.to_string();
    locked_msg_types.value = serde_json::to_string(&locked_msgs).unwrap();

    let tokens = vec![STAKING_TOKEN.clone()];
    let mut locked_tokens = lockup_param;
    locked_tokens.key = LOCKED_TOKEN_DENOMS_PARAM_KEY.to_string();
    locked_tokens.value = serde_json::to_string(&tokens).unwrap();

    vec![locked, lock_exempt, locked_msg_types, locked_tokens]
}

pub async fn fail_to_send(
    contact: &Contact,
    validator_keys: &[ValidatorKeys],
    authorized_users: [EthermintUserKey; 3],
) {
    let sender = validator_keys.first().unwrap().validator_key;
    let receiver = get_user_key(None);
    let amount = ProtoCoin {
        denom: STAKING_TOKEN.clone(),
        amount: one_atom().to_string(),
    };

    let msg_send = create_bank_msg_send(sender, receiver.ethermint_address, amount.clone());
    let res = contact
        .send_message(&[msg_send], None, &[], Some(OPERATION_TIMEOUT), sender)
        .await;
    res.expect_err("Successfully sent via bank MsgSend? Should not be possible!");
    let msg_multi_send =
        create_bank_msg_multi_send(sender, receiver.ethermint_address, amount.clone());
    let res = contact
        .send_message(
            &[msg_multi_send],
            None,
            &[],
            Some(OPERATION_TIMEOUT),
            sender,
        )
        .await;
    res.expect_err("Successfully sent via bank MsgMultiSend? Should not be possible!");
    let msg_microtx =
        create_microtx_msg_microtx(sender, receiver.ethermint_address, amount.clone());
    let res = contact
        .send_message(&[msg_microtx], None, &[], Some(OPERATION_TIMEOUT), sender)
        .await;
    res.expect_err("Successfully sent via microtx MsgMicrotx? Should not be possible!");
    let msg_send_authorized = authorized_users[0];
    let authz_send = create_authz_bank_msg_send(
        contact,
        sender,
        msg_send_authorized,
        receiver.ethermint_address,
        amount.clone(),
    )
    .await
    .unwrap();
    let res = contact
        .send_message(
            &[authz_send.clone()],
            None,
            &[],
            Some(OPERATION_TIMEOUT),
            msg_send_authorized.ethermint_key,
        )
        .await;
    res.expect_err("Successfully sent via authz Exec(MsgSend)? Should not be possible!");
    let msg_multi_send_authorized = authorized_users[1];
    let authz_multi_send = create_authz_bank_msg_multi_send(
        contact,
        sender,
        msg_multi_send_authorized,
        receiver.ethermint_address,
        amount.clone(),
    )
    .await
    .unwrap();
    let res = contact
        .send_message(
            &[authz_multi_send.clone()],
            None,
            &[],
            Some(OPERATION_TIMEOUT),
            msg_multi_send_authorized.ethermint_key,
        )
        .await;
    res.expect_err("Successfully sent via authz Exec(MsgMultiSend)? Should not be possible!");
    let msg_microtx_authorized = authorized_users[1];
    let authz_msg_microtx = create_authz_microtx_msg_microtx(
        contact,
        sender,
        msg_microtx_authorized,
        receiver.ethermint_address,
        amount.clone(),
    )
    .await
    .unwrap();
    let res = contact
        .send_message(
            &[authz_msg_microtx.clone()],
            None,
            &[],
            Some(OPERATION_TIMEOUT),
            msg_microtx_authorized.ethermint_key,
        )
        .await;
    res.expect_err("Successfully sent via authz Exec(MsgMicrotx)? Should not be possible!");
}

/// Creates a x/bank MsgSend to transfer `amount` from `sender` to `receiver`
pub fn create_bank_msg_send(sender: impl PrivateKey, receiver: Address, amount: ProtoCoin) -> Msg {
    let send = MsgSend {
        from_address: sender.to_address(&ADDRESS_PREFIX).unwrap().to_string(),
        to_address: receiver.to_string(),
        amount: vec![amount],
    };
    Msg::new(MSG_SEND_TYPE_URL, send)
}

/// Creates a x/bank MsgMultiSend to transfer `amount` from `sender` to `receiver`
pub fn create_bank_msg_multi_send(
    sender: impl PrivateKey,
    receiver: Address,
    amount: ProtoCoin,
) -> Msg {
    let input = Input {
        address: sender.to_address(&ADDRESS_PREFIX).unwrap().to_string(),
        coins: vec![amount.clone()],
    };
    let output = Output {
        address: receiver.to_string(),
        coins: vec![amount],
    };
    let multi_send = MsgMultiSend {
        inputs: vec![input],
        outputs: vec![output],
    };

    Msg::new(MSG_MULTI_SEND_TYPE_URL, multi_send)
}

/// Creates a x/microtx MsgMicrotx to transfer `amount` from `sender` to `receiver`
pub fn create_microtx_msg_microtx(
    sender: impl PrivateKey,
    receiver: Address,
    amount: ProtoCoin,
) -> Msg {
    let send = MsgMicrotx {
        sender: sender.to_address(&ADDRESS_PREFIX).unwrap().to_string(),
        receiver: receiver.to_string(),
        amount: Some(amount),
    };
    Msg::new(MSG_MICROTX_TYPE_URL, send)
}

/// Submits an Authorization using x/authz to give the returned private key control over `sender`'s tokens, then crafts
/// an authz MsgExec-wrapped bank MsgSend and returns that as well
pub async fn create_authz_bank_msg_send(
    contact: &Contact,
    sender: impl PrivateKey,
    authorizee: EthermintUserKey,
    receiver: Address,
    amount: ProtoCoin,
) -> Result<Msg, CosmosGrpcError> {
    let grant_msg_send = create_authorization(
        sender.clone(),
        authorizee.ethermint_address,
        MSG_SEND_TYPE_URL.to_string(),
    );

    let res = contact
        .send_message(
            &[grant_msg_send],
            None,
            &[],
            Some(OPERATION_TIMEOUT),
            sender.clone(),
        )
        .await;
    info!("Granted MsgSend authorization with response {:?}", res);
    res?;

    let send = create_bank_msg_send(sender.clone(), receiver, amount);
    let send_any: prost_types::Any = send.into();
    let exec = MsgExec {
        grantee: authorizee.ethermint_address.to_string(),
        msgs: vec![send_any],
    };
    let exec_msg = Msg::new(MSG_EXEC_TYPE_URL, exec);

    Ok(exec_msg)
}

/// Submits an Authorization using x/authz to give the returned private key control over `sender`'s tokens, then crafts
/// an authz MsgExec-wrapped bank MsgMultiSend and returns that as well
pub async fn create_authz_bank_msg_multi_send(
    contact: &Contact,
    sender: impl PrivateKey,
    authorizee: EthermintUserKey,
    receiver: Address,
    amount: ProtoCoin,
) -> Result<Msg, CosmosGrpcError> {
    let grant_msg_multi_send = create_authorization(
        sender.clone(),
        authorizee.ethermint_address,
        MSG_MULTI_SEND_TYPE_URL.to_string(),
    );

    let res = contact
        .send_message(
            &[grant_msg_multi_send],
            None,
            &[],
            Some(OPERATION_TIMEOUT),
            sender.clone(),
        )
        .await;
    info!("Granted MsgSend authorization with response {:?}", res);
    res?;

    let multi_send = create_bank_msg_multi_send(sender.clone(), receiver, amount);
    let multi_send_any: prost_types::Any = multi_send.into();
    let exec = MsgExec {
        grantee: authorizee.ethermint_address.to_string(),
        msgs: vec![multi_send_any],
    };
    let exec_msg = Msg::new(MSG_EXEC_TYPE_URL, exec);

    Ok(exec_msg)
}

/// Submits an Authorization using x/authz to give the returned private key control over `sender`'s tokens, then crafts
/// an authz MsgExec-wrapped microtx MsgMicrotx and returns that as well
pub async fn create_authz_microtx_msg_microtx(
    contact: &Contact,
    sender: impl PrivateKey,
    authorizee: EthermintUserKey,
    receiver: Address,
    amount: ProtoCoin,
) -> Result<Msg, CosmosGrpcError> {
    let grant_msg_microtx = create_authorization(
        sender.clone(),
        authorizee.ethermint_address,
        MSG_MICROTX_TYPE_URL.to_string(),
    );

    let res = contact
        .send_message(
            &[grant_msg_microtx],
            None,
            &[],
            Some(OPERATION_TIMEOUT),
            sender.clone(),
        )
        .await;
    info!("Granted MsgMicrotx authorization with response {:?}", res);
    res?;

    let microtx = create_microtx_msg_microtx(sender.clone(), receiver, amount);
    let microtx_any: prost_types::Any = microtx.into();
    let exec = MsgExec {
        grantee: authorizee.ethermint_address.to_string(),
        msgs: vec![microtx_any],
    };
    let exec_msg = Msg::new(MSG_EXEC_TYPE_URL, exec);

    Ok(exec_msg)
}

/// Creates a MsgGrant to give a GenericAuthorization for `authorizee` to submit any Msg with the given `msg_type_url`
/// on behalf of `authorizer`
pub fn create_authorization(
    authorizer: impl PrivateKey,
    authorizee: Address,
    msg_type_url: String,
) -> Msg {
    let granter = authorizer.to_address(&ADDRESS_PREFIX).unwrap().to_string();

    // The authorization we want to store
    let auth = GenericAuthorization { msg: msg_type_url };
    let auth_any = encode_any(auth, GENERIC_AUTHORIZATION_TYPE_URL.to_string());

    let now = SystemTime::now()
        .duration_since(SystemTime::UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;
    let expir = prost_types::Timestamp {
        seconds: now + 60 * 60 * 24 * 365 * 4,
        nanos: 0,
    }; // 4 years
       // The authorization and any associated auth expiration
    let grant = Grant {
        authorization: Some(auth_any),
        expiration: Some(expir),
    };

    // The msg which must be submitted by the granter to give the grantee the specific authorization (with expiration)
    let msg_grant = MsgGrant {
        granter,
        grantee: authorizee.to_string(),
        grant: Some(grant),
    };

    Msg::new(MSG_GRANT_TYPE_URL, msg_grant)
}

async fn send_from_lock_exempt(contact: &Contact, lock_exempt: EthermintUserKey) {
    let amount = Coin {
        denom: STAKING_TOKEN.clone(),
        amount: one_atom(),
    };

    send_from_and_assert_balance_changes(contact, lock_exempt.ethermint_key, amount).await;
}

pub async fn send_from_and_assert_balance_changes(
    contact: &Contact,
    from: impl PrivateKey,
    amount: Coin,
) {
    let receiver = get_user_key(None);
    let pre_balance = contact
        .get_balance(receiver.ethermint_address, amount.denom.clone())
        .await
        .unwrap();
    send_funds_bulk(
        contact,
        from.clone(),
        &[receiver.ethermint_address],
        amount.clone(),
        Some(OPERATION_TIMEOUT),
    )
    .await
    .unwrap();
    let post_balance = contact
        .get_balance(receiver.ethermint_address, amount.denom.clone())
        .await
        .unwrap();
    assert_balance_changes(pre_balance, post_balance, amount.amount);
}

pub fn assert_balance_changes(
    pre_balance: Option<Coin>,
    post_balance: Option<Coin>,
    expected_amount: Uint256,
) {
    let diff: Uint256 = match (pre_balance, post_balance) {
        (Some(pre), Some(post)) => {
            if post.amount < pre.amount {
                panic!("Unexpected lesser balance!");
            }
            post.amount - pre.amount
        }
        (None, Some(post)) => post.amount,
        (_, _) => {
            panic!("Unexpected balance change!");
        }
    };
    if diff != expected_amount {
        panic!("Unexpected diff: {}, expected {}", diff, expected_amount);
    }
}

async fn unlock_the_chain(contact: &Contact, validator_keys: &[ValidatorKeys]) {
    let unlock = ParamChange {
        subspace: "lockup".to_string(),
        key: LOCKED_PARAM_KEY.to_string(),
        value: format!("{}", false),
    };
    let proposer = validator_keys.first().unwrap();
    let zero_fee = Coin {
        denom: STAKING_TOKEN.clone(),
        amount: 0u8.into(),
    };
    create_parameter_change_proposal(contact, proposer.validator_key, vec![unlock], zero_fee).await;

    vote_yes_on_proposals(contact, validator_keys, Some(OPERATION_TIMEOUT)).await;
    wait_for_proposals_to_execute(contact).await;
}

async fn successfully_send(
    contact: &Contact,
    validator_keys: &[ValidatorKeys],
    lock_exempt: EthermintUserKey,
) {
    let val0 = validator_keys.first().unwrap().validator_key;
    let amount = Coin {
        denom: STAKING_TOKEN.clone(),
        amount: one_atom(),
    };
    send_from_and_assert_balance_changes(contact, val0, amount.clone()).await;
    send_from_and_assert_balance_changes(contact, lock_exempt.ethermint_key, amount.clone()).await;
}

async fn send_unlocked_token(contact: &Contact, validator_keys: &[ValidatorKeys]) {
    let val0 = validator_keys.first().unwrap().validator_key;
    let amount = Coin {
        denom: footoken_metadata(contact).await.base,
        amount: one_atom(),
    };
    send_from_and_assert_balance_changes(contact, val0, amount.clone()).await;
}
