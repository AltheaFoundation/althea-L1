#!/bin/bash

rm /althea/test-ready-to-run

set -eux
# Number of validators to start
NODES=$1
# old binary version to run
OLD_VERSION=$2

chmod -R 777 /root/

export OLD_BINARY_LOCATION=/oldalthea

# Clone and build the old version
echo "Cloning old althea-L1 version from $OLD_VERSION..."
git clone https://github.com/AltheaFoundation/althea-L1.git /althea-old
pushd /althea-old
git checkout $OLD_VERSION

# Initialize submodules
echo "Initializing submodules..."
git submodule update --init --recursive

# Prepare the contracts for later deployment
pushd solidity
HUSKY_SKIP_INSTALL=1 npm install
npm run typechain
popd
pushd solidity-dex
npm install
npx hardhat compile
popd

# Build the old binary
echo "Building old althea binary..."
export PATH=$PATH:/usr/local/go/bin
make
make install

# Copy the binary to OLD_BINARY_LOCATION
mkdir -p $(dirname $OLD_BINARY_LOCATION)
echo "Built althea binary location is $(which althea)"
cp $(which althea) $OLD_BINARY_LOCATION
chmod +x $OLD_BINARY_LOCATION


# Start the test using the old tests
pushd /althea-old/
tests/container-scripts/setup-validators.sh $NODES
tests/container-scripts/setup-ibc-validators.sh $NODES

# Run the old binary
tests/container-scripts/run-testnet.sh $NODES

# deploy the ethereum contracts
pushd integration_tests/test_runner
DEPLOY_CONTRACTS=1 RUST_BACKTRACE=full NO_GAS_OPT=1 RUST_LOG="INFO" PATH=$PATH:$HOME/.cargo/bin cargo run --release --bin test-runner
popd

popd  # exit /althea-old/

touch /althea/test-ready-to-run

# This allows the tester to run the first part of the test
# immediately if the nodes are killed by a different process

read -p "Old binary is running. In a separate terminal, use /althea-old/tests/run-tests.sh to run older versions of the tests. Use /althea/tests/run-tests.sh to run the current version of the tests. Hit Enter in this terminal to continue to part 2..."

pushd /althea/

# Build the new binary
make
make install

rm -fr $OLD_BINARY_LOCATION
unset OLD_BINARY_LOCATION
# Run the new binary
pkill oldalthea || true # allowed to fail
pkill gaiad || true # allowed to fail
tests/container-scripts/run-testnet.sh $NODES

# This allows the tester to run the first part of the test
# immediately if the nodes are killed by a different process

read -p "New binary is running, use /althea/tests/run-tests.sh to run tests on the upgraded chain! Hit Enter to close the container and end all tests..."