# Getting started

Welcome! This guide covers how to get your development machine setup to contribute to Althea-L1, as well as the basics of how the code is laid out.

If you find anything in this guide that is confusing or does not work, please open an issue or [chat on Discord](https://discord.com/invite/vw8twzR).

We're always happy to help new developers get started

## Language dependencies

Althea-L1 has three major components

[The Solidity contracts deployed by the chain](https://github.com/AltheaFoundation/althea-L1/tree/main/solidity) and associated tooling. This requires NodeJS.

[The Althea-DEX contracts managed by the chain](https://github.com/AltheaFoundation/althea-dex/tree/main) and associated tooling. This requires NodeJS.

[The Go-based Althea-L1 Cosmos chain](https://github.com/AltheaFoundation/althea-L1/tree/main/). This requires Go.

[The Rust-based Integration Tests](https://github.com/AltheaFoundation/althea-L1/tree/main/integration_tests/test_runner) these require Rust.

### Installing Go

Follow the official guide [here](https://golang.org/doc/install)

Make sure that the go/bin directory is in your path by adding this to your shell profile (~/.bashrc or ~/.zprofile)

```
export PATH=$PATH:$(go env GOPATH)/bin
```

### Installing NodeJS

Follow the official guide [here](https://nodejs.org/en/)

### Installing Rust

Use the official toolchain installer [here](https://rustup.rs/)

### Alternate installation

If you are a linux user and prefer your package manager to manually installed dev dependencies you can try these.

**Fedora**
`sudo dnf install golang rust cargo npm -y`

**Ubuntu**
` audo apt-get update && sudo apt-get install golang rust cargo npm -y`

## Getting everything built

At this step download the repo

```
git clone https://github.com/AltheaFoundation/althea-L1/
```

### Solidity

Change directory into the `althea-L1/solidity` folder and run

```
# Install JavaScript dependencies
npm install

# Build the Solidity contracts used by the chain, run this after making any changes
npx hardhat compile
```

### Solidity-DEX

Change directory into the `althea-L1/solidity-dex` folder and run

```
# Install JavaScript dependencies
npm install

# Build the Solidity DEX contracts, run this after making any changes
npx hardhat compile
```

### Go

Change directory back to the root directory (`althea-L1`) and run

```
# Installing the protobuf tooling
sudo make proto-tools

# Install protobufs plugins

go install github.com/regen-network/cosmos-proto/protoc-gen-gocosmos
go get github.com/regen-network/cosmos-proto/protoc-gen-gocosmos
go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v1.16.0
go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v1.16.0

# generate new protobuf files from the definitions, this makes sure the previous instructions worked
# you will need to run this any time you change a proto file
make proto-gen

# build all code, including your newly generated go protobuf file
make

# run all the unit tests
make test
```

#### Dependency Errors

```
go: downloading github.com/regen-network/protobuf v1.3.3-alpha.regen.1
../../../go/pkg/mod/github.com/tendermint/tendermint@v0.34.13/abci/types/types.pb.go:9:2: reading github.com/regen-network/protobuf/go.mod at revision v1.3.3-alpha.regen.1: unknown revision v1.3.3-alpha.regen.1
../../../go/pkg/mod/github.com/cosmos/cosmos-sdk@v0.44.2/types/tx/service.pb.go:12:2: reading github.com/regen-network/protobuf/go.mod at revision v1.3.3-alpha.regen.1: unknown revision v1.3.3-alpha.regen.1
```

If you see dependency errors like this, clean your cache and build again

```
go clean -modcache
make
```

### Rust

#### Integration Test Runner

Change directory into the `althea-L1/integration_tests` folder and run

```

cargo build --all

```

#### ALT (Althea-L1 Tools)

The `alt` command line tool can be used to interact with the chain via the command line and supports general ERC20, ERC721, and Althea-DEX interactions. This can be very useful for situations where using wallets like MetaMask/Rabby is not possible or you want to run specific queries and transactions against known contract addresses.

`alt` is a work in progress so it is not a full-featured command line tool for Althea-L1, but much can be accomplished already using `alt` for EVM interations and the `althea` binary for Cosmos-layer interactions.

### Tips for IDEs

- We strongly recommend installing [Rust Analyzer](https://rust-analyzer.github.io/) in your IDE.
- Launch VS Code in /solidity or /solidity-dex with the solidity extension enabled to get inline typechecking of the solidity contract
- Launch VS Code in /althea-L1 with the go extension enabled to get inline typechecking of the cosmos chain

## Running the integration tests

We provide a one button integration test that deploys two full arbitrary validator Cosmos chains (one for Althea-L1 and one for IBC testing) for both development + validation.

We believe having an in-depth test environment reflecting the full deployment and production-like use of the code is essential to productive development.

Currently on every commit we send hundreds of transactions, dozens of validator set updates, and several transaction batches in our test environment.
This provides a high level of quality assurance for Althea-L1.

Because the tests build absolutely everything in this repository and deploy a host of smart contracts they do take a significant amount of time to run.

You may wish to simply push to a branch and have Github CI take care of the actual running of the tests.

### Running the integration test environment locally

The integration tests have two methods of operation, one that runs one of a pre-defined series of tests, another that produces a running local instance
of Althea-L1 for you as a developer to interact with. This is very useful for iterating quickly on changes.

```
# This builds the original docker container, only have to run this once

./tests/build-container.sh

# This starts the Cosmos chains (Althea-L1 and a Cosmos Hub fork)

./tests/start-chains.sh
```

Switch to a new terminal and run a test case using the tests/run-tests.sh script and a test name argument. All predefined tests can be found [here](https://github.com/AltheaFoundation/althea-L1/blob/main/integration_tests/test_runner/src/bin/main.rs#L118) by finding lines checking the `test_type` variable (e.g. `test_type == "LOCKUP"`).

```
./tests/run-tests.sh LOCKUP
```
### Running all up tests

All up tests are pre-defined test patterns that are run 'all up' which means including re-building all dependencies and deploying a fresh testnet for each test.
These tests _only_ work on checked in code. You must commit your latest changes to git.

A list of test patterns is defined [here](https://github.com/AltheaFoundation/althea-L1/blob/main/integration_tests/test_runner/src/main.rs#L118) (check for lines like `test_type == "TEST_NAME_HERE"`)

To run an individual test run

```
bash tests/all-up-test.sh TEST_NAME_HERE
```

These test cases potentially spawn an IBC relayer, run through the test scenario and then exit.

If you just want to have a fully functioning instance of Althea-L1 running locally you can just leave the `./tests/start-chains.sh` terminal open, and you can seed data creation using the tests.

In particular, you can run any of the `DEX` tests to connect the Althea-DEX to Cosmos governance and initialize pools, with the `DEX_ADVANCED` and `DEX_SWAP_MANY` tests being convenient ways to get several pools with decent liquidity seeded for general testing.

### Running upgrade tests

There are two ways of running upgrade tests, and they are both manual.
An upgrade test used to be run in CI, however it usually failed and even when preparing for an upgrade, it was not worth the effort.

1. `tests/manual-upgrade-test.sh`

This test sets up a testnet using the old version of the code provided as an argument to the script.
Once the chain is up and running, it expects you to perform any operations or tests manually for the test.
In another terminal window, you will be able to execute any desired tests by running `tests/run-tests.sh <TEST NAME>`. You may wish to run UPGRADE_PART_1, assuming that has been set up for your testing. UPGRADE_PART_1 should initiate a governance proposal to upgrade the chain, and the chain will likely be halted by the time it finishes.

After performing any setup, hitting Enter in the original terminal window will progress the test to phase 2, where the new binary will be running. Again, in another terminal window you may run any tests you like using `tests/run-tests.sh <TEST NAME>` and can log into the container (using `docker exec -it`) to make any post-upgrade state changes happen.

2. Mainnet state upgrade testing

If you check out the tools/manual-upgrade-tester branch, some changes to the app/app.go file will be applied. These changes make it possible to run the upgrade early on a full node which is synced to the current chain state, simulating the upgrade you have prepared (but not running any blocks after the upgrade, unfortunately).

It is recommended to perform a full backup of the node before running any such tests, by running `cp --reflink=always <Node home folder> <backup folder name>`.

On the `tools/manual-upgrade-tester` branch you will need to change the upgrade name (look for the lines with ^v^v^v for help), then run `make test`. If you run the binary which is produced using the node's home folder and provide the ALTHEA_UPGRADE_HEIGHT env var as the last block height + 1, you should see any upgrade logic logs ocurring shortly after startup (it may take 1-5 minutes to start up, so have a little patience).

It should be possible to change the state to make the chain run with a single validator and e.g. run some of the integration tests after the upgrade takes place, but the work to do this has not been done yet.

# Working inside the container

This provides access to a terminal inside the test container

```
docker exec -it althea_test_instance /bin/bash
```
