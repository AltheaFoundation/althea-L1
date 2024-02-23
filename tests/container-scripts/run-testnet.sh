#!/bin/bash
set -eux
# your gaiad binary name
BIN=althea

NODES=$1

for i in $(seq 1 $NODES);
do
    # add this ip for loopback dialing
    ip addr add 7.7.7.$i/32 dev eth0 || true # allowed to fail

    GAIA_HOME="--home /validator$i"
    # this implicitly caps us at ~6000 nodes for this sim
    # note that we start on 26656 the idea here is that the first
    # node (node 1) is at the expected contact address from the gentx
    # faciliating automated peer exchange
    if [[ "$i" -eq 1 ]]; then
        # node one gets localhost so we can easily shunt these ports
        # to the docker host
        RPC_ADDRESS="--rpc.laddr tcp://0.0.0.0:26657"
        GRPC_ADDRESS="--grpc.address 0.0.0.0:9090"
        GRPC_WEB_ADDRESS="--grpc-web.address 0.0.0.0:9092"
        ETH_RPC_ADDRESS="--json-rpc.address 127.0.0.1:8545"
        ETH_WS_RPC_ADDRESS="--json-rpc.ws-address 127.0.0.1:8546"
        sed -i 's/enable-unsafe-cors = false/enable-unsafe-cors = true/g' /validator$i/config/app.toml
        sed -i 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' /validator$i/config/app.toml
        sed -i 's/enable = false/enable = true/g' /validator$i/config/app.toml #enables more than we want, but will work for now
    else
        # move these to another port and address, not becuase they will
        # be used there, but instead to prevent them from causing problems
        # you also can't duplicate the port selection against localhost
        # for reasons that are not clear to me right now.
        RPC_ADDRESS="--rpc.laddr tcp://7.7.7.$i:26658"
        GRPC_ADDRESS="--grpc.address 7.7.7.$i:9091"
        GRPC_WEB_ADDRESS="--grpc-web.address 7.7.7.$i:9093"
        ETH_RPC_ADDRESS="--json-rpc.address 7.7.7.$i:8545"
        ETH_WS_RPC_ADDRESS="--json-rpc.ws-address 7.7.7.$i:8546"
    fi
    LISTEN_ADDRESS="--address tcp://7.7.7.$i:26655"
    P2P_ADDRESS="--p2p.laddr tcp://7.7.7.$i:26656"
    LOG_LEVEL="--log_level info"
    INVARIANTS_CHECK="--inv-check-period 1"
    ARGS="$GAIA_HOME $LISTEN_ADDRESS $RPC_ADDRESS $GRPC_ADDRESS $GRPC_WEB_ADDRESS $ETH_RPC_ADDRESS $ETH_WS_RPC_ADDRESS $INVARIANTS_CHECK $LOG_LEVEL $P2P_ADDRESS"
    $BIN $ARGS start &> /validator$i/logs &
done

# Setup the IBC test chain (chain id ibc-test-1) using gaiad as the binary
# Creates the same number of validators as the althea chain above, with their home directories at /ibc-validator#
BIN=gaiad
for i in $(seq 1 $NODES);
do
    ip addr add 7.7.8.$i/32 dev eth0 || true # allowed to fail

    GAIA_HOME="--home /ibc-validator$i"
    if [[ "$i" -eq 1 ]]; then
        # node one gets localhost so we can easily shunt these ports
        # to the docker host
        RPC_ADDRESS="--rpc.laddr tcp://0.0.0.0:27657"
        GRPC_ADDRESS="--grpc.address 0.0.0.0:9190"
        # Must remap the grpc-web address because it conflicts with what we want to use
        GRPC_WEB_ADDRESS="--grpc-web.address 0.0.0.0:9192"
    else
        RPC_ADDRESS="--rpc.laddr tcp://7.7.8.$i:26658"
        GRPC_ADDRESS="--grpc.address 7.7.8.$i:9091"
        # Must remap the grpc-web address because it conflicts with what we want to use
        GRPC_WEB_ADDRESS="--grpc-web.address 7.7.8.$i:9093"
    fi
    LISTEN_ADDRESS="--address tcp://7.7.8.$i:26655"
    P2P_ADDRESS="--p2p.laddr tcp://7.7.8.$i:26656"
    LOG_LEVEL="--log_level info"
    ARGS="$GAIA_HOME $LISTEN_ADDRESS $RPC_ADDRESS $GRPC_ADDRESS $GRPC_WEB_ADDRESS $LOG_LEVEL $P2P_ADDRESS"
    $BIN $ARGS start &> /ibc-validator$i/logs &
done