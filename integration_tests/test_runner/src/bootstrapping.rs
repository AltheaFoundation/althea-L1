use crate::ibc_utils::get_channel;
use crate::utils::{
    get_chain_id, get_deposit, get_ibc_chain_id, parse_contracts_root, parse_dex_contracts_root,
    ALTHEA_RELAYER_ADDRESS, COSMOS_NODE_GRPC, HERMES_CONFIG, IBC_RELAYER_ADDRESS,
    IBC_STAKING_TOKEN, OPERATION_TIMEOUT, RELAYER_MNEMONIC_FILE,
};
use crate::utils::{
    send_erc20_bulk, EthermintUserKey, ValidatorKeys, ETH_NODE, MINER_PRIVATE_KEY, TOTAL_TIMEOUT,
};

use althea_proto::cosmos_sdk_proto::ibc::core::channel::v1::query_client::QueryClient as IbcChannelQueryClient;
use clarity::Address as EthAddress;
use clarity::Uint256;
use deep_space::private_key::CosmosPrivateKey;
use deep_space::private_key::PrivateKey;
use deep_space::Contact;
use std::fs::File;
use std::io::{BufRead, BufReader, Read, Write};
use std::os::fd::{FromRawFd, IntoRawFd};
use std::path::Path;
use std::process::{Command, ExitStatus, Stdio};
use std::thread;
use std::time::Duration;
use web30::client::Web3;
use web30::jsonrpc::error::Web3Error;

/// Parses the output of the cosmoscli keys add command to import the private key
fn parse_phrases(filename: &str) -> (Vec<CosmosPrivateKey>, Vec<String>) {
    let file = File::open(filename).expect("Failed to find phrases");
    let reader = BufReader::new(file);
    let mut ret_keys = Vec::new();
    let mut ret_phrases = Vec::new();

    for line in reader.lines() {
        let phrase = line.expect("Error reading phrase file!");
        if phrase.is_empty()
            || phrase.contains("write this mnemonic phrase")
            || phrase.contains("recover your account if")
        {
            continue;
        }
        let key = CosmosPrivateKey::from_phrase(&phrase, "").expect("Bad phrase!");
        ret_keys.push(key);
        ret_phrases.push(phrase);
    }
    (ret_keys, ret_phrases)
}

/// Validator private keys are generated via the althea keys add
/// command, from there they are used to create gentx's and start the
/// chain, these keys change every time the container is restarted.
/// The mnemonic phrases are dumped into a text file /validator-phrases
/// the phrases are in increasing order, so validator 1 is the first key
/// and so on. While validators may later fail to start it is guaranteed
/// that we have one key for each validator in this file.
pub fn parse_validator_keys() -> (Vec<CosmosPrivateKey>, Vec<String>) {
    let filename = "/validator-phrases";
    info!("Reading mnemonics from {}", filename);
    parse_phrases(filename)
}

/// The same as parse_validator_keys() except for a second chain accessed
/// over IBC for testing purposes
// pub fn parse_ibc_validator_keys() -> (Vec<CosmosPrivateKey>, Vec<String>) {
//     let filename = "/ibc-validator-phrases";
//     info!("Reading mnemonics from {}", filename);
//     parse_phrases(filename)
// }

pub fn get_keys() -> Vec<ValidatorKeys> {
    let (cosmos_keys, cosmos_phrases) = parse_validator_keys();
    let mut ret = Vec::new();
    for (c_key, c_phrase) in cosmos_keys.into_iter().zip(cosmos_phrases) {
        ret.push(ValidatorKeys {
            validator_key: c_key,
            validator_phrase: c_phrase,
        })
    }
    ret
}

/// The same as parse_validator_keys() except for a second chain accessed
/// over IBC for testing purposes
pub fn parse_ibc_validator_keys() -> (Vec<CosmosPrivateKey>, Vec<String>) {
    let filename = "/ibc-validator-phrases";
    info!("Reading mnemonics from {}", filename);
    parse_phrases(filename)
}

/// This function deploys the required contracts onto the Ethereum testnet
/// this runs only when the DEPLOY_CONTRACTS env var is set right after
/// the Ethereum test chain starts in the testing environment. We write
/// the stdout of this to a file for later test runs to parse
pub async fn deploy_erc20_contracts(contact: &Contact) {
    // prevents the node deployer from failing (rarely) when the chain has not
    // yet produced the next block after submitting each eth address
    contact.wait_for_next_block(TOTAL_TIMEOUT).await.unwrap();

    // the default unmoved locations for the Gravity repo
    const A: [&str; 2] = ["/althea/solidity/contract-deployer.ts", "/althea/solidity/"];
    // the default unmoved locations for Github Actions
    const B: [&str; 2] = [
        "/home/runner/work/althea-L1/althea-L1/solidity/contract-deployer.ts",
        "/home/runner/work/althea-L1/althea-L1/solidity/",
    ];

    // the user specified contracts root
    let contracts_root = parse_contracts_root();
    let paths = if let Ok(root) = contracts_root {
        Some(vec![
            format!("{}/contract-deployer.ts", root),
            format!("{}/", root),
        ])
    } else {
        return_existing(vec![A, B]).map(|path| vec![path[0].to_string(), path[1].to_string()])
    };

    let output = match paths {
        Some(path) => {
            info!("Deploying contracts from {:?}", path);
            Command::new("npx")
                .args([
                    "ts-node",
                    &path[0],
                    &format!("--eth-node={}", ETH_NODE.as_str()),
                    &format!("--eth-privkey={:#x}", *MINER_PRIVATE_KEY),
                    &format!("--artifacts-root={}", path[1]),
                ])
                .current_dir(&path[1])
                .output()
                .expect("Failed to deploy contracts!")
        }
        None => {
            panic!("Could not find json contract artifacts in any known location!")
        }
    };
    info!("stdout: {}", String::from_utf8_lossy(&output.stdout));
    info!("stderr: {}", String::from_utf8_lossy(&output.stderr));
    if !ExitStatus::success(&output.status) {
        panic!("Contract deploy failed!")
    }
    let mut file = File::create(ERC20_CONTRACTS_FILE).unwrap();
    file.write_all(&output.stdout).unwrap();
}

/// The file where the contract addresses are stored
const DEX_CONTRACTS_FILE: &str = "/tmp/dex-contracts";

/// This function deploys the Ambient (aka CrocSwap) dex contracts and configures them
pub async fn deploy_dex() {
    // the default unmoved locations for the Gravity repo in the docker container
    const A: [&str; 2] = [
        "/althea/solidity-dex/misc/scripts/dex-deployer.ts",
        "/althea/solidity-dex/artifacts/contracts/",
    ];
    // the default unmoved locations for Github Actions
    const B: [&str; 2] = [
        "/home/runner/work/althea-L1/althea-L1/solidity-dex/misc/scripts/dex-deployer.ts",
        "/home/runner/work/althea-L1/althea-L1/solidity-dex/artifacts/contracts/",
    ];
    // the user specified contracts root
    let contracts_root = parse_dex_contracts_root();
    let paths = if let Ok(root) = contracts_root {
        Some(vec![
            format!("{}/misc/scripts/dex-deployer.ts", root),
            format!("{}/artifacts/contracts/", root),
        ])
    } else {
        return_existing(vec![A, B]).map(|path| vec![path[0].to_string(), path[1].to_string()])
    };

    let output = match paths {
        Some(path) => {
            info!("Deploying contracts from {:?}", path);
            Command::new("npx")
                .args([
                    "ts-node",
                    &path[0],
                    &format!("--eth-node={}", ETH_NODE.as_str()),
                    &format!("--eth-privkey={:#x}", *MINER_PRIVATE_KEY),
                    &format!("--artifacts-root={}", path[1]),
                ])
                .current_dir(&path[1])
                .output()
                .expect("Failed to deploy contracts!")
        }
        None => {
            panic!("Could not find json contract artifacts in any known location!")
        }
    };
    info!("stdout: {}", String::from_utf8_lossy(&output.stdout));
    info!("stderr: {}", String::from_utf8_lossy(&output.stderr));
    if !ExitStatus::success(&output.status) {
        panic!("Contract deploy failed!")
    }
    let mut file = File::create(DEX_CONTRACTS_FILE).unwrap();
    file.write_all(&output.stdout).unwrap();
}

/// This function deploys Multicall3, which is used by various frontends
pub async fn deploy_multicall() {
    // the default unmoved locations for the Gravity repo in the docker container
    const A: [&str; 2] = [
        "/althea/solidity-dex/misc/scripts/multicall-deployer.ts",
        "/althea/solidity-dex/artifacts/contracts/periphery/",
    ];
    // locations for Github Actions
    const B: [&str; 2] = [
        "/home/runner/work/althea-L1/althea-L1/solidity-dex/misc/scripts/multicall-deployer.ts",
        "/home/runner/work/althea-L1/althea-L1/solidity-dex/artifacts/contracts/periphery/",
    ];
    // the user specified contracts root
    let contracts_root = parse_dex_contracts_root();
    let paths = if let Ok(root) = contracts_root {
        Some(vec![
            format!("{}/misc/scripts/multicall-deployer.ts", root),
            format!("{}/artifacts/contracts/periphery/", root),
        ])
    } else {
        return_existing(vec![A, B]).map(|path| vec![path[0].to_string(), path[1].to_string()])
    };

    let output = match paths {
        Some(path) => {
            info!("Deploying contracts from {:?}", path);
            Command::new("npx")
                .args([
                    "ts-node",
                    &path[0],
                    &format!("--eth-node={}", ETH_NODE.as_str()),
                    &format!("--eth-privkey={:#x}", *MINER_PRIVATE_KEY),
                    &format!("--artifacts-root={}", path[1]),
                ])
                .current_dir(&path[1])
                .output()
                .expect("Failed to deploy contracts!")
        }
        None => {
            panic!("Could not find json contract artifacts in any known location!")
        }
    };
    info!("stdout: {}", String::from_utf8_lossy(&output.stdout));
    info!("stderr: {}", String::from_utf8_lossy(&output.stderr));
    if !ExitStatus::success(&output.status) {
        panic!("Contract deploy failed!")
    }
    let mut file = File::create("/tmp/multicall-contract").unwrap();
    file.write_all(&output.stdout).unwrap();
}

// TODO: Fix send_erc20_bulk to make this method not so slow
pub async fn send_erc20s_to_evm_users(
    web3: &Web3,
    erc20_contracts: Vec<EthAddress>,
    evm_users: Vec<EthermintUserKey>,
    amount: Uint256,
) -> Result<(), Web3Error> {
    let destinations: Vec<EthAddress> = evm_users.into_iter().map(|euk| euk.eth_address).collect();

    // The users have been funded, skip sending erc20s
    info!("Checking for existing balances, might skip funding");
    if !web3
        .get_erc20_balance(
            *erc20_contracts.first().unwrap(),
            *destinations.first().unwrap(),
        )
        .await
        .unwrap()
        .is_zero()
    {
        return Ok(());
    }

    info!("Actually funding EVM users with the ERC20s");
    for erc20 in erc20_contracts {
        send_erc20_bulk(amount, erc20, &destinations, web3).await;
    }
    Ok(())
}

fn all_paths_exist(input: &[&str]) -> bool {
    for i in input {
        if !Path::new(i).exists() {
            return false;
        }
    }
    true
}

fn return_existing(paths: Vec<[&str; 2]>) -> Option<[&str; 2]> {
    paths.into_iter().find(|&path| all_paths_exist(&path))
}

pub struct BootstrapContractAddresses {
    pub erc20_addresses: Vec<EthAddress>,
    pub erc721_addresses: Vec<EthAddress>,
    pub weth_address: EthAddress,
    pub uniswap_liquidity_address: Option<EthAddress>,
}

pub const ERC20_CONTRACTS_FILE: &str = "/tmp/contracts";

/// Parses the ERC20 and Gravity contract addresses from the file created
/// in deploy_contracts()
pub fn parse_contract_addresses() -> BootstrapContractAddresses {
    let mut file =
        File::open(ERC20_CONTRACTS_FILE).expect("Failed to find contracts! did they not deploy?");
    let mut output = String::new();
    file.read_to_string(&mut output).unwrap();
    let mut erc20_addresses = Vec::new();
    let mut erc721_addresses = Vec::new();
    let mut weth_address = EthAddress::default();
    let mut uniswap_liquidity = None;
    for line in output.lines() {
        if line.contains("ERC20 deployed at Address -") {
            let address_string = line.split('-').last().unwrap();
            erc20_addresses.push(address_string.trim().parse().unwrap());
            info!("found erc20 address it is {}", address_string);
        } else if line.contains("ERC721 deployed at Address -") {
            let address_string = line.split('-').last().unwrap();
            erc721_addresses.push(address_string.trim().parse().unwrap());
            info!("found erc721 address it is {}", address_string);
        } else if line.contains("WETH deployed at Address -") {
            let address_string = line.split('-').last().unwrap();
            weth_address = address_string.trim().parse().unwrap();
        } else if line.contains("Uniswap Liquidity test deployed at Address - ") {
            let address_string = line.split('-').last().unwrap();
            uniswap_liquidity = Some(address_string.trim().parse().unwrap());
        }
    }
    BootstrapContractAddresses {
        erc20_addresses,
        erc721_addresses,
        weth_address,
        uniswap_liquidity_address: uniswap_liquidity,
    }
}

#[derive(Debug, Clone)]
pub struct DexAddresses {
    pub dex: EthAddress,
    pub query: EthAddress,
    pub impact: EthAddress,
    pub policy: EthAddress,
    pub upgrade: EthAddress,
}

/// Parses the DEX contract addresses from the file created
/// in deploy_dex()
pub fn parse_dex_contract_addresses() -> DexAddresses {
    let mut file =
        File::open(DEX_CONTRACTS_FILE).expect("Failed to find dex contracts! did they not deploy?");
    let mut output = String::new();
    file.read_to_string(&mut output).unwrap();
    let mut dex: EthAddress = EthAddress::default();
    let mut query: EthAddress = EthAddress::default();
    let mut impact: EthAddress = EthAddress::default();
    let mut policy: EthAddress = EthAddress::default();
    let mut upgrade: EthAddress = EthAddress::default();
    for line in output.lines() {
        if line.contains("CrocSwapDex deployed at Address -") {
            let address_string = line.split('-').last().unwrap();
            dex = address_string.trim().parse().unwrap();
            info!("found dex address it is {}", address_string);
        } else if line.contains("CrocQuery deployed at Address -") {
            let address_string = line.split('-').last().unwrap();
            query = address_string.trim().parse().unwrap();
            info!("found query address it is {}", address_string);
        } else if line.contains("CrocImpact deployed at Address -") {
            let address_string = line.split('-').last().unwrap();
            impact = address_string.trim().parse().unwrap();
            info!("found impact address it is {}", address_string);
        } else if line.contains("CrocPolicy deployed at Address -") {
            let address_string = line.split('-').last().unwrap();
            policy = address_string.trim().parse().unwrap();
            info!("found policy address it is {}", address_string);
        } else if line.contains("ColdPathUpgrade deployed at Address -") {
            let address_string = line.split('-').last().unwrap();
            upgrade = address_string.trim().parse().unwrap();
            info!("found upgrade address it is {}", address_string);
        }
    }
    DexAddresses {
        dex,
        query,
        impact,
        policy,
        upgrade,
    }
}

// Creates a key in the relayer's test keyring, which the relayer should use
// Hermes stores its keys in hermes_home/
pub fn setup_relayer_keys() -> Result<(), Box<dyn std::error::Error>> {
    let mut command = hermes_base();
    let mut mnemonic_path: Option<&str> = None;
    for path in RELAYER_MNEMONIC_FILE {
        if Path::new(path).exists() {
            mnemonic_path = Some(path);
            break;
        }
    }
    let mnemonic_path = mnemonic_path.expect("Could not find relayer mnemonic file");
    let _althea_key = command
        .args([
            "keys",
            "add",
            "--key-name",
            "altheakey",
            "--chain",
            &get_chain_id(),
            "--hd-path",
            "m/44'/60'/0'/0/0",
            "--mnemonic-file",
            mnemonic_path,
        ])
        .spawn()
        .expect("Failed to add althea key");
    info!("Added altheakey to hermes keybase");

    let mut command = hermes_base();
    let _ibc_key = command
        .args([
            "keys",
            "add",
            "--key-name",
            "ibckey",
            "--chain",
            &get_ibc_chain_id(),
            "--mnemonic-file",
            mnemonic_path,
        ])
        .spawn()
        .expect("Failed to add ibc key");
    info!("Added ibckey to hermes keybase");

    Ok(())
}

const IBC_RELAYER_LOGS_ROOT: &str = "/tmp/ibc-relayer-logs/";

// Create a channel between gravity chain and the ibc test chain over the "transfer" port
// Writes the output to /ibc-relayer-logs/channel-creation
pub fn create_ibc_channel(hermes_base: Command) {
    info!("Creating IBC channel with hermes");
    let mut hermes_base = hermes_base;
    // hermes -c config.toml create channel gravity-test-1 ibc-test-1 --port-a transfer --port-b transfer
    let create_channel = hermes_base.args([
        "create",
        "channel",
        "--a-chain",
        &get_chain_id(),
        "--b-chain",
        &get_ibc_chain_id(),
        "--a-port",
        "transfer",
        "--b-port",
        "transfer",
        "--new-client-connection",
        "--yes",
    ]);

    std::fs::create_dir_all(IBC_RELAYER_LOGS_ROOT).unwrap();
    let out_file = File::options()
        .write(true)
        .create(true)
        .open(format!("{}channel-creation", IBC_RELAYER_LOGS_ROOT))
        .unwrap()
        .into_raw_fd();
    unsafe {
        // unsafe needed for stdout + stderr redirect to file
        let create_channel = create_channel
            .stdout(Stdio::from_raw_fd(out_file))
            .stderr(Stdio::from_raw_fd(out_file));
        create_channel.spawn().expect("Could not create channel");
    }
}

// Start an IBC relayer locally and run until it terminates
// full_scan Force a full scan of the chains for clients, connections and channels
// Writes the output to /ibc-relayer-logs/hermes-logs
pub fn run_ibc_relayer(hermes_base: Command, full_scan: bool) {
    info!("Running ibc relayer");
    let mut hermes_base = hermes_base;
    let mut start = hermes_base.arg("start");
    if full_scan {
        start = start.arg("--full-scan");
    }
    std::fs::create_dir_all(IBC_RELAYER_LOGS_ROOT).unwrap();
    let out_file = File::options()
        .write(true)
        .create(true)
        .open(format!("{}hermes-logs", IBC_RELAYER_LOGS_ROOT))
        .unwrap()
        .into_raw_fd();
    unsafe {
        // unsafe needed for stdout + stderr redirect to file
        start
            .stdout(Stdio::from_raw_fd(out_file))
            .stderr(Stdio::from_raw_fd(out_file))
            .spawn()
            .expect("Could not run hermes");
    }
}

// starts up the IBC relayer (hermes) in a background thread
pub async fn start_ibc_relayer(
    contact: &Contact,
    ibc_contact: &Contact,
    keys: &[ValidatorKeys],
    ibc_keys: &[CosmosPrivateKey],
) {
    let althea_deposit = get_deposit(None);
    let ibc_deposit = get_deposit(Some(IBC_STAKING_TOKEN.to_string()));
    info!("Sending relayer {althea_deposit:?} on althea");
    contact
        .send_coins(
            althea_deposit,
            None,
            ALTHEA_RELAYER_ADDRESS.parse().unwrap(),
            Some(OPERATION_TIMEOUT),
            keys[0].validator_key,
        )
        .await
        .unwrap();
    info!("Sending relayer {ibc_deposit:?} on ibc-test");
    ibc_contact
        .send_coins(
            ibc_deposit,
            Some(deep_space::Coin {
                amount: 100u8.into(),
                denom: IBC_STAKING_TOKEN.to_string(),
            }),
            IBC_RELAYER_ADDRESS.parse().unwrap(),
            Some(OPERATION_TIMEOUT),
            ibc_keys[0],
        )
        .await
        .unwrap();
    info!("test-runner starting IBC relayer mode: init hermes, create ibc channel, start hermes");
    setup_relayer_keys().unwrap();
    info!("Relayer keys set up");

    let althea_channel_qc = IbcChannelQueryClient::connect(COSMOS_NODE_GRPC.as_str())
        .await
        .expect("Could not connect channel query client");

    // Wait for the ibc channel to be created and find the channel ids
    let channel_id_timeout = Duration::from_secs(60 * 5);
    info!("Getting channel...");
    let althea_channel = get_channel(
        althea_channel_qc,
        get_ibc_chain_id(),
        Some(channel_id_timeout),
    )
    .await;
    if althea_channel.is_err() {
        info!("No IBC channels exist between althea_6633438-1 and ibc-test-1, creating one now...");
        create_ibc_channel(hermes_base());
    }
    thread::spawn(|| {
        run_ibc_relayer(hermes_base(), true); // likely will not return from here, just keep running
    });
    info!(
        "Running ibc relayer in the background, directing output to {}",
        IBC_RELAYER_LOGS_ROOT
    );
}

fn hermes_base() -> Command {
    let mut hermes_base = Command::new("hermes");
    let mut config: Option<&str> = None;
    for path in HERMES_CONFIG {
        if Path::new(path).exists() {
            config = Some(path);
            break;
        }
    }
    let config = config.expect("Could not find hermes config file");
    hermes_base.arg("--config").arg(config);
    hermes_base
}
