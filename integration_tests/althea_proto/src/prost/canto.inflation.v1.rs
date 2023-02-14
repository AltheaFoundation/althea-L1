/// InflationDistribution defines the distribution in which inflation is
/// allocated through minting on each epoch (staking, incentives, community). It
/// excludes the team vesting distribution, as this is minted once at genesis.
/// The initial InflationDistribution can be calculated from the Evmos Token
/// Model like this:
/// mintDistribution1 = distribution1 / (1 - teamVestingDistribution)
/// 0.5333333         = 40%           / (1 - 25%)
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct InflationDistribution {
    /// staking_rewards defines the proportion of the minted minted_denom that is
    /// to be allocated as staking rewards
    #[prost(string, tag="1")]
    pub staking_rewards: ::prost::alloc::string::String,
    /// // usage_incentives defines the proportion of the minted minted_denom that is
    /// // to be allocated to the incentives module address
    /// string usage_incentives = 2 [
    ///   (gogoproto.customtype) = "github.com/cosmos/cosmos-sdk/types.Dec",
    ///   (gogoproto.nullable) = false
    /// ];
    /// community_pool defines the proportion of the minted minted_denom that is to
    /// be allocated to the community pool
    #[prost(string, tag="3")]
    pub community_pool: ::prost::alloc::string::String,
}
/// ExponentialCalculation holds factors to calculate exponential inflation on
/// each period. Calculation reference:
/// periodProvision = exponentialDecay       *  bondingIncentive
/// f(x)            = (a * (1 - r) ^ x + c)  *  (1 + max_variance - bondedRatio *
/// (max_variance / bonding_target))
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ExponentialCalculation {
    /// initial value
    #[prost(string, tag="1")]
    pub a: ::prost::alloc::string::String,
    /// reduction factor
    #[prost(string, tag="2")]
    pub r: ::prost::alloc::string::String,
    /// long term inflation
    #[prost(string, tag="3")]
    pub c: ::prost::alloc::string::String,
    /// bonding target
    #[prost(string, tag="4")]
    pub bonding_target: ::prost::alloc::string::String,
    /// max variance
    #[prost(string, tag="5")]
    pub max_variance: ::prost::alloc::string::String,
}
/// GenesisState defines the inflation module's genesis state.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GenesisState {
    /// params defines all the paramaters of the module.
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
    /// amount of past periods, based on the epochs per period param
    #[prost(uint64, tag="2")]
    pub period: u64,
    /// inflation epoch identifier
    #[prost(string, tag="3")]
    pub epoch_identifier: ::prost::alloc::string::String,
    /// number of epochs after which inflation is recalculated
    #[prost(int64, tag="4")]
    pub epochs_per_period: i64,
    /// number of epochs that have passed while inflation is disabled
    #[prost(uint64, tag="5")]
    pub skipped_epochs: u64,
}
/// Params holds parameters for the inflation module.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Params {
    /// type of coin to mint
    #[prost(string, tag="1")]
    pub mint_denom: ::prost::alloc::string::String,
    /// variables to calculate exponential inflation
    #[prost(message, optional, tag="2")]
    pub exponential_calculation: ::core::option::Option<ExponentialCalculation>,
    /// inflation distribution of the minted denom
    #[prost(message, optional, tag="3")]
    pub inflation_distribution: ::core::option::Option<InflationDistribution>,
    /// parameter to enable inflation and halt increasing the skipped_epochs
    #[prost(bool, tag="4")]
    pub enable_inflation: bool,
}
/// QueryPeriodRequest is the request type for the Query/Period RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPeriodRequest {
}
/// QueryPeriodResponse is the response type for the Query/Period RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryPeriodResponse {
    /// period is the current minting per epoch provision value.
    #[prost(uint64, tag="1")]
    pub period: u64,
}
/// QueryEpochMintProvisionRequest is the request type for the
/// Query/EpochMintProvision RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryEpochMintProvisionRequest {
}
/// QueryEpochMintProvisionResponse is the response type for the
/// Query/EpochMintProvision RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryEpochMintProvisionResponse {
    /// epoch_mint_provision is the current minting per epoch provision value.
    #[prost(message, optional, tag="1")]
    pub epoch_mint_provision: ::core::option::Option<cosmos_sdk_proto::cosmos::base::v1beta1::DecCoin>,
}
/// QuerySkippedEpochsRequest is the request type for the Query/SkippedEpochs RPC
/// method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySkippedEpochsRequest {
}
/// QuerySkippedEpochsResponse is the response type for the Query/SkippedEpochs
/// RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QuerySkippedEpochsResponse {
    /// number of epochs that the inflation module has been disabled.
    #[prost(uint64, tag="1")]
    pub skipped_epochs: u64,
}
/// QueryCirculatingSupplyRequest is the request type for the
/// Query/CirculatingSupply RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCirculatingSupplyRequest {
}
/// QueryCirculatingSupplyResponse is the response type for the
/// Query/CirculatingSupply RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryCirculatingSupplyResponse {
    /// total amount of coins in circulation
    #[prost(message, optional, tag="1")]
    pub circulating_supply: ::core::option::Option<cosmos_sdk_proto::cosmos::base::v1beta1::DecCoin>,
}
/// QueryInflationRateRequest is the request type for the Query/InflationRate RPC
/// method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryInflationRateRequest {
}
/// QueryInflationRateResponse is the response type for the Query/InflationRate
/// RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryInflationRateResponse {
    /// rate by which the total supply increases within one period
    #[prost(string, tag="1")]
    pub inflation_rate: ::prost::alloc::string::String,
}
/// QueryParamsRequest is the request type for the Query/Params RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryParamsRequest {
}
/// QueryParamsResponse is the response type for the Query/Params RPC method.
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct QueryParamsResponse {
    /// params defines the parameters of the module.
    #[prost(message, optional, tag="1")]
    pub params: ::core::option::Option<Params>,
}
/// Generated client implementations.
pub mod query_client {
    #![allow(unused_variables, dead_code, missing_docs, clippy::let_unit_value)]
    use tonic::codegen::*;
    /// Query provides defines the gRPC querier service.
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
        /// Period retrieves current period.
        pub async fn period(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryPeriodRequest>,
        ) -> Result<tonic::Response<super::QueryPeriodResponse>, tonic::Status> {
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
                "/canto.inflation.v1.Query/Period",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// EpochMintProvision retrieves current minting epoch provision value.
        pub async fn epoch_mint_provision(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryEpochMintProvisionRequest>,
        ) -> Result<
            tonic::Response<super::QueryEpochMintProvisionResponse>,
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
                "/canto.inflation.v1.Query/EpochMintProvision",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// SkippedEpochs retrieves the total number of skipped epochs.
        pub async fn skipped_epochs(
            &mut self,
            request: impl tonic::IntoRequest<super::QuerySkippedEpochsRequest>,
        ) -> Result<tonic::Response<super::QuerySkippedEpochsResponse>, tonic::Status> {
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
                "/canto.inflation.v1.Query/SkippedEpochs",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// CirculatingSupply retrieves the total number of tokens that are in
        /// circulation (i.e. excluding unvested tokens).
        pub async fn circulating_supply(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryCirculatingSupplyRequest>,
        ) -> Result<
            tonic::Response<super::QueryCirculatingSupplyResponse>,
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
                "/canto.inflation.v1.Query/CirculatingSupply",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// InflationRate retrieves the inflation rate of the current period.
        pub async fn inflation_rate(
            &mut self,
            request: impl tonic::IntoRequest<super::QueryInflationRateRequest>,
        ) -> Result<tonic::Response<super::QueryInflationRateResponse>, tonic::Status> {
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
                "/canto.inflation.v1.Query/InflationRate",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// Params retrieves the total set of minting parameters.
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
                "/canto.inflation.v1.Query/Params",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
    }
}
