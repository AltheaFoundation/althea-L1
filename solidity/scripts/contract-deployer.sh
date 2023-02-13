#!/bin/bash
npx ts-node \
contract-deployer.ts \
--cosmos-node="http://localhost:26657" \
--eth-node="http://localhost:8545" \
--eth-privkey="0x34d97aaf58b1a81d3ed3068a870d8093c6341cf5d1ef7e6efa03fe7f7fc2c3a8" \
--contract=Gravity.json \
--contractERC721=GravityERC721.json \
--test-mode=true
