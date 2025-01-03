#!/bin/bash
TEST_TYPE=$1
set -eu

CONTAINER=$(docker ps | grep althea_test_instance | awk '{print $1}')
echo "Waiting for container to start before starting the test"
while [ -z "$CONTAINER" ];
do
    CONTAINER=$(docker ps | grep althea_test_instance | awk '{print $1}')
    sleep 1
done
echo "Container started, running tests"

# Run test entry point script
docker exec althea_test_instance /bin/sh -c "pushd /althea/ && tests/container-scripts/integration-tests.sh 1 $TEST_TYPE"