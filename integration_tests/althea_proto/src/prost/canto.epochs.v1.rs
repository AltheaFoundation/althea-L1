#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EpochInfo {
    #[prost(string, tag="1")]
    pub identifier: ::prost::alloc::string::String,
    #[prost(message, optional, tag="2")]
    pub start_time: ::core::option::Option<::prost_types::Timestamp>,
    #[prost(message, optional, tag="3")]
    pub duration: ::core::option::Option<::prost_types::Duration>,
    #[prost(int64, tag="4")]
    pub current_epoch: i64,
    #[prost(message, optional, tag="5")]
    pub current_epoch_start_time: ::core::option::Option<::prost_types::Timestamp>,
    #[prost(bool, tag="6")]
    pub epoch_counting_started: bool,
    #[prost(int64, tag="7")]
    pub current_epoch_start_height: i64,
}
/// GenesisState defines the epochs module's genesis state.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    #[prost(message, repeated, tag="1")]
    pub epochs: ::prost::alloc::vec::Vec<EpochInfo>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryEpochsInfoRequest {
    #[prost(message, optional, tag="1")]
    pub pagination: ::core::option::Option<cosmos_sdk_proto::cosmos::base::query::v1beta1::PageRequest>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryEpochsInfoResponse {
    #[prost(message, repeated, tag="1")]
    pub epochs: ::prost::alloc::vec::Vec<EpochInfo>,
    #[prost(message, optional, tag="2")]
    pub pagination: ::core::option::Option<cosmos_sdk_proto::cosmos::base::query::v1beta1::PageResponse>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCurrentEpochRequest {
    #[prost(string, tag="1")]
    pub identifier: ::prost::alloc::string::String,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCurrentEpochResponse {
    #[prost(int64, tag="1")]
    pub current_epoch: i64,
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
        /// EpochInfos provide running epochInfos
        pub async fn epoch_infos(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryEpochsInfoRequest>,
        ) -> Result<tonic::Response<super::QueryEpochsInfoResponse>, tonic::Status> {
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
                "/canto.epochs.v1.Query/EpochInfos",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// CurrentEpoch provide current epoch of specified identifier
        pub async fn current_epoch(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryCurrentEpochRequest>,
        ) -> Result<tonic::Response<super::QueryCurrentEpochResponse>, tonic::Status> {
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
                "/canto.epochs.v1.Query/CurrentEpoch",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
    }
}
