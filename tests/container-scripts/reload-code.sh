#!/bin/bash
# Number of validators to start
NODES=$1
# what test to execute
TEST_TYPE=$2
set -eux

# Stop any currently running peggy and eth processes
pkill althea || true # allowed to fail

# Wipe filesystem changes
for i in $(seq 1 $NODES);
do
    rm -rf "/validator$i"
done


# Remove the setup complete flag file
set +e
rm -fr /althea/test-ready-to-run
set -e

cd /althea/
export PATH=$PATH:/usr/local/go/bin
make install-core
tests/container-scripts/setup-validators.sh $NODES
tests/container-scripts/setup-ibc-validators.sh $NODES
tests/container-scripts/run-testnet.sh $NODES

# Setup relayer files to avoid permissions issues later
set +e
mkdir /ibc-relayer-logs
touch /ibc-relayer-logs/hermes-logs
touch /ibc-relayer-logs/channel-creation
set -e

sleep 10

# deploy the ethereum contracts
pushd /althea/integration_tests/test_runner
DEPLOY_CONTRACTS=1 RUST_BACKTRACE=full TEST_TYPE=$TEST_TYPE NO_GAS_OPT=1 RUST_LOG="INFO" PATH=$PATH:$HOME/.cargo/bin cargo run --release --bin test-runner

# Create a setup complete flag file used by the integration tests
touch /althea/test-ready-to-run

# This keeps the script open to prevent Docker from stopping the container
# immediately if the nodes are killed by a different process
read -p "Press Return to Close..."