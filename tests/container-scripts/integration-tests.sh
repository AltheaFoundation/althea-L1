#!/bin/bash
NODES=$1
TEST_TYPE=$2
set -eu

echo "Waiting for /althea/test-ready-to-run to exist before starting the test"
while [ ! -f /althea/test-ready-to-run ];
do
    sleep 1
done

pushd /althea/integration_tests/test_runner
RUST_BACKTRACE=full TEST_TYPE=$TEST_TYPE RUST_LOG=INFO PATH=$PATH:$HOME/.cargo/bin cargo run --release --bin test-runner
