/// Fee defines an instance that organizes fee distribution conditions for the
/// owner of a given smart contract
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Fee {
    /// hex address of registered contract
    #[prost(string, tag="1")]
    pub contract_address: ::prost::alloc::string::String,
    /// bech32 address of contract deployer
    #[prost(string, tag="2")]
    pub deployer_address: ::prost::alloc::string::String,
    /// bech32 address of account receiving the transaction fees it defaults to
    /// deployer_address
    #[prost(string, tag="3")]
    pub withdraw_address: ::prost::alloc::string::String,
}
/// GenesisState defines the module's genesis state.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    /// module parameters
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
    /// active registered contracts for fee distribution
    #[prost(message, repeated, tag="2")]
    pub fees: ::prost::alloc::vec::Vec<Fee>,
}
/// Params defines the fees module params
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Params {
    /// parameter to enable fees
    #[prost(bool, tag="1")]
    pub enable_fees: bool,
    /// developer_shares defines the proportion of the transaction fees to be
    /// distributed to the registered contract owner
    #[prost(string, tag="2")]
    pub developer_shares: ::prost::alloc::string::String,
    /// addr_derivation_cost_create defines the cost of address derivation for
    /// verifying the contract deployer at fee registration
    #[prost(uint64, tag="3")]
    pub addr_derivation_cost_create: u64,
}
/// QueryFeesRequest is the request type for the Query/Fees RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryFeesRequest {
    /// pagination defines an optional pagination for the request.
    #[prost(message, optional, tag="1")]
    pub pagination: ::core::option::Option<cosmos_sdk_proto::cosmos::base::query::v1beta1::PageRequest>,
}
/// QueryFeesResponse is the response type for the Query/Fees RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryFeesResponse {
    #[prost(message, repeated, tag="1")]
    pub fees: ::prost::alloc::vec::Vec<Fee>,
    /// pagination defines the pagination in the response.
    #[prost(message, optional, tag="2")]
    pub pagination: ::core::option::Option<cosmos_sdk_proto::cosmos::base::query::v1beta1::PageResponse>,
}
/// QueryFeeRequest is the request type for the Query/Fee RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryFeeRequest {
    /// contract identifier is the hex contract address of a contract
    #[prost(string, tag="1")]
    pub contract_address: ::prost::alloc::string::String,
}
/// QueryFeeResponse is the response type for the Query/Fee RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryFeeResponse {
    #[prost(message, optional, tag="1")]
    pub fee: ::core::option::Option<Fee>,
}
/// QueryParamsRequest is the request type for the Query/Params RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryParamsRequest {
}
/// QueryParamsResponse is the response type for the Query/Params RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryParamsResponse {
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
}
/// QueryDeployerFeesRequest is the request type for the Query/DeployerFees RPC
/// method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryDeployerFeesRequest {
    /// deployer bech32 address
    #[prost(string, tag="1")]
    pub deployer_address: ::prost::alloc::string::String,
    /// pagination defines an optional pagination for the request.
    #[prost(message, optional, tag="2")]
    pub pagination: ::core::option::Option<cosmos_sdk_proto::cosmos::base::query::v1beta1::PageRequest>,
}
/// QueryDeployerFeesResponse is the response type for the Query/DeployerFees RPC
/// method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryDeployerFeesResponse {
    #[prost(message, repeated, tag="1")]
    pub fees: ::prost::alloc::vec::Vec<Fee>,
    /// pagination defines the pagination in the response.
    #[prost(message, optional, tag="2")]
    pub pagination: ::core::option::Option<cosmos_sdk_proto::cosmos::base::query::v1beta1::PageResponse>,
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
        /// Fees retrieves all registered contracts for fee distribution
        pub async fn fees(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryFeesRequest>,
        ) -> Result<tonic::Response<super::QueryFeesResponse>, tonic::Status> {
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
            let path = http::uri::PathAndQuery::from_static("/canto.fees.v1.Query/Fees");
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// Fee retrieves a registered contract for fee distribution for a given
        /// address
        pub async fn fee(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryFeeRequest>,
        ) -> Result<tonic::Response<super::QueryFeeResponse>, tonic::Status> {
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
            let path = http::uri::PathAndQuery::from_static("/canto.fees.v1.Query/Fee");
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// Params retrieves the fees module params
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
                "/canto.fees.v1.Query/Params",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// DeployerFees retrieves all contracts that a given deployer has registered
        /// for fee distribution
        pub async fn deployer_fees(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryDeployerFeesRequest>,
        ) -> Result<tonic::Response<super::QueryDeployerFeesResponse>, tonic::Status> {
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
                "/canto.fees.v1.Query/DeployerFees",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
    }
}
/// MsgRegisterFee defines a message that registers a Fee
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgRegisterFee {
    /// contract hex address
    #[prost(string, tag="1")]
    pub contract_address: ::prost::alloc::string::String,
    /// bech32 address of message sender, must be the same as the origin EOA
    /// sending the transaction which deploys the contract
    #[prost(string, tag="2")]
    pub deployer_address: ::prost::alloc::string::String,
    /// bech32 address of account receiving the transaction fees
    #[prost(string, tag="3")]
    pub withdraw_address: ::prost::alloc::string::String,
    /// array of nonces from the address path, where the last nonce is
    /// the nonce that determines the contract's address - it can be an EOA nonce
    /// or a factory contract nonce
    #[prost(uint64, repeated, tag="4")]
    pub nonces: ::prost::alloc::vec::Vec<u64>,
}
/// MsgRegisterFeeResponse defines the MsgRegisterFee response type
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgRegisterFeeResponse {
}
/// MsgCancelFee defines a message that cancels a registered a Fee
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgCancelFee {
    /// contract hex address
    #[prost(string, tag="1")]
    pub contract_address: ::prost::alloc::string::String,
    /// deployer bech32 address
    #[prost(string, tag="2")]
    pub deployer_address: ::prost::alloc::string::String,
}
/// MsgCancelFeeResponse defines the MsgCancelFee response type
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgCancelFeeResponse {
}
/// MsgUpdateFee defines a message that updates the withdraw address for a
/// registered Fee
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgUpdateFee {
    /// contract hex address
    #[prost(string, tag="1")]
    pub contract_address: ::prost::alloc::string::String,
    /// deployer bech32 address
    #[prost(string, tag="2")]
    pub deployer_address: ::prost::alloc::string::String,
    /// new withdraw bech32 address for receiving the transaction fees
    #[prost(string, tag="3")]
    pub withdraw_address: ::prost::alloc::string::String,
}
/// MsgUpdateFeeResponse defines the MsgUpdateFee response type
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgUpdateFeeResponse {
}
/// Generated client implementations.
pub mod msg_client {
    #![allow(unused_variables, dead_code, missing_docs, clippy::let_unit_value)]
    use tonic::codegen::*;
    /// Msg defines the fees Msg service.
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
        /// RegisterFee registers a new contract for receiving transaction fees
        pub async fn register_fee(
            &mut self,
            request: impl tonic::IntoRequest<super::MsgRegisterFee>,
        ) -> Result<tonic::Response<super::MsgRegisterFeeResponse>, tonic::Status> {
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
                "/canto.fees.v1.Msg/RegisterFee",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// CancelFee cancels a contract's fee registration and further receival of
        /// transaction fees
        pub async fn cancel_fee(
            &mut self,
            request: impl tonic::IntoRequest<super::MsgCancelFee>,
        ) -> Result<tonic::Response<super::MsgCancelFeeResponse>, tonic::Status> {
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
                "/canto.fees.v1.Msg/CancelFee",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// UpdateFee updates the withdraw address
        pub async fn update_fee(
            &mut self,
            request: impl tonic::IntoRequest<super::MsgUpdateFee>,
        ) -> Result<tonic::Response<super::MsgUpdateFeeResponse>, tonic::Status> {
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
                "/canto.fees.v1.Msg/UpdateFee",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
    }
}
