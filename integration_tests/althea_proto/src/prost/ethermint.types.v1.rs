/// EthAccount implements the authtypes.AccountI interface and embeds an
/// authtypes.BaseAccount type. It is compatible with the auth AccountKeeper.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EthAccount {
    #[prost(message, optional, tag="1")]
    pub base_account: ::core::option::Option<cosmos_sdk_proto::cosmos::auth::v1beta1::BaseAccount>,
    #[prost(string, tag="2")]
    pub code_hash: ::prost::alloc::string::String,
}
/// TxResult is the value stored in eth tx indexer
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct TxResult {
    /// the block height
    #[prost(int64, tag="1")]
    pub height: i64,
    /// cosmos tx index
    #[prost(uint32, tag="2")]
    pub tx_index: u32,
    /// the msg index in a batch tx
    #[prost(uint32, tag="3")]
    pub msg_index: u32,
    /// eth tx index, the index in the list of valid eth tx in the block, 
    /// aka. the transaction list returned by eth_getBlock api.
    #[prost(int32, tag="4")]
    pub eth_tx_index: i32,
    /// if the eth tx is failed
    #[prost(bool, tag="5")]
    pub failed: bool,
    /// gas used by tx, if exceeds block gas limit,
    /// it's set to gas limit which is what's actually deducted by ante handler.
    #[prost(uint64, tag="6")]
    pub gas_used: u64,
    /// the cumulative gas used within current batch tx
    #[prost(uint64, tag="7")]
    pub cumulative_gas_used: u64,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ExtensionOptionsWeb3Tx {
    /// typed data chain id used only in EIP712 Domain and should match
    /// Ethereum network ID in a Web3 provider (e.g. Metamask).
    #[prost(uint64, tag="1")]
    pub typed_data_chain_id: u64,
    /// fee payer is an account address for the fee payer. It will be validated
    /// during EIP712 signature checking.
    #[prost(string, tag="2")]
    pub fee_payer: ::prost::alloc::string::String,
    /// fee payer sig is a signature data from the fee paying account,
    /// allows to perform fee delegation when using EIP712 Domain.
    #[prost(bytes="vec", tag="3")]
    pub fee_payer_sig: ::prost::alloc::vec::Vec<u8>,
}
