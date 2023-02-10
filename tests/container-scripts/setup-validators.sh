#!/bin/bash
set -eux
# your gaiad binary name
BIN=althea

CHAIN_ID="althea_417834-1"

NODES=$1

STAKING_TOKEN="aalthea"
ALLOCATION="1000000000000000000000000${STAKING_TOKEN},1000000000000footoken"
 DELEGATION="500000000000000000000000${STAKING_TOKEN}"

# first we start a genesis.json with validator 1
# validator 1 will also collect the gentx's once gnerated
STARTING_VALIDATOR=1
STARTING_VALIDATOR_HOME="--home /validator$STARTING_VALIDATOR"
# todo add git hash to chain name
$BIN init $STARTING_VALIDATOR_HOME --chain-id=$CHAIN_ID validator$STARTING_VALIDATOR


## Modify generated genesis.json to our liking by editing fields using jq
## we could keep a hardcoded genesis file around but that would prevent us from
## testing the generated one with the default values provided by the module.

# add in denom metadata for both native tokens
jq '.app_state.bank.denom_metadata += [{"name": "althea", "symbol": "althea", "base": "aalthea", display: "althea", "description": "The native staking token of Althea-Chain (18 decimals)", "denom_units": [{"denom": "aalthea", "exponent": 0, "aliases": ["attoalthea", "althea-wei"]}, {"denom": "nalthea", "exponent": 9, "aliases": ["nanoalthea", "althea-gwei"]}, {"denom": "althea", "exponent": 18}]}]' /validator$STARTING_VALIDATOR/config/genesis.json > /staking-token-genesis.json
jq '.app_state.bank.denom_metadata += [{"name": "FOO", "symbol": "FOO", "base": "ufootoken", display: "footoken", "description": "A non-staking native test token (6 decimals)", "denom_units": [{"denom": "ufootoken", "exponent": 0}, {"denom": "footoken", "exponent": 6}]}]' /staking-token-genesis.json > /foo-token-genesis.json

# a 120 second voting period to allow us to pass governance proposals in the tests
jq '.app_state.gov.voting_params.voting_period = "120s"' /foo-token-genesis.json > /edited-genesis.json

# rename base denom to aalthea
sed -i 's/stake/aalthea/g' /edited-genesis.json

mv /edited-genesis.json /genesis.json


# Sets up an arbitrary number of validators on a single machine by manipulating
# the --home parameter on gaiad
for i in $(seq 1 $NODES);
do
    GAIA_HOME="--home /validator$i"
    GENTX_HOME="--home-client /validator$i"
    ARGS="$GAIA_HOME --keyring-backend test"
    KEY_ARGS="--algo secp256k1 --coin-type 118"

    $BIN keys add $ARGS $KEY_ARGS validator$i 2>> /validator-phrases

    VALIDATOR_KEY=$($BIN keys show validator$i -a $ARGS)
    # move the genesis in
    mkdir -p /validator$i/config/
    mv /genesis.json /validator$i/config/genesis.json
    $BIN add-genesis-account $ARGS $VALIDATOR_KEY $ALLOCATION
    # move the genesis back out
    mv /validator$i/config/genesis.json /genesis.json
done


for i in $(seq 1 $NODES);
do
    cp /genesis.json /validator$i/config/genesis.json
    GAIA_HOME="--home /validator$i"
    ARGS="$GAIA_HOME --keyring-backend test --chain-id=$CHAIN_ID --ip 7.7.7.$i"
    GENTX_FLAGS="--moniker validator$i --commission-rate 0.05 --commission-max-rate 0.05"
    # the /8 containing 7.7.7.7 is assigned to the DOD and never routable on the public internet
    # we're using it in private to prevent gaia from blacklisting it as unroutable
    # and allow local pex
    $BIN gentx $ARGS $GENTX_FLAGS validator$i $DELEGATION
    # obviously we don't need to copy validator1's gentx to itself
    if [ $i -gt 1 ]; then
        cp /validator$i/config/gentx/* /validator1/config/gentx/
    fi
done


$BIN collect-gentxs $STARTING_VALIDATOR_HOME
GENTXS=$(ls /validator1/config/gentx | wc -l)
cp /validator1/config/genesis.json /genesis.json
echo "Collected $GENTXS gentx"

# put the now final genesis.json into the correct folders
for i in $(seq 1 $NODES);
do
    cp /genesis.json /validator$i/config/genesis.json
done
