use althea_proto::{
    canto::erc20::v1::{RegisterCoinProposal, RegisterErc20Proposal},
    cosmos_sdk_proto::cosmos::{
        bank::v1beta1::Metadata,
        gov::v1beta1::VoteOption,
        params::v1beta1::{ParamChange, ParameterChangeProposal},
        staking::v1beta1::{DelegationResponse, QueryValidatorsRequest},
        upgrade::v1beta1::{Plan, SoftwareUpgradeProposal},
    },
};
use bytes::BytesMut;

use clarity::{Address as EthAddress, PrivateKey as EthPrivateKey, Uint256};
use deep_space::address::Address as CosmosAddress;
use deep_space::client::{types::LatestBlock, ChainStatus};
use deep_space::coin::Coin;
use deep_space::error::CosmosGrpcError;
use deep_space::private_key::{CosmosPrivateKey, PrivateKey};
use deep_space::{Contact, EthermintPrivateKey};
use futures::future::join_all;
use prost::{DecodeError, Message};
use prost_types::Any;
use rand::Rng;
use std::{convert::TryInto, env};
use std::{
    str::FromStr,
    time::{Duration, Instant},
};
use tokio::time::sleep;
use web30::{client::Web3, jsonrpc::error::Web3Error, types::SendTxOption};

/// the timeout for individual requests
pub const OPERATION_TIMEOUT: Duration = Duration::from_secs(30);
/// the timeout for the total system
pub const TOTAL_TIMEOUT: Duration = Duration::from_secs(300);
// The config file location for hermes
pub const HERMES_CONFIG: &str = "/althea/tests/assets/ibc-relayer-config.toml";

/// this value reflects the contents of /tests/container-scripts/setup-validator.sh
/// and is used to compute if a stake change is big enough to trigger a validator set
/// update since we want to make several such changes intentionally
pub const STAKE_SUPPLY_PER_VALIDATOR: u128 = 1000000000;
/// this is the amount each validator bonds at startup
pub const STARTING_STAKE_PER_VALIDATOR: u128 = STAKE_SUPPLY_PER_VALIDATOR / 2;
// Retrieve values from runtime ENV vars
lazy_static! {
    // ALTHEA CHAIN CONSTANTS
    // These constants all apply to the althea instance running (althea-test-1)
    pub static ref ADDRESS_PREFIX: String =
        env::var("ADDRESS_PREFIX").unwrap_or_else(|_| "althea".to_string());
    pub static ref STAKING_TOKEN: String =
        env::var("STAKING_TOKEN").unwrap_or_else(|_| "aalthea".to_owned());
    pub static ref COSMOS_NODE_GRPC: String =
        env::var("COSMOS_NODE_GRPC").unwrap_or_else(|_| "http://localhost:9090".to_owned());
    pub static ref COSMOS_NODE_ABCI: String =
        env::var("COSMOS_NODE_ABCI").unwrap_or_else(|_| "http://localhost:26657".to_owned());

    // IBC CHAIN CONSTANTS
    // These constants all apply to the gaiad instance running (ibc-test-1)
    pub static ref IBC_ADDRESS_PREFIX: String =
        env::var("IBC_ADDRESS_PREFIX").unwrap_or_else(|_| "cosmos".to_string());
    pub static ref IBC_STAKING_TOKEN: String =
        env::var("IBC_STAKING_TOKEN").unwrap_or_else(|_| "stake".to_owned());
    pub static ref IBC_NODE_GRPC: String =
        env::var("IBC_NODE_GRPC").unwrap_or_else(|_| "http://localhost:9190".to_owned());
    pub static ref IBC_NODE_ABCI: String =
        env::var("IBC_NODE_ABCI").unwrap_or_else(|_| "http://localhost:27657".to_owned());

    // LOCAL ETHEREUM CONSTANTS
    pub static ref ETH_NODE: String =
        env::var("ETH_NODE").unwrap_or_else(|_| "http://localhost:8545".to_owned());
    pub static ref MINER_PRIVATE_KEY: EthPrivateKey =
        "0xb1bab011e03a9862664706fc3bbaa1b16651528e5f0e7fbfcbfdd8be302a13e7"
            .parse()
            .unwrap();
    pub static ref MINER_ETH_ADDRESS: EthAddress = MINER_PRIVATE_KEY.to_address();
    pub static ref MINER_COSMOS_ADDRESS: CosmosAddress = CosmosAddress::from_bech32("althea1hanqss6jsq66tfyjz56wz44z0ejtyv0768q8r4".to_string()).unwrap();
    pub static ref EVM_USER_KEYS: Vec<EthermintUserKey> = get_funded_evm_users();
}

/// Parses the DEPLOY_CONTRACTS env variable and determines if the ethereum contracts must be deployed
pub fn should_deploy_contracts() -> bool {
    match env::var("DEPLOY_CONTRACTS") {
        Ok(s) => s == "1" || s.to_lowercase() == "yes" || s.to_lowercase() == "true",
        _ => false,
    }
}

/// Gets the standard non-token fee for the testnet. We deploy the test chain with STAKE
/// and FOOTOKEN balances by default, one footoken is sufficient for any Cosmos tx fee except
/// fees for send_to_eth messages which have to be of the same bridged denom so that the relayers
/// on the Ethereum side can be paid in that token.
pub fn get_fee(denom: Option<String>) -> Coin {
    match denom {
        None => Coin {
            denom: get_test_token_name(),
            amount: 1u32.into(),
        },
        Some(denom) => Coin {
            denom,
            amount: 1u32.into(),
        },
    }
}

pub fn get_deposit() -> Coin {
    Coin {
        denom: STAKING_TOKEN.to_string(),
        amount: 1_000_000_000u64.into(),
    }
}

pub fn get_test_token_name() -> String {
    "ufootoken".to_string()
}

/// Returns the chain-id of the althea instance running, see ALTHEA CHAIN CONSTANTS above
pub fn get_chain_id() -> String {
    "althea-test-1".to_string()
}

/// Returns the chain-id of the gaiad instance running, see IBC CHAIN CONSTANTS above
pub fn get_ibc_chain_id() -> String {
    "ibc-test-1".to_string()
}

pub fn one_atom() -> Uint256 {
    one_atom_128().into()
}

pub fn one_atom_128() -> u128 {
    1000000u128
}

pub fn one_eth() -> Uint256 {
    one_eth_128().into()
}

pub fn one_eth_128() -> u128 {
    1000000000000000000u128
}

pub fn one_hundred_eth() -> Uint256 {
    (1000000000000000000u128 * 100).into()
}

/// returns the required denom metadata for deployed the Footoken
/// token defined in our test environment
pub async fn footoken_metadata(contact: &Contact) -> Metadata {
    let metadata = contact.get_all_denoms_metadata().await.unwrap();
    for m in metadata {
        if m.base == "ufootoken" || m.display == "footoken" {
            return m;
        }
    }
    panic!("Footoken metadata not set?");
}

pub fn get_decimals(meta: &Metadata) -> u32 {
    for m in meta.denom_units.iter() {
        if m.denom == meta.display {
            return m.exponent;
        }
    }
    panic!("Invalid metadata!")
}

pub fn get_coins(denom: &str, balances: &[Coin]) -> Option<Coin> {
    for coin in balances {
        if coin.denom.starts_with(denom) {
            return Some(coin.clone());
        }
    }
    None
}

/// This is a hardcoded very high gas value used in transaction stress test to counteract rollercoaster
/// gas prices due to the way that test fills blocks
pub const HIGH_GAS_PRICE: u64 = 1_000_000_000u64;

// Generates a new BridgeUserKey through randomly generated secrets
// cosmos_prefix allows for generation of a cosmos_address with a different prefix than "althea"
pub fn get_user_key(cosmos_prefix: Option<&str>) -> CosmosUser {
    *bulk_get_user_keys(cosmos_prefix, 1).get(0).unwrap()
}

// Generates many CosmosUser keys + addresses
pub fn bulk_get_user_keys(cosmos_prefix: Option<&str>, num_users: i64) -> Vec<CosmosUser> {
    let cosmos_prefix = cosmos_prefix.unwrap_or(ADDRESS_PREFIX.as_str());

    let mut rng = rand::thread_rng();
    let mut users = Vec::with_capacity(num_users.try_into().unwrap());
    for _ in 0..num_users {
        let secret: [u8; 32] = rng.gen();
        let cosmos_key = CosmosPrivateKey::from_secret(&secret);
        let cosmos_address = cosmos_key.to_address(cosmos_prefix).unwrap();
        let user = CosmosUser {
            cosmos_address,
            cosmos_key,
        };

        users.push(user)
    }

    users
}

#[derive(Debug, Eq, PartialEq, Clone, Copy, Hash)]
pub struct CosmosUser {
    pub cosmos_address: CosmosAddress,
    pub cosmos_key: CosmosPrivateKey,
}

// Generates a new EthermintUserKey through a randomly generated secret
// cosmos_prefix allows for generation of a cosmos_address with a different prefix than "althea"
pub fn get_ethermint_key(cosmos_prefix: Option<&str>) -> EthermintUserKey {
    let cosmos_prefix = cosmos_prefix.unwrap_or(ADDRESS_PREFIX.as_str());

    let mut rng = rand::thread_rng();
    let secret: [u8; 32] = rng.gen();
    // the starting location of the funds
    // the destination on cosmos that sends along to the final ethereum destination
    let ethermint_key = EthermintPrivateKey::from_secret(&secret);
    let eth_privkey = EthPrivateKey::from_bytes(secret).unwrap();
    let ethermint_address = ethermint_key.to_address(cosmos_prefix).unwrap();
    // TODO: Verify that this conversion works like `evmosd debug addr`
    let eth_address = EthAddress::from_slice(ethermint_address.get_bytes()).unwrap();

    EthermintUserKey {
        ethermint_address,
        ethermint_key,
        eth_privkey,
        eth_address,
    }
}

// Represents an Ethermint account, with address represented in the cosmos-sdk and Ethereum styles
#[derive(Debug, Eq, PartialEq, Clone, Copy)]
pub struct EthermintUserKey {
    pub ethermint_address: CosmosAddress, // the user's address according to ethsecp256k1
    pub ethermint_key: EthermintPrivateKey, // the user's private key
    pub eth_privkey: EthPrivateKey,       // the value from althea keys unsafe-export-eth-key
    pub eth_address: EthAddress,          // the ethermint_address treated as an EthAddress
}

// Returns a vec of EVM users which all have some aalthea
pub fn get_funded_evm_users() -> Vec<EthermintUserKey> {
    let cosm_addrs = [
        "althea1xlcvjwhpku7slrdue6s4zng5xj5dwzemfs0lxj",
        "althea1v5lygpttvvfdksdnrvjuxqv98enut6x83zpu2e",
        "althea1czdncnejmxe2fkw7z7huk6ckh5g0arnp5ts4l3",
        "althea17gv9tajr3dv35h0ah57mxtg9q2epmq6f5zxsxl",
        "althea17aq8r2a92m4kq82z7mnvt8dpcnndks4ezrk3ec",
    ];
    let eth_privkeys = [
        "3b23c86080c9abc8870936b2eb17ecb808f5ad3b318018b3e23873013379e4d6",
        "a9c7120f7a13a0bb0b0c513e6145bc1e4c55a126a055da53c5e7612d25aca8c7",
        "3f4eeb27124d1fcf9bffa1bc2bfa4660f75777dbfc268f0349636e429105aa7f",
        "5791240cd5798ecf4862be2c1c1ae882b80a804e7a3fc615a93910c554b23115",
        "34d97aaf58b1a81d3ed3068a870d8093c6341cf5d1ef7e6efa03fe7f7fc2c3a8",
    ];
    let eth_addrs = [
        "0x37f0c93ae1b73d0f8dbccea1514d1434a8d70b3b",
        "0x653e44056b6312db41b31b25c301853e67c5e8c7",
        "0xc09b3c4f32d9b2a4d9de17afcb6b16bd10fe8e61",
        "0xf21855f6438b591a5dfdbd3db32d0502b21d8349",
        "0xf74071aba556eb601d42f6e6c59da1c4e6db42b9",
    ];
    let mnemonics = [
    "dial point debris employ position cheap inmate nominee crisp grow hello body meadow clever cloth strike agree include dirt tenant hello pattern tattoo option" ,
    "poverty inside weasel way rabbit staff initial fire near machine icon favorite simple address skill couple embark acquire asthma deny census flush ensure shiver",
    "potato apart credit boy canyon walnut mirror inherit note market increase gentle ostrich siege verify clown grab blur rifle inner diagram filter absurd believe",
    "talent rib law noble clog stamp avocado key skull ritual urge metal decorate exist lizard wide section census broken recipe expand unhappy razor small",
    "party normal injury water lecture rude civil disorder hawk split wonder dizzy immense humor couple toilet seed there flip animal lyrics shift give cotton",
    ];
    let mut res = Vec::with_capacity(mnemonics.len());
    for i in 0..mnemonics.len() {
        res.push(EthermintUserKey {
            ethermint_address: CosmosAddress::from_str(cosm_addrs[i]).unwrap(),
            ethermint_key: EthermintPrivateKey::from_phrase(mnemonics[i], "")
                .expect("invalid mnemonic?"),
            eth_privkey: EthPrivateKey::from_str(eth_privkeys[i]).unwrap(),
            eth_address: EthAddress::from_str(eth_addrs[i]).unwrap(),
        });
    }
    res
}

#[derive(Debug, Clone)]
pub struct ValidatorKeys {
    /// The validator key used by this validator to actually sign and produce blocks
    pub validator_key: CosmosPrivateKey,
    // The mnemonic phrase used to generate validator_key
    pub validator_phrase: String,
}

/// Creates a proposal to change the params of our test chain
pub async fn create_parameter_change_proposal(
    contact: &Contact,
    key: impl PrivateKey,
    params_to_change: Vec<ParamChange>,
    fee_coin: Coin,
) {
    let proposal = ParameterChangeProposal {
        title: "Set althea settings!".to_string(),
        description: "test proposal".to_string(),
        changes: params_to_change,
    };
    let res = contact
        .submit_parameter_change_proposal(
            proposal,
            get_deposit(),
            fee_coin,
            key,
            Some(TOTAL_TIMEOUT),
        )
        .await
        .unwrap();
    trace!("Gov proposal executed with {:?}", res);
}

// Prints out current stake to the console
pub async fn print_validator_stake(contact: &Contact) {
    let validators = contact
        .get_validators_list(QueryValidatorsRequest::default())
        .await
        .unwrap();
    for validator in validators {
        info!(
            "Validator {} has {} tokens",
            validator.operator_address, validator.tokens
        );
    }
}

// Simple arguments to create a proposal with
pub struct RegisterErc20ProposalParams {
    pub erc20_address: String,

    pub proposal_title: String,
    pub proposal_desc: String,
}

// Creates and submits a RegisterErc20Proposal to the chain, then votes yes with all validators
pub async fn execute_register_erc20_proposal(
    contact: &Contact,
    keys: &[ValidatorKeys],
    timeout: Option<Duration>,
    erc20_params: RegisterErc20ProposalParams,
) {
    let duration = match timeout {
        Some(dur) => dur,
        None => OPERATION_TIMEOUT,
    };

    let proposal = RegisterErc20Proposal {
        title: erc20_params.proposal_title,
        description: erc20_params.proposal_desc,
        erc20address: erc20_params.erc20_address,
    };
    let res = contact
        .submit_register_erc20_proposal(
            proposal,
            get_deposit(),
            get_fee(None),
            keys[0].validator_key,
            Some(duration),
        )
        .await
        .unwrap();
    info!("Gov proposal executed with {:?}", res);

    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
}

// Simple arguments to create a proposal with
pub struct RegisterCoinProposalParams {
    pub coin_metadata: Metadata,

    pub proposal_title: String,
    pub proposal_desc: String,
}

// Creates and submits a RegisterCoinProposal to the chain, then votes yes with all validators
pub async fn execute_register_coin_proposal(
    contact: &Contact,
    keys: &[ValidatorKeys],
    timeout: Option<Duration>,
    coin_params: RegisterCoinProposalParams,
) {
    let duration = match timeout {
        Some(dur) => dur,
        None => OPERATION_TIMEOUT,
    };

    let proposal = RegisterCoinProposal {
        title: coin_params.proposal_title,
        description: coin_params.proposal_desc,
        metadata: Some(coin_params.coin_metadata),
    };
    let res = contact
        .submit_register_coin_proposal(
            proposal,
            get_deposit(),
            get_fee(None),
            keys[0].validator_key,
            Some(duration),
        )
        .await
        .unwrap();
    info!("Gov proposal executed with {:?}", res);

    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
}

// Simple arguments to create a proposal with
pub struct UpgradeProposalParams {
    pub upgrade_height: i64,
    pub plan_name: String,
    pub plan_info: String,
    pub proposal_title: String,
    pub proposal_desc: String,
}

// Creates and submits a SoftwareUpgradeProposal to the chain, then votes yes with all validators
pub async fn execute_upgrade_proposal(
    contact: &Contact,
    keys: &[ValidatorKeys],
    timeout: Option<Duration>,
    upgrade_params: UpgradeProposalParams,
) {
    let duration = match timeout {
        Some(dur) => dur,
        None => OPERATION_TIMEOUT,
    };

    #[allow(deprecated)]
    let plan = Plan {
        name: upgrade_params.plan_name,
        time: None,
        height: upgrade_params.upgrade_height,
        info: upgrade_params.plan_info,
        upgraded_client_state: None,
    };
    let proposal = SoftwareUpgradeProposal {
        title: upgrade_params.proposal_title,
        description: upgrade_params.proposal_desc,
        plan: Some(plan),
    };
    let res = contact
        .submit_upgrade_proposal(
            proposal,
            get_deposit(),
            get_fee(None),
            keys[0].validator_key,
            Some(duration),
        )
        .await
        .unwrap();
    info!("Gov proposal executed with {:?}", res);

    vote_yes_on_proposals(contact, keys, None).await;
    wait_for_proposals_to_execute(contact).await;
}

// votes yes on every proposal available
pub async fn vote_yes_on_proposals(
    contact: &Contact,
    keys: &[ValidatorKeys],
    timeout: Option<Duration>,
) {
    let duration = match timeout {
        Some(dur) => dur,
        None => OPERATION_TIMEOUT,
    };
    // Vote yes on all proposals with all validators
    let proposals = contact
        .get_governance_proposals_in_voting_period()
        .await
        .unwrap();
    trace!("Found proposals: {:?}", proposals.proposals);
    let mut futs = Vec::new();
    for proposal in proposals.proposals {
        for key in keys.iter() {
            let res =
                vote_yes_with_retry(contact, proposal.proposal_id, key.validator_key, duration);
            futs.push(res);
        }
    }
    // vote on the proposal in parallel, reducing the number of blocks we wait for all
    // the tx's to get in.
    join_all(futs).await;
}

/// this utility function repeatedly attempts to vote yes on a governance
/// proposal up to MAX_VOTES times before failing
pub async fn vote_yes_with_retry(
    contact: &Contact,
    proposal_id: u64,
    key: impl PrivateKey,
    timeout: Duration,
) {
    const MAX_VOTES: u64 = 5;
    let mut counter = 0;
    let mut res = contact
        .vote_on_gov_proposal(
            proposal_id,
            VoteOption::Yes,
            get_fee(None),
            key.clone(),
            Some(timeout),
        )
        .await;
    while let Err(e) = res {
        contact.wait_for_next_block(TOTAL_TIMEOUT).await.unwrap();
        res = contact
            .vote_on_gov_proposal(
                proposal_id,
                VoteOption::Yes,
                get_fee(None),
                key.clone(),
                Some(timeout),
            )
            .await;
        counter += 1;
        if counter > MAX_VOTES {
            error!(
                "Vote for proposal has failed more than {} times, error {:?}",
                MAX_VOTES, e
            );
            panic!("failed to vote{}", e);
        }
    }
    let res = res.unwrap();
    info!(
        "Voting yes on governance proposal costing {} gas",
        res.gas_used
    );
}

// Checks that cosmos_account has each balance specified in expected_cosmos_coins.
// Note: ignores balances not in expected_cosmos_coins
pub async fn check_cosmos_balances(
    contact: &Contact,
    cosmos_account: CosmosAddress,
    expected_cosmos_coins: &[Coin],
) {
    let mut num_found = 0;

    let start = Instant::now();

    while Instant::now() - start < TOTAL_TIMEOUT {
        let mut good = true;
        let curr_balances = contact.get_balances(cosmos_account).await.unwrap();
        // These loops use loop labels, see the documentation on loop labels here for more information
        // https://doc.rust-lang.org/reference/expressions/loop-expr.html#loop-labels
        'outer: for bal in curr_balances.iter() {
            if num_found == expected_cosmos_coins.len() {
                break 'outer; // done searching entirely
            }
            'inner: for j in 0..expected_cosmos_coins.len() {
                if num_found == expected_cosmos_coins.len() {
                    break 'outer; // done searching entirely
                }
                if expected_cosmos_coins[j].denom != bal.denom {
                    continue;
                }
                let check = expected_cosmos_coins[j].amount == bal.amount;
                good = check;
                if !check {
                    warn!(
                        "found balance {}! expected {} trying again",
                        bal, expected_cosmos_coins[j].amount
                    );
                }
                num_found += 1;
                break 'inner; // done searching for this particular balance
            }
        }

        let check = num_found == curr_balances.len();
        // if it's already false don't set to true
        good = check || good;
        if !check {
            warn!(
                "did not find the correct balance for each expected coin! found {} of {}, trying again",
                num_found,
                curr_balances.len()
            );
        }
        if good {
            return;
        } else {
            sleep(Duration::from_secs(1)).await;
        }
    }
    panic!("Failed to find correct balances in check_cosmos_balances")
}

/// waits for the cosmos chain to start producing blocks, used to prevent race conditions
/// where our tests try to start running before the Cosmos chain is ready
pub async fn wait_for_cosmos_online(contact: &Contact, timeout: Duration) {
    // First check if we're past the first block, we can just start
    let latest = contact.get_latest_block().await;
    if latest.is_ok() {
        if let LatestBlock::Latest { block } = latest.unwrap() {
            if block.header.unwrap().height > 1 {
                return;
            }
        }
    };

    // The chain is not started, wait for some blocks to be produced
    let start = Instant::now();
    while let Err(CosmosGrpcError::NodeNotSynced) | Err(CosmosGrpcError::ChainNotRunning) =
        contact.wait_for_next_block(timeout).await
    {
        sleep(Duration::from_secs(1)).await;
        if Instant::now() - start > timeout {
            panic!("Cosmos node has not come online during timeout!")
        }
    }
    contact.wait_for_next_block(timeout).await.unwrap();
}

/// This function returns the valoper address of a validator
/// to whom delegating the returned amount of staking token will
/// create a 5% or greater change in voting power, triggering the
/// creation of a validator set update.
pub async fn get_validator_to_delegate_to(contact: &Contact) -> (CosmosAddress, Coin) {
    let validators = contact.get_active_validators().await.unwrap();
    let mut total_bonded_stake: Uint256 = 0u8.into();
    let mut has_the_least = None;
    let mut lowest = 0u8.into();
    for v in validators {
        let amount: Uint256 = v.tokens.parse().unwrap();
        total_bonded_stake += amount;

        if lowest == 0u8.into() || amount < lowest {
            lowest = amount;
            has_the_least = Some(v.operator_address.parse().unwrap());
        }
    }

    // since this is five percent of the total bonded stake
    // delegating this to the validator who has the least should
    // do the trick
    let five_percent = total_bonded_stake / 20u8.into();
    let five_percent = Coin {
        denom: STAKING_TOKEN.clone(),
        amount: five_percent,
    };

    (has_the_least.unwrap(), five_percent)
}

/// Waits for a particular block to be created
/// Returns an error if the chain fails to progress in a timely manner or the chain is not running
/// Panics if the block has already been surpassed
pub async fn wait_for_block(contact: &Contact, height: u64) -> Result<(), CosmosGrpcError> {
    let status = contact.get_chain_status().await?;
    let mut curr_height = match status {
        // Check the current height
        ChainStatus::Syncing => return Err(CosmosGrpcError::NodeNotSynced),
        ChainStatus::WaitingToStart => return Err(CosmosGrpcError::ChainNotRunning),
        ChainStatus::Moving { block_height } => {
            if block_height > height {
                panic!(
                    "Block height {} surpassed, current height is {}",
                    height, block_height
                );
            }
            block_height
        }
    };
    while curr_height < height {
        // Wait for the desired height
        contact.wait_for_next_block(OPERATION_TIMEOUT).await?; // Err if any block takes 30s+
        let new_status = contact.get_chain_status().await?;
        if let ChainStatus::Moving { block_height } = new_status {
            curr_height = block_height
        } else {
            // wait_for_next_block checks every second, so it's not likely the chain could halt for
            // an upgrade before we find the desired height
            return Err(CosmosGrpcError::BadResponse(
                "Wait for block: Chain was running and now it's not?".to_string(),
            ));
        }
    }
    Ok(())
}

/// Delegates `delegate_amount` to `delegate_to` and queries for confirmation of that delegation
/// Returns an error if the delegation or the query fail, returns the result of the delegation query
pub async fn delegate_and_confirm(
    contact: &Contact,
    user_key: impl PrivateKey,
    user_address: CosmosAddress,
    delegate_to: CosmosAddress,
    delegate_amount: Coin,
    fee_coin: Coin,
) -> Result<Option<DelegationResponse>, CosmosGrpcError> {
    let deleg_result = contact
        .delegate_to_validator(
            delegate_to,
            delegate_amount.clone(),
            fee_coin,
            user_key,
            Some(TOTAL_TIMEOUT),
        )
        .await;
    if deleg_result.is_err() {
        let err_str = format!(
            "Failed to delegate {} to validator {}, error {:?}",
            delegate_amount,
            delegate_to,
            deleg_result.unwrap_err()
        );
        error!("{}", err_str);
        return Err(CosmosGrpcError::BadResponse(err_str));
    }
    let deleg_confirm = contact.get_delegation(delegate_to, user_address).await;
    if deleg_confirm.is_err() {
        let err_str = format!(
            "Failed to query for delegation of {} to validator {}, error {:?}",
            delegate_amount,
            delegate_to,
            deleg_confirm.unwrap_err()
        );
        error!("{}", err_str);
        return Err(CosmosGrpcError::BadResponse(err_str));
    }
    Ok(deleg_confirm.unwrap())
}

/// Sends the given `amount` to each of `receivers` coming from `sender`
pub async fn send_funds_bulk(
    contact: &Contact,
    sender: impl PrivateKey,
    receivers: &[CosmosAddress],
    amount: Coin,
    timeout: Option<Duration>,
) -> Result<(), CosmosGrpcError> {
    let fee = Some(Coin {
        denom: STAKING_TOKEN.clone(),
        amount: 10u8.into(),
    });
    for dest in receivers {
        contact
            .send_coins(amount.clone(), fee.clone(), *dest, timeout, sender.clone())
            .await?;
    }

    Ok(())
}

/// Waits up to TOTAL_TIMEOUT or provided timeout for the `user_address` account to gain at least `balance`
pub async fn wait_for_balance(
    contact: &Contact,
    user_address: CosmosAddress,
    balance: Coin,
    timeout: Option<Duration>,
) {
    let duration = timeout.unwrap_or(TOTAL_TIMEOUT);
    let start = Instant::now();
    while Instant::now() - start < duration {
        let actual_balance = contact
            .get_balance(user_address, balance.denom.clone())
            .await;
        if let Ok(Some(bal)) = actual_balance {
            if bal.denom == balance.denom && bal.amount >= balance.amount {
                return;
            }
        }

        contact.wait_for_next_block(duration).await.unwrap();
    }

    panic!("User did not attain >= expected balance");
}

/// waits for the governance proposal to execute by waiting for it to leave
/// the 'voting' status
pub async fn wait_for_proposals_to_execute(contact: &Contact) {
    let start = Instant::now();
    loop {
        let proposals = contact
            .get_governance_proposals_in_voting_period()
            .await
            .unwrap();
        if Instant::now() - start > TOTAL_TIMEOUT {
            panic!("Gov proposal did not execute")
        } else if proposals.proposals.is_empty() {
            return;
        }
        sleep(Duration::from_secs(5)).await;
    }
}

/// Helper function for encoding the the proto any type
pub fn encode_any(input: impl prost::Message, type_url: impl Into<String>) -> Any {
    let mut value = Vec::new();
    input.encode(&mut value).unwrap();
    Any {
        type_url: type_url.into(),
        value,
    }
}

pub fn decode_any<T: Message + Default>(any: Any) -> Result<T, DecodeError> {
    let bytes = any.value;

    decode_bytes(bytes)
}

pub fn decode_bytes<T: Message + Default>(bytes: Vec<u8>) -> Result<T, DecodeError> {
    let mut buf = BytesMut::with_capacity(bytes.len());
    buf.extend_from_slice(&bytes);

    // Here we use the `T` type to decode whatever type of message this attestation holds
    // for use in the `f` function
    T::decode(buf)
}

/// TODO: Web30?
///
/// This function efficiently distributes ERC20 tokens to a large number of provided Ethereum addresses
/// the real problem here is that you can't do more than one send operation at a time from a
/// single address without your sequence getting out of whack. By manually setting the nonce
/// here we can send thousands of transactions in only a few blocks
pub async fn send_erc20_bulk(
    amount: Uint256,
    erc20: EthAddress,
    destinations: &[EthAddress],
    web3: &Web3,
) {
    check_erc20_balance(erc20, amount, *MINER_ETH_ADDRESS, web3).await;
    let mut nonce = web3
        .eth_get_transaction_count(*MINER_ETH_ADDRESS)
        .await
        .unwrap();
    let mut transactions = Vec::new();
    for address in destinations {
        let send = web3.erc20_send(
            amount,
            *address,
            erc20,
            *MINER_PRIVATE_KEY,
            Some(OPERATION_TIMEOUT),
            vec![SendTxOption::Nonce(nonce.clone())],
        );
        transactions.push(send);
        nonce += 1u64.into();
    }
    let txids = join_all(transactions).await;
    wait_for_txids(txids, web3).await;
    let mut balance_checks = Vec::new();
    for address in destinations {
        let check = check_erc20_balance(erc20, amount, *address, web3);
        balance_checks.push(check);
    }
    join_all(balance_checks).await;
}

/// TODO: Web30?
///
/// utility function for bulk checking erc20 balances, used to provide
/// a single future that contains the assert as well s the request
pub async fn check_erc20_balance(
    erc20: EthAddress,
    amount: Uint256,
    address: EthAddress,
    web3: &Web3,
) {
    let new_balance = get_erc20_balance_safe(erc20, web3, address).await;
    let new_balance = new_balance.unwrap();
    assert!(new_balance >= amount);
}

/// TODO: Web30?
///
/// utility function for bulk checking erc20 balances, used to provide
/// a single future that contains the assert as well s the request
pub async fn get_erc20_balance_safe(
    erc20: EthAddress,
    web3: &Web3,
    address: EthAddress,
) -> Result<Uint256, Web3Error> {
    let start = Instant::now();
    // overly complicated retry logic allows us to handle the possibility that gas prices change between blocks
    // and cause any individual request to fail.
    let mut new_balance = Err(Web3Error::BadInput("Intentional Error".to_string()));
    while new_balance.is_err() && Instant::now() - start < TOTAL_TIMEOUT {
        new_balance = web3.get_erc20_balance(erc20, address).await;
        // only keep trying if our error is gas related
        if let Err(ref e) = new_balance {
            if !e.to_string().contains("maxFeePerGas") {
                break;
            }
        }
    }
    Ok(new_balance.unwrap())
}

/// utility function that waits for a large number of txids to enter a block
async fn wait_for_txids(txids: Vec<Result<Uint256, Web3Error>>, web3: &Web3) {
    let mut wait_for_txid = Vec::new();
    for txid in txids {
        let wait = web3.wait_for_transaction(txid.unwrap(), TOTAL_TIMEOUT, None);
        wait_for_txid.push(wait);
    }
    join_all(wait_for_txid).await;
}
