use crate::args::{Args, CosmosArgs, CosmosSubcommand};

use althea_proto::cosmos_sdk_proto::cosmos::bank::v1beta1::{
    QuerySpendableBalancesRequest, query_client::QueryClient as BankQueryClient,
};
use deep_space::Contact;

pub async fn handle_cosmos_subcommand(_contact: &Contact, args: &Args, cosmos_args: &CosmosArgs) {
    match &cosmos_args.subcmd {
        CosmosSubcommand::SpendableBalance(cmd_args) => {
            let mut grpc = BankQueryClient::connect(args.cosmos_grpc.clone())
                .await
                .expect("Unable to connect to bank query client");
            let balance = grpc.spendable_balances(QuerySpendableBalancesRequest{
                address: cmd_args.address.to_string(),
                pagination: None,
            }).await.expect("Unable to get spendable balance").into_inner();
            println!(
                "Spendable balance for {}: {:?}",
                cmd_args.address, balance
            );
        },
    }
}
