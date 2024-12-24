#!/bin/bash

rm /althea/test-ready-to-run
set -eux
# Number of validators to start
NODES=$1
# old binary version to run
OLD_VERSION=$2

echo "Downloading old althea version at https://github.com/AltheaFoundation/althea-L1/releases/download/${OLD_VERSION}/althea-linux-amd64"
wget https://github.com/AltheaFoundation/althea-L1/releases/download/${OLD_VERSION}/althea-linux-amd64
mv althea-linux-amd64 oldalthea
# Make old althea executable
chmod +x oldalthea

export OLD_BINARY_LOCATION=/oldalthea

# Prepare the contracts for later deployment
pushd /althea/solidity/
HUSKY_SKIP_INSTALL=1 npm install
npm run typechain
popd

pushd /althea/solidity-dex/
npx hardhat compile
popd

pushd /althea/
export PATH=$PATH:/usr/local/go/bin
make
make install
tests/container-scripts/setup-validators.sh $NODES
tests/container-scripts/setup-ibc-validators.sh $NODES

# Run the old binary
tests/container-scripts/run-testnet.sh $NODES
popd

# deploy the ethereum contracts
pushd /althea/integration_tests/test_runner
DEPLOY_CONTRACTS=1 RUST_BACKTRACE=full NO_GAS_OPT=1 RUST_LOG="INFO" PATH=$PATH:$HOME/.cargo/bin cargo run --release --bin test-runner
popd

touch /althea/test-ready-to-run

# Run the pre-upgrade tests
pushd /althea/
tests/container-scripts/integration-tests.sh $NODES UPGRADE_PART_1

unset OLD_BINARY_LOCATION
# Run the new binary
pkill oldalthea || true # allowed to fail
tests/container-scripts/run-testnet.sh $NODES

# Run the post-upgrade test
tests/container-scripts/integration-tests.sh $NODES UPGRADE_PART_2
popd
