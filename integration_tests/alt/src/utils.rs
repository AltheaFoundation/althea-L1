use std::time::Duration;

use clarity::{Address as EthAddress, PrivateKey};
use num256::Uint256;
use web30::client::Web3;

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
            info!("Approving DEX to spend {amount} base token wei");
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
        info!("Approving DEX to spend {amount} quote token wei");
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
