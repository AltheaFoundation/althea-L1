# Cardinal Upgrade

The *Cardnal* upgrade contains the following changes.

## Summary of Changes

* Add new x/erc20 module messages for machine accounts:
* * MsgSendCoinToEVM allows accounts to send Cosmos coins to the EVM and pay fees denominated in that coin
* * MsgSendERC20ToCosmos allows accounts to send ERC20s to the Cosmos layer and pay fees denominated in that ERC20
* * MsgSendERC20ToCosmosAndIBCTransfer allows accounts to send ERC20s to IBC connected chains, paying fees in that ERC20
* * These messages will allow machine accounts to manage funds without needing to trade on a DEX, rely on a faucet, or be topped up with gas manually
* Update the iFi DEX configuration to enable Concentrated liquidity positions on stablecoin pairs (e.g. USDC/USDT)
* Enable the nativedex module to manage iFi DEX configuration via governance proposals