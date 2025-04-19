use crate::args::{Args, CosmosArgs, CosmosSubcommand};

use althea_proto::cosmos_sdk_proto::ibc::core::channel::v1::query_client::QueryClient as ChannelQueryClient;
use althea_proto::cosmos_sdk_proto::ibc::core::channel::v1::{
    QueryPacketAcknowledgementRequest, QueryPacketAcknowledgementsRequest,
};
use deep_space::Contact;

pub async fn handle_cosmos_subcommand(contact: Contact, _args: &Args, cosmos_args: &CosmosArgs) {
    match &cosmos_args.subcmd {
        CosmosSubcommand::PacketAcks(packet_acks_args) => {
            // let mut transfer_grpc = TransferQueryClient::connect(contact.get_url()) .await .unwrap();
            let mut channel_grpc = ChannelQueryClient::connect(contact.get_url())
                .await
                .unwrap();

            let res = channel_grpc
                .packet_acknowledgements(QueryPacketAcknowledgementsRequest {
                    port_id: packet_acks_args.port_id.clone(),
                    channel_id: packet_acks_args.channel_id.clone(),
                    pagination: None,
                    packet_commitment_sequences: packet_acks_args.sequences.clone(),
                })
                .await;
            println!("Packet acknowledgements response: {:?}", res);
        }
        CosmosSubcommand::PacketAck(packet_ack_args) => {
            // let mut transfer_grpc = TransferQueryClient::connect(contact.get_url()) .await .unwrap();
            let mut channel_grpc = ChannelQueryClient::connect(contact.get_url())
                .await
                .unwrap();

            let res = channel_grpc
                .packet_acknowledgement(QueryPacketAcknowledgementRequest {
                    port_id: packet_ack_args.port_id.clone(),
                    channel_id: packet_ack_args.channel_id.clone(),
                    sequence: packet_ack_args.sequence,
                })
                .await;
            println!("Packet acknowledgements response: {:?}", res);
        }
    }
}
