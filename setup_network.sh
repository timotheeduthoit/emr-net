#!/bin/bash

# Bring up the network and create channel
./network.sh up createChannel -c emrchannel

# Deploy the chaincode
./network.sh deployCC -ccn emrcc -ccp chaincode/ -ccl go -c emrchannel

