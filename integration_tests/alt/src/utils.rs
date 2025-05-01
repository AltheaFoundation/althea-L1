use std::{
    collections::HashMap,
    str::FromStr,
    time::{Duration, SystemTime},
};

use clarity::{abi::encode_tokens, utils::bytes_to_hex_str, Address as EthAddress, PrivateKey};
use num256::Uint256;
use num_traits::ToPrimitive;
use serde::{Deserialize, Serialize};
use test_runner::dex_utils::{COLD_PATH, LONG_PATH};
use web30::client::Web3;

use crate::long_path::pack_order;

pub async fn approve_erc20s(
    web30: &Web3,
    dex_contract: EthAddress,
    wallet: PrivateKey,
    base: EthAddress,
    quote: EthAddress,
    amount: Uint256,
) {
    let wallet_addr = wallet.to_address();
    if base != EthAddress::default() {
        let allowance = web30
            .get_erc20_allowance(base, wallet_addr, dex_contract)
            .await
            .expect("Failed to get base ERC20 allowance");
        if allowance < amount {
            info!("Approving DEX to spend {} base token wei", amount);
            web30
                .erc20_approve(
                    base,
                    amount,
                    wallet,
                    dex_contract,
                    Some(Duration::from_secs(20)),
                    vec![],
                )
                .await
                .expect("Failed to approve DEX to spend base token");
        }
    }
    let allowance = web30
        .get_erc20_allowance(quote, wallet_addr, dex_contract)
        .await
        .expect("Failed to get quote ERC20 allowance");
    if allowance < amount {
        info!("Approving DEX to spend {} quote token wei", amount);
        web30
            .erc20_approve(
                quote,
                amount,
                wallet,
                dex_contract,
                Some(Duration::from_secs(20)),
                vec![],
            )
            .await
            .expect("Failed to approve DEX to spend base token");
    }
}

#[derive(Debug, Serialize, Deserialize)]
#[allow(non_snake_case)]
pub struct Domain {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub version: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub chainId: Option<u128>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub verifyingContract: Option<EthAddress>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub salt: Option<[u8; 32]>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Eip712Type {
    pub name: String,
    #[serde(rename = "type")]
    pub type_: String,
}

#[allow(non_snake_case)]
#[derive(Debug, Serialize, Deserialize)]
pub struct TypedData<T> {
    pub types: HashMap<String, Vec<Eip712Type>>,
    pub primaryType: String,
    pub domain: Domain,
    pub message: HashMap<String, T>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct PermitMessage {
    pub owner: EthAddress,
    pub spender: EthAddress,
    pub value: u128,
    pub nonce: u128,
    pub deadline: u64,
}
pub const PERMIT_PRIMARY_TYPE: &str = "Permit";
pub fn get_permit_types() -> Vec<Eip712Type> {
    vec![
        Eip712Type {
            name: "owner".to_string(),
            type_: "address".to_string(),
        },
        Eip712Type {
            name: "spender".to_string(),
            type_: "address".to_string(),
        },
        Eip712Type {
            name: "value".to_string(),
            type_: "uint256".to_string(),
        },
        Eip712Type {
            name: "nonce".to_string(),
            type_: "uint256".to_string(),
        },
        Eip712Type {
            name: "deadline".to_string(),
            type_: "uint256".to_string(),
        },
    ]
}

pub async fn create_permit_transaction(
    web30: &Web3,
    signer: EthAddress,
    token: EthAddress,
    spender: EthAddress,
    amount: u128,
    querier: Option<EthAddress>,
) -> TypedData<PermitMessage> {
    let nonce = web30
        .get_erc20_nonces(token, signer, querier.unwrap_or(signer))
        .await
        .expect("Failed to get nonce")
        .to_u128()
        .unwrap();
    let deadline = one_day_deadline();

    let domain = web30
        .get_eip712_domain(token, querier.unwrap_or(signer))
        .await
        .unwrap();

    let message = PermitMessage {
        owner: signer,
        spender,
        value: amount,
        nonce,
        deadline,
    };

    TypedData {
        types: HashMap::from([(PERMIT_PRIMARY_TYPE.to_string(), get_permit_types())]),
        primaryType: PERMIT_PRIMARY_TYPE.to_string(),
        domain: Domain {
            name: domain.name,
            version: domain.version,
            chainId: domain.chainId.map(|v| v.to_u128().unwrap()),
            verifyingContract: domain.verifyingContract,
            salt: domain.salt,
        },
        message: HashMap::from([("Permit".to_string(), message)]),
    }
}

fn one_day_deadline() -> u64 {
    const ONE_DAY_SECS: u64 = 24 * 60 * 60;
    SystemTime::now()
        .duration_since(SystemTime::UNIX_EPOCH)
        .expect("Could not get current time")
        .as_secs()
        + ONE_DAY_SECS
}

pub fn get_gasless_types() -> Vec<Eip712Type> {
    vec![
        Eip712Type {
            name: "callpath".to_string(),
            type_: "uint8".to_string(),
        },
        Eip712Type {
            name: "cmd".to_string(),
            type_: "bytes".to_string(),
        },
        Eip712Type {
            name: "conds".to_string(),
            type_: "bytes".to_string(),
        },
        Eip712Type {
            name: "tip".to_string(),
            type_: "bytes".to_string(),
        },
    ]
}
pub const GASLESS_PRIMARY_TYPE: &str = "CrocRelayerCall";

#[derive(Debug, Serialize, Deserialize)]
pub struct GaslessMessage {
    pub callpath: u16,
    pub cmd: String,
    pub conds: String,
    pub tip: String,
}

pub fn get_gasless_message(
    callpath: u16,
    cmd_bytes: Vec<u8>,
    conds_bytes: Vec<u8>,
    tip_bytes: Vec<u8>,
) -> HashMap<String, GaslessMessage> {
    HashMap::from([(
        GASLESS_PRIMARY_TYPE.to_string(),
        GaslessMessage {
            callpath,
            cmd: format!("0x{}", bytes_to_hex_str(&cmd_bytes)),
            conds: format!("0x{}", bytes_to_hex_str(&conds_bytes)),
            tip: format!("0x{}", bytes_to_hex_str(&tip_bytes)),
        },
    )])
}

pub fn create_gasless_deposit_with_permit(
    dex_address: EthAddress,
    depositor: EthAddress,
    token: EthAddress,
    amount: u128,
    permit_sig_v: u8,
    permit_sig_r: [u8; 32],
    permit_sig_s: [u8; 32],
    permit: PermitMessage,
    tip: Uint256,
    chain_id: Uint256,
) -> TypedData<GaslessMessage> {
    let cmd_code: u16 = 83;
    // cmd is the abi encoded [cmd, recv, value, token, deadline, v, r, s] arguments
    let cmd = encode_tokens(&[
        cmd_code.into(),
        depositor.into(),
        amount.into(),
        token.into(),
        permit.deadline.into(),
        permit_sig_v.into(),
        Uint256::from_be_bytes(&permit_sig_r).into(), // Need bytes32 encoding
        Uint256::from_be_bytes(&permit_sig_s).into(), // Need bytes32 encoding
    ]);

    let mut rng = rand::rng();
    let salt: [u8; 32] = rand::Rng::random(&mut rng);
    let conds = encode_gasless_conds(
        permit.deadline.to_u64().unwrap(),
        0,
        salt,
        0,
        EthAddress::default(),
    );

    let tip_relayer = EthAddress::from_str("0x0000000000000000000000000000000000000100").unwrap();
    let tip = encode_gasless_tip(token, tip, tip_relayer);

    let domain = crocswap_domain(chain_id, dex_address);

    let types = HashMap::from([("CrocRelayerCall".to_string(), get_gasless_types())]);
    let message = get_gasless_message(COLD_PATH, cmd, conds, tip);

    TypedData {
        types,
        primaryType: GASLESS_PRIMARY_TYPE.to_string(),
        domain,
        message,
    }
}

pub fn create_gasless_swap(
    dex_address: EthAddress,
    base_token: EthAddress,
    quote_token: EthAddress,
    pool_idx: Uint256,
    is_buy: bool,
    qty: u128,
    in_base_qty: bool,
    limit_price: u128,
    min_out: u128,
    tip: Uint256,
    chain_id: Uint256,
) -> TypedData<GaslessMessage> {
    let order = pack_order(
        base_token,
        quote_token,
        pool_idx,
        is_buy,
        qty.into(),
        in_base_qty,
        limit_price.into(),
        min_out,
        true,
        false,
    );
    let cmd_bytes = order.encode_bytes();

    let mut rng = rand::rng();
    let salt: [u8; 32] = rand::Rng::random(&mut rng);
    let conds_bytes = encode_gasless_conds(one_day_deadline(), 0, salt, 0, EthAddress::default());

    let tip_relayer = EthAddress::from_str("0x0000000000000000000000000000000000000100").unwrap();
    let tip_token = if is_buy {
        quote_token // output of swap
    } else {
        base_token // output of swap
    };
    let tip_bytes = encode_gasless_tip(tip_token, tip, tip_relayer);

    let domain = crocswap_domain(chain_id, dex_address);

    let types = HashMap::from([("CrocRelayerCall".to_string(), get_gasless_types())]);
    let message = get_gasless_message(LONG_PATH, cmd_bytes, conds_bytes, tip_bytes);

    TypedData {
        types,
        primaryType: GASLESS_PRIMARY_TYPE.to_string(),
        domain,
        message,
    }
}

pub fn encode_gasless_conds(
    deadline: u64,
    alive: u64,
    salt: [u8; 32],
    nonce: u8,
    conds_relayer: EthAddress,
) -> Vec<u8> {
    encode_tokens(&[
        Uint256::from(deadline).into(),
        alive.into(),
        Uint256::from_be_bytes(&salt).into(), // Need bytes32 encoding
        nonce.into(),
        conds_relayer.into(),
    ])
}

pub fn encode_gasless_tip(token: EthAddress, amount: Uint256, relayer: EthAddress) -> Vec<u8> {
    encode_tokens(&[token.into(), amount.into(), relayer.into()])
}

pub fn crocswap_domain(chain_id: Uint256, dex_address: EthAddress) -> Domain {
    Domain {
        name: Some("CrocSwap".to_string()),
        version: Some("1.0".to_string()),
        chainId: Some(chain_id.to_u128().unwrap()),
        verifyingContract: Some(dex_address),
        salt: None,
    }
}
