/// QueryBalancesRequest is the request type for the Query/Balances RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryBalancesRequest {
    /// address of the clawback vesting account
    #[prost(string, tag="1")]
    pub address: ::prost::alloc::string::String,
}
/// QueryBalancesResponse is the response type for the Query/Balances RPC
/// method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryBalancesResponse {
    /// current amount of locked tokens
    #[prost(message, repeated, tag="1")]
    pub locked: ::prost::alloc::vec::Vec<cosmos_sdk_proto::cosmos::base::v1beta1::Coin>,
    /// current amount of unvested tokens
    #[prost(message, repeated, tag="2")]
    pub unvested: ::prost::alloc::vec::Vec<cosmos_sdk_proto::cosmos::base::v1beta1::Coin>,
    /// current amount of vested tokens
    #[prost(message, repeated, tag="3")]
    pub vested: ::prost::alloc::vec::Vec<cosmos_sdk_proto::cosmos::base::v1beta1::Coin>,
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
        /// Retrieves the unvested, vested and locked tokens for a vesting account
        pub async fn balances(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryBalancesRequest>,
        ) -> Result<tonic::Response<super::QueryBalancesResponse>, tonic::Status> {
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
                "/canto.vesting.v1.Query/Balances",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
    }
}
/// MsgCreateClawbackVestingAccount defines a message that enables creating a
/// ClawbackVestingAccount.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgCreateClawbackVestingAccount {
    /// from_address specifies the account to provide the funds and sign the
    /// clawback request
    #[prost(string, tag="1")]
    pub from_address: ::prost::alloc::string::String,
    /// to_address specifies the account to receive the funds
    #[prost(string, tag="2")]
    pub to_address: ::prost::alloc::string::String,
    /// start_time defines the time at which the vesting period begins
    #[prost(message, optional, tag="3")]
    pub start_time: ::core::option::Option<::prost_types::Timestamp>,
    /// lockup_periods defines the unlocking schedule relative to the start_time
    #[prost(message, repeated, tag="4")]
    pub lockup_periods: ::prost::alloc::vec::Vec<cosmos_sdk_proto::cosmos::vesting::v1beta1::Period>,
    /// vesting_periods defines thevesting schedule relative to the start_time
    #[prost(message, repeated, tag="5")]
    pub vesting_periods: ::prost::alloc::vec::Vec<cosmos_sdk_proto::cosmos::vesting::v1beta1::Period>,
    /// merge specifies a the creation mechanism for existing
    /// ClawbackVestingAccounts. If true, merge this new grant into an existing
    /// ClawbackVestingAccount, or create it if it does not exist. If false,
    /// creates a new account. New grants to an existing account must be from the
    /// same from_address.
    #[prost(bool, tag="6")]
    pub merge: bool,
}
/// MsgCreateClawbackVestingAccountResponse defines the
/// MsgCreateClawbackVestingAccount response type.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgCreateClawbackVestingAccountResponse {
}
/// MsgClawback defines a message that removes unvested tokens from a
/// ClawbackVestingAccount.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgClawback {
    /// funder_address is the address which funded the account
    #[prost(string, tag="1")]
    pub funder_address: ::prost::alloc::string::String,
    /// account_address is the address of the ClawbackVestingAccount to claw back
    /// from.
    #[prost(string, tag="2")]
    pub account_address: ::prost::alloc::string::String,
    /// dest_address specifies where the clawed-back tokens should be transferred
    /// to. If empty, the tokens will be transferred back to the original funder of
    /// the account.
    #[prost(string, tag="3")]
    pub dest_address: ::prost::alloc::string::String,
}
/// MsgClawbackResponse defines the MsgClawback response type.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MsgClawbackResponse {
}
/// Generated client implementations.
pub mod msg_client {
    #![allow(unused_variables, dead_code, missing_docs, clippy::let_unit_value)]
    use tonic::codegen::*;
    /// Msg defines the vesting Msg service.
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
        /// CreateClawbackVestingAccount creats a vesting account that is subject to
        /// clawback and the configuration of vesting and lockup schedules.
        pub async fn create_clawback_vesting_account(
            &mut self,
            request: impl tonic::IntoRequest<super::MsgCreateClawbackVestingAccount>,
        ) -> Result<
            tonic::Response<super::MsgCreateClawbackVestingAccountResponse>,
            tonic::Status,
        > {
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
                "/canto.vesting.v1.Msg/CreateClawbackVestingAccount",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// Clawback removes the unvested tokens from a ClawbackVestingAccount.
        pub async fn clawback(
            &mut self,
            request: impl tonic::IntoRequest<super::MsgClawback>,
        ) -> Result<tonic::Response<super::MsgClawbackResponse>, tonic::Status> {
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
                "/canto.vesting.v1.Msg/Clawback",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
    }
}
/// ClawbackVestingAccount implements the VestingAccount interface. It provides
/// an account that can hold contributions subject to "lockup" (like a
/// PeriodicVestingAccount), or vesting which is subject to clawback
/// of unvested tokens, or a combination (tokens vest, but are still locked).
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ClawbackVestingAccount {
    /// base_vesting_account implements the VestingAccount interface. It contains
    /// all the necessary fields needed for any vesting account implementation
    #[prost(message, optional, tag="1")]
    pub base_vesting_account: ::core::option::Option<cosmos_sdk_proto::cosmos::vesting::v1beta1::BaseVestingAccount>,
    /// funder_address specifies the account which can perform clawback
    #[prost(string, tag="2")]
    pub funder_address: ::prost::alloc::string::String,
    /// start_time defines the time at which the vesting period begins
    #[prost(message, optional, tag="3")]
    pub start_time: ::core::option::Option<::prost_types::Timestamp>,
    /// lockup_periods defines the unlocking schedule relative to the start_time
    #[prost(message, repeated, tag="4")]
    pub lockup_periods: ::prost::alloc::vec::Vec<cosmos_sdk_proto::cosmos::vesting::v1beta1::Period>,
    /// vesting_periods defines the vesting schedule relative to the start_time
    #[prost(message, repeated, tag="5")]
    pub vesting_periods: ::prost::alloc::vec::Vec<cosmos_sdk_proto::cosmos::vesting::v1beta1::Period>,
}
