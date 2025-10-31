# Cardinal Upgrade

The *Cardnal* upgrade contains the following changes.

## Summary of Changes

* Add new gasfree x/erc20 module messages for machine accounts. These new messages will make it possible for machine accounts to work on Althea-L1 without ever needing to acquire the native gas token (ALTHEA).
   * MsgSendCoinToEVM allows accounts to send Cosmos coins to the EVM and pay fees denominated in that coin
   * MsgSendERC20ToCosmos allows accounts to send ERC20s to the Cosmos layer and pay fees denominated in that ERC20
   * MsgSendERC20ToCosmosAndIBCTransfer allows accounts to send ERC20s to IBC connected chains, paying fees in that ERC20
   * These messages will allow machine accounts to manage funds without needing to trade on a DEX, rely on a faucet, or be topped up with gas manually
   Update the iFi DEX configuration to enable Concentrated liquidity positions on stablecoin pairs (e.g. USDC/USDT).
   * These new messages will be added to the gasfree module's GasFreeMessageTypes Param to make them exempt from normal Cosmos Tx fee collection. The message handler itself will collect a fee.
   * The fee for these new message types is controlled by the gasfree module's GasFreeERC20InteropFeeBasisPoints Param, which will be set to 100 basis points during this upgrade. This corresponds to a 1% fee which will be collected on top of the amount converted/transferred. For example, if an account wants 100 USDC on the EVM layer on another chain it can submit a MsgSendERC20ToCosmosAndIBCTransfer which will convert 101 USDC to the Cosmos layer, enqueue an IBC Transfer of 100 USDC, and the 1 USDC fee will be distributed to validators and stakers. Fees for these messages are always collected on the Cosmos layer, and will be collected on top of the converted amount.
   * These new messages can only be used with whitelisted tokens, which are set in the gasfree module's GasFreeERC20InteropTokens Param. If an account tries to convert a token which is not on this list, the message will fail. The whitelisted tokens for these messages will be set in the upgrade to USDC, USDS, sUSDS, and USDT.
* Add a new ExecuteContractProposal, which can execute any contract call as the nativedex module's account on whitelisted contracts
    *  The whitelist is a new nativedex Param called WhitelistedContractAddresses
    *  This proposal gives Althea-L1 governance the ability to control iFi DEX incentives directly by whitelisting an incentives contract. Incentives can potentially be funded via a CommunityPoolSpendProposal.
* Enable the nativedex module to manage iFi DEX configuration via governance proposals
* iFi DEX configuration on Stablecoin-pair pools
    * With the initial deployment of the iFi DEX, the configuration was incorrectly set for the Stablecoin-pair pool template, which controls pools where both tokens are Stablecoins like USDC/USDT. This misconfiguration makes it impossible to create Concentrated (Ranged) liquidity positions on these pools.
    * The configuration template for new Stablecoin-pair pools (index 36000) will be updated to unlock Concentrated liquidity positions. 
    * Update the Stablecoin-pair pools to use the new configuration. These pools are USDC / USDS, sUSDS / USDS, and USDS / USDT.
    * Any new pools created using this template will use the updated configuration.