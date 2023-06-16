use std::time::Duration;

use clarity::{
    utils::bytes_to_hex_str, Address as EthAddress, PrivateKey as EthPrivateKey, Uint256,
};

use num::ToPrimitive;
use web30::{
    client::Web3,
    jsonrpc::error::Web3Error,
    types::{TransactionRequest, TransactionResponse},
};

use crate::utils::OPERATION_TIMEOUT;

lazy_static! {
    pub static ref TOKENIZED_ACCOUNT_TOKEN_ID: Uint256 = 1u8.into();
}

// ==========================================================================================================================================
//                                                  TOKENIZED ACCOUNT CONVENIENCE FUNCTIONS
// ==========================================================================================================================================

/// Queries the given `tokenized_account_contract` for the owner of the NFT (token id = 1), optionally querying as a given `querier` address
/// Note: Provide a querier if the tokenized account nft contract does not have any native token, since simulated calls require gas
pub async fn get_tokenized_account_owner(
    web30: &Web3,
    tokenized_account_contract: EthAddress,
    querier: Option<EthAddress>,
) -> Result<EthAddress, Web3Error> {
    get_erc721_owner(
        web30,
        tokenized_account_contract,
        *TOKENIZED_ACCOUNT_TOKEN_ID,
        querier,
    )
    .await
}

/// Approves `to_approve` to transfer ownership of the tokenized account at `tokenized_account_contract` controlled by `approver`
/// Provide a custom timeout or None to wait ~30 seconds for the block
pub async fn approve_tokenized_account(
    web30: &Web3,
    tokenized_account_contract: EthAddress,
    approver: EthPrivateKey,
    to_approve: EthAddress,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    approve_erc721(
        web30,
        tokenized_account_contract,
        *TOKENIZED_ACCOUNT_TOKEN_ID,
        approver,
        to_approve,
        timeout,
    )
    .await
}

/// Transfers ownership of the tokenized account at `tokenized_account_contract` owned by `from` to `to`
/// Provide a custom timeout or None to wait ~30 seconds for the block
pub async fn send_tokenized_account(
    web30: &Web3,
    tokenized_account_contract: EthAddress,
    from: EthPrivateKey,
    to: EthAddress,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    send_erc721(
        web30,
        tokenized_account_contract,
        *TOKENIZED_ACCOUNT_TOKEN_ID,
        from,
        to,
        timeout,
    )
    .await
}

#[derive(Clone, Copy, Debug)]
// An individual threshold to control the operating balance of a single token of a tokenized account
pub struct TokenizedAccountThreshold {
    pub token: EthAddress,
    pub amount: Uint256,
}

// Allows collections to convert themselves into types understandable by Ethereum
trait EthRepr<T> {
    fn to_eth_repr(&self) -> T;
}

// Convert a collection of TokenizedAccountThresholds into an EVM-compatible type
impl EthRepr<(Vec<EthAddress>, Vec<Uint256>)> for Vec<TokenizedAccountThreshold> {
    // Converts the collection of thresholds into the address[] and uint256[] collections the Ethereum ABI expects
    fn to_eth_repr(&self) -> (Vec<EthAddress>, Vec<Uint256>) {
        let mut tokens = vec![];
        let mut amounts = vec![];

        for thresh in self {
            tokens.push(thresh.token);
            amounts.push(thresh.amount);
        }

        (tokens, amounts)
    }
}

/// Queries the given TokenizedAccountNFT's configured balance thresholds, which specify the maximum operating balance of a tokenized account
/// Calls the TokenizedAccountNFT's `getThresholds` function
pub async fn get_thresholds(
    web30: &Web3,
    tokenized_account_contract: EthAddress,
    querier: Option<EthAddress>,
) -> Result<Vec<TokenizedAccountThreshold>, Web3Error> {
    // ABI: getThresholds() public virtual view returns (address[] memory, uint256[] memory)
    let caller = querier.unwrap_or(tokenized_account_contract);

    let payload = clarity::abi::encode_call("getThresholds()", &[])?;
    let thresholds_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, tokenized_account_contract, payload),
            None,
        )
        .await?;

    info!(
        "Got thresholds response: {}",
        clarity::utils::debug_print_data(&thresholds_res)
    );

    // We expect the following in the response, regardless of the contents of the thresholds arrays:
    // 1. the offset index of the dynamic-encoded address[], whose first bytes will have the length of the array
    // 2. the offset index of the dynamic-encoded uint256[], whose first bytes will have the length of the array
    // 3. the length of the dynamic-encoded address[]
    // 4. all elements of the address[], if any
    // 5. the length of the dynamic-encoded uint256[]
    // 6. all elements of the uint256[], if any
    let mut decoded_thresholds: Vec<TokenizedAccountThreshold> = vec![];

    let thresholds = thresholds_res.as_slice();
    let address_arr_head = &thresholds[0..32];
    let address_arr_offset = Uint256::from_be_bytes(address_arr_head)
        .to_usize()
        .expect("address[] offset bigger than usize!");
    debug!("address array info offset at {address_arr_offset}");
    assert!(address_arr_offset > 0u8.into()); // Even if there are no elements, we should have an index
    let uint256_arr_head = &thresholds[32..64];
    let uint256_arr_offset = Uint256::from_be_bytes(uint256_arr_head)
        .to_usize()
        .expect("uint256[] offset bigger than usize!");
    debug!("amounts array info offset at {uint256_arr_offset}");
    assert!(uint256_arr_offset > 0u8.into()); // Even if there are no elements, we should have an index

    let address_arr_info =
        Uint256::from_be_bytes(&thresholds[address_arr_offset..address_arr_offset + 32]);
    let uint256_arr_info =
        Uint256::from_be_bytes(&thresholds[uint256_arr_offset..uint256_arr_offset + 32]);
    debug!(
        "Read address arr info from {address_arr_offset} to {}: {}",
        address_arr_offset + 32,
        address_arr_info
    );
    debug!(
        "Read amounts arr info from {uint256_arr_offset} to {}: {}",
        uint256_arr_offset + 32,
        uint256_arr_info
    );
    assert_eq!(address_arr_info, uint256_arr_info);
    let num_elts = address_arr_info
        .to_usize()
        .expect("array size threshold bigger than usize!");
    debug!("Expecting {num_elts} thresholds (tokens + amounts)");
    for i in 0..num_elts {
        let address_block_start = address_arr_offset + 32 + (i * 32); // Array is at Info + 32, elements are spaced every 32 bytes
        let address_start = address_block_start + 12; // Ignore the 0-padding on the left
        let address_end = address_block_start + 32; // Addresses are 20 bytes, but packed for JSON-RPC they are 32 bytes left padded with 0's
        let address_bytes = &thresholds[address_start..address_end];

        let uint256_start = uint256_arr_offset + 32 + (i * 32); // Array is at Info + 32, elements are spaced every 32 bytes
        let uint256_end = uint256_start + 32; // Uint256es are 32 bytes
        let uint256_bytes = &thresholds[uint256_start..uint256_end];

        debug!("Reading address from {address_start} to {address_end}: {}, and amount from {uint256_start} to {uint256_end}: {}", bytes_to_hex_str(address_bytes), bytes_to_hex_str(uint256_bytes));

        let address = EthAddress::from_slice(address_bytes).expect("Invalid address in thresholds");
        let uint256 = Uint256::from_be_bytes(uint256_bytes);
        decoded_thresholds.push(TokenizedAccountThreshold {
            token: address,
            amount: uint256,
        });
    }

    Ok(decoded_thresholds)
}

/// Specifies the given TokenizedAccountNFT's configured balance thresholds, which control the maximum operating balance of a tokenized account per token
/// Calls the TokenizedAccountNFT's `setThresholds` function
/// This function is protected and only callable by the owner of the NFT or someone approved to control it
/// Provide a custom timeout or None to wait ~30 seconds for the block
pub async fn set_thresholds(
    web30: &Web3,
    tokenized_account_contract: EthAddress,
    owner_or_approved: EthPrivateKey,
    thresholds: Vec<TokenizedAccountThreshold>,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let timeout = timeout.unwrap_or(OPERATION_TIMEOUT);
    let (tokens, amounts) = thresholds.to_eth_repr();
    // ABI: setThresholds(address[] calldata newErc20s, uint256[] calldata newAmounts) public virtual onlyOwnerOrApproved(AccountId)
    let payload = clarity::abi::encode_call(
        "setThresholds(address[],uint256[])",
        &[tokens.into(), amounts.into()],
    )?;

    let transfer_res = web30
        .send_transaction(
            tokenized_account_contract,
            payload,
            0u8.into(),
            owner_or_approved,
            vec![],
        )
        .await?;

    web30
        .wait_for_transaction(transfer_res, timeout, None)
        .await
}

/// Withdraws the redirected balances to the NFT owner
/// Calls the TokenizedAccountNFT's `withdrawBalances` function, which will send all balances of the input ERC20 addresses to the owner of the NFT
/// This function is protected and only callable by the owner of the NFT or someone approved to control it
/// Provide a custom timeout or None to wait ~30 seconds for the block
pub async fn withdraw_tokenized_account_balances(
    web30: &Web3,
    tokenized_account_contract: EthAddress,
    owner_or_approved: EthPrivateKey,
    erc20s: Vec<EthAddress>,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let timeout = timeout.unwrap_or(OPERATION_TIMEOUT);
    // ABI: withdrawBalances(address[] calldata erc20s) public virtual onlyOwnerOrApproved(AccountId)
    let payload = clarity::abi::encode_call(
        "withdrawBalances(address[])",
        &[erc20s.into()],
    )?;

    let transfer_res = web30
        .send_transaction(
            tokenized_account_contract,
            payload,
            0u8.into(),
            owner_or_approved,
            vec![],
        )
        .await?;

    web30
        .wait_for_transaction(transfer_res, timeout, None)
        .await
}

/// Withdraws the redirected balances to the specified address
/// Calls the TokenizedAccountNFT's `withdrawBalancesTo` function, which will send all balances of the input ERC20 addresses to the specified address
/// This function is protected and only callable by the owner of the NFT or someone approved to control it
/// Provide a custom timeout or None to wait ~30 seconds for the block
pub async fn withdraw_tokenized_account_balances_to(
    web30: &Web3,
    tokenized_account_contract: EthAddress,
    owner_or_approved: EthPrivateKey,
    withdraw_to: EthAddress,
    erc20s: Vec<EthAddress>,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let timeout = timeout.unwrap_or(OPERATION_TIMEOUT);
    // ABI: withdrawBalancesTo(address[] calldata erc20s, address destination) public virtual onlyOwnerOrApproved(AccountId)
    let payload = clarity::abi::encode_call(
        "withdrawBalancesTo(address[],address)",
        &[erc20s.into(), withdraw_to.into()],
    )?;

    let transfer_res = web30
        .send_transaction(
            tokenized_account_contract,
            payload,
            0u8.into(),
            owner_or_approved,
            vec![],
        )
        .await?;

    web30
        .wait_for_transaction(transfer_res, timeout, None)
        .await
}

// ==========================================================================================================================================
//                                                    ERC721 CONVENIENCE QUERY FUNCTIONS
// ==========================================================================================================================================

/// Queries for the owner of `token_id` on the ERC721 contract at `contract_address`, optionally using a `querier` address to make the call
/// If no querier is provided, then the `contract_address` will be used instead.
// Note: Provide a querier if the tokenized account nft contract does not have any native token, since simulated calls require gas
pub async fn get_erc721_owner(
    web30: &Web3,
    contract_address: EthAddress,
    token_id: Uint256,
    querier: Option<EthAddress>,
) -> Result<EthAddress, Web3Error> {
    let caller = querier.unwrap_or(contract_address);

    // ABI: ownerOf(uint256 tokenId) external view returns (address owner)
    let payload = clarity::abi::encode_call("ownerOf(uint256)", &[token_id.into()])?;
    let owner_res = web30
        .simulate_transaction(
            TransactionRequest::quick_tx(caller, contract_address, payload),
            None,
        )
        .await?;

    let owner = EthAddress::from_slice(match owner_res.get(12..32) {
        Some(val) => val,
        None => {
            return Err(Web3Error::ContractCallError(
                "erc721 ownerOf(uint256) failed".to_string(),
            ))
        }
    })?;

    Ok(owner)
}

// ==========================================================================================================================================
//                                                    ERC721 STATE CHANGING FUNCTIONS
// ==========================================================================================================================================

/// Sends the token with id `token_id` on the `erc721_address` contract from `from` to `to`.
/// Provide a custom timeout or None to wait ~30 seconds for the block
pub async fn send_erc721(
    web30: &Web3,
    erc721_address: EthAddress,
    token_id: Uint256,
    from: EthPrivateKey,
    to: EthAddress,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let timeout = timeout.unwrap_or(OPERATION_TIMEOUT);
    let transfer_from = from.to_address();
    // ABI: transferFrom(address from, address to, uint256 tokenId) external
    let payload = clarity::abi::encode_call(
        "transferFrom(address,address,uint256)",
        &[transfer_from.into(), to.into(), token_id.into()],
    )?;

    let transfer_res = web30
        .send_transaction(erc721_address, payload, 0u8.into(), from, vec![])
        .await?;

    web30
        .wait_for_transaction(transfer_res, timeout, None)
        .await
}

/// Approves `to_approve` to transfer the token with `token_id` owned by `approver` on the `erc721_address` contract
/// Provide a custom timeout or None to wait ~30 seconds for the block
pub async fn approve_erc721(
    web30: &Web3,
    erc721_address: EthAddress,
    token_id: Uint256,
    approver: EthPrivateKey,
    to_approve: EthAddress,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let timeout = timeout.unwrap_or(OPERATION_TIMEOUT);
    // ABI: approve(address to, uint256 tokenId) external
    let payload = clarity::abi::encode_call(
        "approve(address,uint256)",
        &[to_approve.into(), token_id.into()],
    )?;

    let approve_res = web30
        .send_transaction(erc721_address, payload, 0u8.into(), approver, vec![])
        .await?;

    web30.wait_for_transaction(approve_res, timeout, None).await
}

/// Approves `to_approve` to transfer any tokens owned by `approver` on the `erc721_address` contract
/// Provide a custom timeout or None to wait ~30 seconds for the block
pub async fn approve_erc721_for_all(
    web30: &Web3,
    erc721_address: EthAddress,
    approver: EthPrivateKey,
    to_approve: EthAddress,
    timeout: Option<Duration>,
) -> Result<TransactionResponse, Web3Error> {
    let timeout = timeout.unwrap_or(OPERATION_TIMEOUT);
    // ABI: setApprovalForAll(address operator, bool _approved) external
    let payload = clarity::abi::encode_call(
        "setApprovalForAll(address,bool)",
        &[to_approve.into(), true.into()],
    )?;

    let transfer_res = web30
        .send_transaction(erc721_address, payload, 0u8.into(), approver, vec![])
        .await?;

    web30
        .wait_for_transaction(transfer_res, timeout, None)
        .await
}
