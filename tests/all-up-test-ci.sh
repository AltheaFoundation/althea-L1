#!/bin/bash
# This script is used in Github actions CI to prep and run a testnet environment
TEST_TYPE=$1
set -eux
NODES=4

sudo apt-get update
sudo apt-get install -y git make gcc g++ iproute2 iputils-ping procps vim tmux net-tools htop tar jq npm libssl-dev perl rustc cargo wget

# Setup Althea L1 binary
GOPROXY=https://proxy.golang.org make
make install
sudo cp ~/go/bin/althea /usr/bin/althea

# Download the althea gaia fork as a IBC test chain
sudo wget https://github.com/althea-net/ibc-test-chain/releases/download/v9.1.2/gaiad-v9.1.2-linux-amd64 -O /usr/bin/gaiad

# Setup Hermes for IBC connections between chains
pushd /tmp/
wget https://github.com/informalsystems/ibc-rs/releases/download/v1.7.0/hermes-v1.7.0-x86_64-unknown-linux-gnu.tar.gz
tar -xvf hermes-v1.7.0-x86_64-unknown-linux-gnu.tar.gz
sudo mv hermes /usr/bin/
popd

# make log dirs
sudo mkdir /ibc-relayer-logs
sudo touch /ibc-relayer-logs/hermes-logs
sudo touch /ibc-relayer-logs/channel-creation

# Compile the solidity contracts
pushd solidity/
HUSKY_SKIP_INSTALL=1 npm install
npm run typechain
ls -lah
ls -lah artifacts/
ls -lah artifacts/contracts/
pwd
popd

git clone https://github.com/AltheaFoundation/althea-dex.git solidity-dex/
pushd solidity-dex/
HUSKY_SKIP_INSTALL=1 npm install
npx hardhat compile
ls -lah
ls -lah misc/scripts/
pwd
popd

sudo bash tests/container-scripts/setup-validators.sh $NODES
sudo bash tests/container-scripts/setup-ibc-validators.sh $NODES
sudo bash tests/container-scripts/run-testnet.sh $NODES $TEST_TYPE

# deploy the ethereum contracts
pushd integration_tests/test_runner
DEPLOY_CONTRACTS=1 RUST_BACKTRACE=full TEST_TYPE=$TEST_TYPE NO_GAS_OPT=1 RUST_LOG="INFO" PATH=$PATH:$HOME/.cargo/bin cargo run --release --bin test-runner
popd

echo "Running ibc relayer in the background, directing output to /ibc-relayer-logs"

pushd integration_tests
RUST_BACKTRACE=full TEST_TYPE=$TEST_TYPE RUST_LOG=INFO PATH=$PATH:$HOME/.cargo/bin cargo run --release --bin test-runner
