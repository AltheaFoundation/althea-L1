use crate::utils::*;
use althea_proto::cosmos_sdk_proto::ibc::core::channel::v1::query_client::QueryClient as IbcChannelQueryClient;
use althea_proto::cosmos_sdk_proto::ibc::core::channel::v1::IdentifiedChannel;
use althea_proto::cosmos_sdk_proto::ibc::core::channel::v1::{
    QueryChannelClientStateRequest, QueryChannelsRequest,
};
use althea_proto::cosmos_sdk_proto::ibc::lightclients::tendermint::v1::ClientState;
use deep_space::error::CosmosGrpcError;
use deep_space::utils::decode_any;
use std::time::Duration;
use std::time::Instant;
use tokio::time::sleep as delay_for;
use tonic::transport::Channel;

// Retrieves the channel connecting the chain behind `ibc_channel_qc` and the chain with id `foreign_chain_id`
// Retries up to `timeout` (or OPERATION_TIMEOUT if None), checking each channel's client state to find the foreign chain's id
pub async fn get_channel(
    ibc_channel_qc: IbcChannelQueryClient<Channel>, // The Src chain's IbcChannelQueryClient
    foreign_chain_id: String,                       // The chain-id of the Dst chain
    timeout: Option<Duration>,
) -> Result<IdentifiedChannel, CosmosGrpcError> {
    let mut ibc_channel_qc = ibc_channel_qc;
    let timeout = match timeout {
        Some(t) => t,
        None => OPERATION_TIMEOUT,
    };

    let start = Instant::now();
    while Instant::now() - start < timeout {
        let channels = ibc_channel_qc
            .channels(QueryChannelsRequest { pagination: None })
            .await;
        if channels.is_err() {
            delay_for(Duration::from_secs(5)).await;
            continue;
        }
        let channels = channels.unwrap().into_inner().channels;
        for channel in channels {
            // Make an IBC Channel ClientState request with port=transfer channel=channel.channel_id
            let client_state_res = ibc_channel_qc
                .channel_client_state(QueryChannelClientStateRequest {
                    port_id: "transfer".to_string(),
                    channel_id: channel.channel_id.clone(),
                })
                .await;
            if client_state_res.is_err() {
                continue;
            }

            let client_state = client_state_res
                .unwrap()
                .into_inner()
                .identified_client_state;
            if client_state.is_none() || client_state.clone().unwrap().client_state.is_none() {
                error!(
                    "Got None response for {}/transfer client state!",
                    channel.channel_id.clone()
                );
                continue;
            }
            let client_state_any = client_state.unwrap().client_state.unwrap();

            // Check to see if this client state contains foreign_chain_id (e.g. "cavity-1")
            let client_state = decode_any::<ClientState>(client_state_any).unwrap();
            if client_state.chain_id == foreign_chain_id {
                info!("Discovered IBC Channel: {:?}", channel);
                return Ok(channel);
            }
        }
    }
    Err(CosmosGrpcError::BadResponse("No such channel".to_string()))
}

// Retrieves just the ID of the channel connecting the chain behind `ibc_channel_qc` and the chain with id `foreign_chain_id`
// Retries up to `timeout` (or OPERATION_TIMEOUT if None), checking each channel's client state to find the foreign chain's id
pub async fn get_channel_id(
    ibc_channel_qc: IbcChannelQueryClient<Channel>, // The Src chain's IbcChannelQueryClient
    foreign_chain_id: String,                       // The chain-id of the Dst chain
    timeout: Option<Duration>,
) -> Result<String, CosmosGrpcError> {
    Ok(get_channel(ibc_channel_qc, foreign_chain_id, timeout)
        .await?
        .channel_id)
}
