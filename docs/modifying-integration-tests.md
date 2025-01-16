# Modifying the integration tests

This document is a short guide on how to add new integration tests.

Before starting you should read through and run the [environment setup](/docs/developer/environment-setup.md) guide.

## Basic structure

The integration tests build and launch a docker container with a 4 validator Cosmos chain running the Althea-L1, another chain running a Cosmos Hub fork for IBC testing, test ERC20 contracts and the Althea-DEX contracts, and a 'test runner' Rust binary.

The [test runner](/integration_tests/test_runner/src/bin/main.rs) is a single rust binary that coordinates the test setup and actual test logic.

The test runner module contains some logic for running the `althea-L1/solidity` and `althea-L1/solidity-dex` contract deployers and parsing the resulting contract addresses. This is all done before we get into starting the actual tests logic.

### Chain Upgrades

It is possible to simulate the Althea-L1 chain going through an upgrade by using [run-upgrade-test.sh](/tests/run-upgrade-test.sh).

Pass a version like `v1.5.0` as a CLI argument and the test will download the given version of Althea-L1 from the [Althea-L1 releases page](https://github.com/AltheaFoundation/althea-L1/releases/) and execute several of the integration tests to generate data.

Then a software-upgrade gov proposal will execute and halt the chain, followed by the current chain code executing the same set of tests on the new binary.

## Adding tests

In order to add a new test define a new value for the `test_type` variable (specified as TEST_TYPE in the `althea-L1/tests` scripts) in the test runners' `src/bin/main.rs` file.

From there you can create a new file containing the test logic templated off of the various existing examples in `integration_tests/test_runner/src/tests/`.

Every test should perform some action and then meticulously verify that it actually took place.
It is especially important to go off the happy path and ensure correct functionality.
Tests typically `panic!()` or have an invalid assertion (e.g. `assert!(false)`) on failure, making it very clear when the chain behaves unexpectedly.
