#! /bin/bash

# Find the PID of the validator1 node
VALIDATOR_1_PID=$(ps -eafww | grep /validator1 | head -n1 | awk '{print $2}')

# Run dlv and attach to validator1
/althea/tests/assets/dlv attach $VALIDATOR_1_PID --listen=:2345 --headless --api-version=2 --accept-multiclient --continue &