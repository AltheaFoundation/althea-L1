#!/bin/bash
# the script run inside the container for all-up-test.sh
NODES=$1
TEST_TYPE=$2
set -eux

bash /althea/tests/container-scripts/setup-validators.sh $NODES
bash /althea/tests/container-scripts/setup-ibc-validators.sh $NODES
bash /althea/tests/container-scripts/run-testnet.sh $NODES $TEST_TYPE &

sleep 30

# deploy the ethereum contracts
pushd /althea/integration_tests/test_runner
DEPLOY_CONTRACTS=1 RUST_BACKTRACE=full TEST_TYPE=$TEST_TYPE NO_GAS_OPT=1 RUST_LOG="INFO" PATH=$PATH:$HOME/.cargo/bin cargo run --release --bin test-runner

bash /althea/tests/container-scripts/integration-tests.sh $NODES $TEST_TYPE