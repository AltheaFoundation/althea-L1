/// TokenPair defines an instance that records a pairing consisting of a native
///  Cosmos Coin and an ERC20 token address.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct TokenPair {
    /// address of ERC20 contract token
    #[prost(string, tag="1")]
    pub erc20_address: ::prost::alloc::string::String,
    /// cosmos base denomination to be mapped to
    #[prost(string, tag="2")]
    pub denom: ::prost::alloc::string::String,
    /// shows token mapping enable status
    #[prost(bool, tag="3")]
    pub enabled: bool,
    /// ERC20 owner address ENUM (0 invalid, 1 ModuleAccount, 2 external address)
    #[prost(enumeration="Owner", tag="4")]
    pub contract_owner: i32,
}
/// RegisterCoinProposal is a gov Content type to register a token pair for a
/// native Cosmos coin.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct RegisterCoinProposal {
    /// title of the proposal
    #[prost(string, tag="1")]
    pub title: ::prost::alloc::string::String,
    /// proposal description
    #[prost(string, tag="2")]
    pub description: ::prost::alloc::string::String,
    /// metadata of the native Cosmos coin
    #[prost(message, optional, tag="3")]
    pub metadata: ::core::option::Option<cosmos_sdk_proto::cosmos::bank::v1beta1::Metadata>,
}
/// RegisterERC20Proposal is a gov Content type to register a token pair for an
/// ERC20 token
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct RegisterErc20Proposal {
    /// title of the proposa  string title = 1;
    #[prost(string, tag="1")]
    pub title: ::prost::alloc::string::String,
    /// proposal description
    #[prost(string, tag="2")]
    pub description: ::prost::alloc::string::String,
    /// contract address of ERC20 token
    #[prost(string, tag="3")]
    pub erc20address: ::prost::alloc::string::String,
}
/// ToggleTokenConversionProposal is a gov Content type to toggle the conversion
/// of a token pair.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ToggleTokenConversionProposal {
    /// title of the proposal
    #[prost(string, tag="1")]
    pub title: ::prost::alloc::string::String,
    /// proposal description
    #[prost(string, tag="2")]
    pub description: ::prost::alloc::string::String,
    /// token identifier can be either the hex contract address of the ERC20 or the
    /// Cosmos base denomination
    #[prost(string, tag="3")]
    pub token: ::prost::alloc::string::String,
}
/// Owner enumerates the ownership of a ERC20 contract.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
#[repr(i32)]
pub enum Owner {
    /// OWNER_UNSPECIFIED defines an invalid/undefined owner.
    Unspecified = 0,
    /// OWNER_MODULE erc20 is owned by the erc20 module account.
    Module = 1,
    /// EXTERNAL erc20 is owned by an external account.
    External = 2,
}
/// GenesisState defines the module's genesis state.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    /// module parameters
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
    /// registered token pairs
    #[prost(message, repeated, tag="2")]
    pub token_pairs: ::prost::alloc::vec::Vec<TokenPair>,
}
/// Params defines the erc20 module params
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Params {
    /// parameter to enable the conversion of Cosmos coins <--> ERC20 tokens.
    #[prost(bool, tag="1")]
    pub enable_erc20: bool,
    /// parameter to enable the EVM hook that converts an ERC20 token to a Cosmos
    /// Coin by transferring the Tokens through a MsgEthereumTx to the
    /// ModuleAddress Ethereum address.
    #[prost(bool, tag="2")]
    pub enable_evm_hook: bool,
}
/// QueryTokenPairsRequest is the request type for the Query/TokenPairs RPC
/// method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryTokenPairsRequest {
    /// pagination defines an optional pagination for the request.
    #[prost(message, optional, tag="1")]
    pub pagination: ::core::option::Option<cosmos_sdk_proto::cosmos::base::query::v1beta1::PageRequest>,
}
/// QueryTokenPairsResponse is the response type for the Query/TokenPairs RPC
/// method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryTokenPairsResponse {
    #[prost(message, repeated, tag="1")]
    pub token_pairs: ::prost::alloc::vec::Vec<TokenPair>,
    /// pagination defines the pagination in the response.
    #[prost(message, optional, tag="2")]
    pub pagination: ::core::option::Option<cosmos_sdk_proto::cosmos::base::query::v1beta1::PageResponse>,
}
/// QueryTokenPairRequest is the request type for the Query/TokenPair RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryTokenPairRequest {
    /// token identifier can be either the hex contract address of the ERC20 or the
    /// Cosmos base denomination
    #[prost(string, tag="1")]
    pub token: ::prost::alloc::string::String,
}
/// QueryTokenPairResponse is the response type for the Query/TokenPair RPC
/// method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryTokenPairResponse {
    #[prost(message, optional, tag="1")]
    pub token_pair: ::core::option::Option<TokenPair>,
}
/// QueryParamsRequest is the request type for the Query/Params RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryParamsRequest {
}
/// QueryParamsResponse is the response type for the Query/Params RPC
/// method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryParamsResponse {
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
}
/// Generated client implementations.
pub mod query_client {
    #![allow(unused_variables, dead_code, missing_docs, clippy::let_unit_value)]
    use tonic::codegen::*;
    /// Query defines the gRPC querier service.
    #[derive(Debug, Clone)]
    pub struct QueryClient<T> {
        inner: tonic::client::Grpc<T>,
    }
    impl QueryClient<tonic::transport::Channel> {
        /// Attempt to create a new client by connecting to a given endpoint.
        pub async fn connect<D>(dst: D) -> Result<Self, tonic::transport::Error>
        where
            D: std::convert::TryInto<tonic::transport::Endpoint>,
            D::Error: Into<StdError>,
        {
            let conn = tonic::transport::Endpoint::new(dst)?.connect().await?;
            Ok(Self::new(conn))
        }
    }
    impl<T> QueryClient<T>
    where
        T: tonic::client::GrpcService<tonic::body::BoxBody>,
        T::Error: Into<StdError>,
        T::ResponseBody: Body<Data = Bytes> + Send + 'static,
        <T::ResponseBody as Body>::Error: Into<StdError> + Send,
    {
        pub fn new(inner: T) -> Self {
            let inner = tonic::client::Grpc::new(inner);
            Self { inner }
        }
        pub fn with_interceptor<F>(
            inner: T,
            interceptor: F,
        ) -> QueryClient<InterceptedService<T, F>>
        where
            F: tonic::service::Interceptor,
            T::ResponseBody: Default,
            T: tonic::codegen::Service<
                http::Request<tonic::body::BoxBody>,
                Response = http::Response<
                    <T as tonic::client::GrpcService<tonic::body::BoxBody>>::ResponseBody,
                >,
            >,
            <T as tonic::codegen::Service<
                http::Request<tonic::body::BoxBody>,
            >>::Error: Into<StdError> + Send + Sync,
        {
            QueryClient::new(InterceptedService::new(inner, interceptor))
        }
        /// Compress requests with `gzip`.
        ///
        /// This requires the server to support it otherwise it might respond with an
        /// error.
        #[must_use]
        pub fn send_gzip(mut self) -> Self {
            self.inner = self.inner.send_gzip();
            self
        }
        /// Enable decompressing responses with `gzip`.
        #[must_use]
        pub fn accept_gzip(mut self) -> Self {
            self.inner = self.inner.accept_gzip();
            self
        }
        /// TokenPairs retrieves registered token pairs
        pub async fn token_pairs(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryTokenPairsRequest>,
        ) -> Result<tonic::Response<super::QueryTokenPairsResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/canto.erc20.v1.Query/TokenPairs",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// TokenPair retrieves a registered token pair
        pub async fn token_pair(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryTokenPairRequest>,
        ) -> Result<tonic::Response<super::QueryTokenPairResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/canto.erc20.v1.Query/TokenPair",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// Params retrieves the erc20 module params
        pub async fn params(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryParamsRequest>,
        ) -> Result<tonic::Response<super::QueryParamsResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/canto.erc20.v1.Query/Params",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
    }
}
/// MsgConvertCoin defines a Msg to convert a native Cosmos coin to a ERC20 token
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgConvertCoin {
    /// Cosmos coin which denomination is registered in a token pair. The coin
    /// amount defines the amount of coins to convert.
    #[prost(message, optional, tag="1")]
    pub coin: ::core::option::Option<cosmos_sdk_proto::cosmos::base::v1beta1::Coin>,
    /// recipient hex address to receive ERC20 token
    #[prost(string, tag="2")]
    pub receiver: ::prost::alloc::string::String,
    /// cosmos bech32 address from the owner of the given Cosmos coins
    #[prost(string, tag="3")]
    pub sender: ::prost::alloc::string::String,
}
/// MsgConvertCoinResponse returns no fields
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgConvertCoinResponse {
}
/// MsgConvertERC20 defines a Msg to convert a ERC20 token to a native Cosmos
/// coin.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgConvertErc20 {
    /// ERC20 token contract address registered in a token pair
    #[prost(string, tag="1")]
    pub contract_address: ::prost::alloc::string::String,
    /// amount of ERC20 tokens to convert
    #[prost(string, tag="2")]
    pub amount: ::prost::alloc::string::String,
    /// bech32 address to receive native Cosmos coins
    #[prost(string, tag="3")]
    pub receiver: ::prost::alloc::string::String,
    /// sender hex address from the owner of the given ERC20 tokens
    #[prost(string, tag="4")]
    pub sender: ::prost::alloc::string::String,
}
/// MsgConvertERC20Response returns no fields
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgConvertErc20Response {
}
/// Generated client implementations.
pub mod msg_client {
    #![allow(unused_variables, dead_code, missing_docs, clippy::let_unit_value)]
    use tonic::codegen::*;
    /// Msg defines the erc20 Msg service.
    #[derive(Debug, Clone)]
    pub struct MsgClient<T> {
        inner: tonic::client::Grpc<T>,
    }
    impl MsgClient<tonic::transport::Channel> {
        /// Attempt to create a new client by connecting to a given endpoint.
        pub async fn connect<D>(dst: D) -> Result<Self, tonic::transport::Error>
        where
            D: std::convert::TryInto<tonic::transport::Endpoint>,
            D::Error: Into<StdError>,
        {
            let conn = tonic::transport::Endpoint::new(dst)?.connect().await?;
            Ok(Self::new(conn))
        }
    }
    impl<T> MsgClient<T>
    where
        T: tonic::client::GrpcService<tonic::body::BoxBody>,
        T::Error: Into<StdError>,
        T::ResponseBody: Body<Data = Bytes> + Send + 'static,
        <T::ResponseBody as Body>::Error: Into<StdError> + Send,
    {
        pub fn new(inner: T) -> Self {
            let inner = tonic::client::Grpc::new(inner);
            Self { inner }
        }
        pub fn with_interceptor<F>(
            inner: T,
            interceptor: F,
        ) -> MsgClient<InterceptedService<T, F>>
        where
            F: tonic::service::Interceptor,
            T::ResponseBody: Default,
            T: tonic::codegen::Service<
                http::Request<tonic::body::BoxBody>,
                Response = http::Response<
                    <T as tonic::client::GrpcService<tonic::body::BoxBody>>::ResponseBody,
                >,
            >,
            <T as tonic::codegen::Service<
                http::Request<tonic::body::BoxBody>,
            >>::Error: Into<StdError> + Send + Sync,
        {
            MsgClient::new(InterceptedService::new(inner, interceptor))
        }
        /// Compress requests with `gzip`.
        ///
        /// This requires the server to support it otherwise it might respond with an
        /// error.
        #[must_use]
        pub fn send_gzip(mut self) -> Self {
            self.inner = self.inner.send_gzip();
            self
        }
        /// Enable decompressing responses with `gzip`.
        #[must_use]
        pub fn accept_gzip(mut self) -> Self {
            self.inner = self.inner.accept_gzip();
            self
        }
        /// ConvertCoin mints a ERC20 representation of the native Cosmos coin denom
        /// that is registered on the token mapping.
        pub async fn convert_coin(
            &mut self,
            request: impl tonic::IntoRequest<super::MsgConvertCoin>,
        ) -> Result<tonic::Response<super::MsgConvertCoinResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/canto.erc20.v1.Msg/ConvertCoin",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// ConvertERC20 mints a native Cosmos coin representation of the ERC20 token
        /// contract that is registered on the token mapping.
        pub async fn convert_erc20(
            &mut self,
            request: impl tonic::IntoRequest<super::MsgConvertErc20>,
        ) -> Result<tonic::Response<super::MsgConvertErc20Response>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/canto.erc20.v1.Msg/ConvertERC20",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
    }
}
