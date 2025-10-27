#!/bin/bash
NODES=$1
TEST_TYPE=$2
set -eu

echo "Waiting for /althea/test-ready-to-run to exist before starting the test"
while [ ! -f /althea/test-ready-to-run ];
do
    sleep 1
done

# Get the directory of this script and navigate to integration_tests/test_runner
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

pushd "$PROJECT_ROOT/integration_tests/test_runner"
RUST_BACKTRACE=full TEST_TYPE=$TEST_TYPE RUST_LOG=INFO PATH=$PATH:$HOME/.cargo/bin cargo run --release --bin test-runner
