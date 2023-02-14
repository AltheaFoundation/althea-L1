/// The CSR struct is a wrapper to all of the metadata associated with a given CST NFT
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Csr {
    /// Contracts is the list of all EVM address that are registered to this NFT
    #[prost(string, repeated, tag="1")]
    pub contracts: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
    /// The NFT id which this CSR corresponds to
    #[prost(uint64, tag="2")]
    pub id: u64,
    /// The total number of transactions for this CSR NFT
    #[prost(uint64, tag="3")]
    pub txs: u64,
    /// The cumulative revenue for this CSR NFT -> represented as a sdk.Int
    #[prost(string, tag="4")]
    pub revenue: ::prost::alloc::string::String,
}
/// Params holds parameters for the csr module
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Params {
    /// boolean to enable the csr module
    #[prost(bool, tag="1")]
    pub enable_csr: bool,
    /// decimal to determine the transaction fee split between network operators (validators) and CSR
    #[prost(string, tag="2")]
    pub csr_shares: ::prost::alloc::string::String,
}
/// GenesisState defines the csr module's genesis state.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    /// params defines all of the parameters of the module
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
}
/// QueryParamsRequest is the request type for the Query/Params RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryParamsRequest {
}
/// QueryParamsResponse is the response type for the Query/Params RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryParamsResponse {
    /// params holds all the parameters of this module.
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
}
/// QueryCSRsRequest is the request type for the Query/CSRs RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCsRsRequest {
    /// pagination defines an optional pagination for the request.
    #[prost(message, optional, tag="1")]
    pub pagination: ::core::option::Option<cosmos_sdk_proto::cosmos::base::query::v1beta1::PageRequest>,
}
/// QueryCSRsResponse is the response type for the Query/CSRs RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCsRsResponse {
    #[prost(message, repeated, tag="1")]
    pub csrs: ::prost::alloc::vec::Vec<Csr>,
    /// pagination for response
    #[prost(message, optional, tag="2")]
    pub pagination: ::core::option::Option<cosmos_sdk_proto::cosmos::base::query::v1beta1::PageResponse>,
}
/// QueryCSRByNFTRequest is the request type for the Query/CSRByNFT RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCsrByNftRequest {
    #[prost(uint64, tag="1")]
    pub nft_id: u64,
}
/// QueryCSRByNFTResponse is the response type for the Query/CSRByNFT RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCsrByNftResponse {
    /// csr object queried by nft id
    #[prost(message, optional, tag="1")]
    pub csr: ::core::option::Option<Csr>,
}
/// QueryCSRByContractRequest is the request type for the Query/CSRByContract RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCsrByContractRequest {
    #[prost(string, tag="1")]
    pub address: ::prost::alloc::string::String,
}
/// QueryCSRByContractResponse is the response type for the Query/CSRByContract RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCsrByContractResponse {
    /// csr object queried by smart contract address
    #[prost(message, optional, tag="1")]
    pub csr: ::core::option::Option<Csr>,
}
/// QueryTurnstileRequest is the request type for the Query/Turnstile RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryTurnstileRequest {
}
/// QueryTurnstileResponse is the response type for the Query/Turnstile RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryTurnstileResponse {
    #[prost(string, tag="1")]
    pub address: ::prost::alloc::string::String,
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
        /// Parameters queries the parameters of the module.
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
                "/canto.csr.v1.Query/Params",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// query all registered CSRs
        pub async fn cs_rs(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryCsRsRequest>,
        ) -> Result<tonic::Response<super::QueryCsRsResponse>, tonic::Status> {
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
            let path = http::uri::PathAndQuery::from_static("/canto.csr.v1.Query/CSRs");
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// query a specific CSR by the nftId
        pub async fn csr_by_nft(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryCsrByNftRequest>,
        ) -> Result<tonic::Response<super::QueryCsrByNftResponse>, tonic::Status> {
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
                "/canto.csr.v1.Query/CSRByNFT",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// query a CSR by smart contract address
        pub async fn csr_by_contract(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryCsrByContractRequest>,
        ) -> Result<tonic::Response<super::QueryCsrByContractResponse>, tonic::Status> {
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
                "/canto.csr.v1.Query/CSRByContract",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// query the turnstile address
        pub async fn turnstile(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryTurnstileRequest>,
        ) -> Result<tonic::Response<super::QueryTurnstileResponse>, tonic::Status> {
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
                "/canto.csr.v1.Query/Turnstile",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
    }
}
