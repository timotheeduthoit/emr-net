#!/bin/bash
#
# Copyright IBM Corp All Rights Reserved
#
# SPDX-License-Identifier: Apache-2.0
#

# This is a collection of bash functions used by different scripts

# imports
# test network home var targets to emr-net folder
# the reason we use a var here is to accommodate scenarios
# where execution occurs from folders outside of default as $PWD, such as the emr-net/addOrg3 folder.
# For setting environment variables, simple relative paths like ".." could lead to unintended references
# due to how they interact with FABRIC_CFG_PATH. It's advised to specify paths more explicitly,
# such as using "../${PWD}", to ensure that Fabric's environment variables are pointing to the correct paths.
TEST_NETWORK_HOME=${TEST_NETWORK_HOME:-${PWD}}
. ${TEST_NETWORK_HOME}/scripts/utils.sh

export CORE_PEER_TLS_ENABLED=true
export ORDERER_CA=${TEST_NETWORK_HOME}/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem
export PEER0_ORG1_CA=${TEST_NETWORK_HOME}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export PEER0_ORG2_CA=${TEST_NETWORK_HOME}/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem
export PEER0_ORG3_CA=${TEST_NETWORK_HOME}/organizations/peerOrganizations/org3.example.com/tlsca/tlsca.org3.example.com-cert.pem

# Set environment variables for the peer org
# Added comments to align with the new roles and attributes in registerEnroll.sh
# Ensure the environment variables are consistent with the roles defined in registerEnroll.sh

# Updated setGlobals function to include role-specific attributes if needed
setGlobals() {
  local USING_ORG=""
  local USING_PEER=${2:-0} # Default to peer0 if $2 is not provided

  if [ -z "$OVERRIDE_ORG" ]; then
    USING_ORG=$1
  else
    USING_ORG="${OVERRIDE_ORG}"
  fi

  infoln "Using organization ${USING_ORG} and peer ${USING_PEER}"
  if [ $USING_ORG -eq 1 ]; then
    export CORE_PEER_LOCALMSPID=Org1MSP
    export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
    export CORE_PEER_MSPCONFIGPATH=${TEST_NETWORK_HOME}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
    if [ $USING_PEER -eq 0 ]; then
      export CORE_PEER_ADDRESS=localhost:7051 # Matches peer0.org1.example.com
    elif [ $USING_PEER -eq 1 ]; then
      export CORE_PEER_ADDRESS=localhost:8051 # Matches peer1.org1.example.com
    else
      errorln "Unknown peer for Org1"
    fi
  elif [ $USING_ORG -eq 2 ]; then
    export CORE_PEER_LOCALMSPID=Org2MSP
    export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG2_CA
    export CORE_PEER_MSPCONFIGPATH=${TEST_NETWORK_HOME}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
    if [ $USING_PEER -eq 0 ]; then
      export CORE_PEER_ADDRESS=localhost:9051 # Matches peer0.org2.example.com
    elif [ $USING_PEER -eq 1 ]; then
      export CORE_PEER_ADDRESS=localhost:10051 # Matches peer1.org2.example.com
    else
      errorln "Unknown peer for Org2"
    fi
  elif [ $USING_ORG -eq 3 ]; then
    export CORE_PEER_LOCALMSPID=Org3MSP
    export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG3_CA
    export CORE_PEER_MSPCONFIGPATH=${TEST_NETWORK_HOME}/organizations/peerOrganizations/org3.example.com/users/Admin@org3.example.com/msp
    if [ $USING_PEER -eq 0 ]; then
      export CORE_PEER_ADDRESS=localhost:11051 # Matches peer0.org3.example.com
    elif [ $USING_PEER -eq 1 ]; then
      export CORE_PEER_ADDRESS=localhost:12051 # Matches peer1.org3.example.com
    else
      errorln "Unknown peer for Org3"
    fi
  else
    errorln "ORG Unknown"
  fi

  # Added VERBOSE check for debugging
  if [ "$VERBOSE" = "true" ]; then
    env | grep CORE
  fi
}

# parsePeerConnectionParameters $@
# Helper function that sets the peer connection parameters for a chaincode
# operation
parsePeerConnectionParameters() {
  PEER_CONN_PARMS=()
  PEERS=""
  while [ "$#" -gt 0 ]; do
    setGlobals $1
    PEER="peer0.org$1"
    ## Set peer addresses
    if [ -z "$PEERS" ]
    then
    PEERS="$PEER"
    else
    PEERS="$PEERS $PEER"
    fi
    PEER_CONN_PARMS=("${PEER_CONN_PARMS[@]}" --peerAddresses $CORE_PEER_ADDRESS)
    ## Set path to TLS certificate
    CA=PEER0_ORG$1_CA
    TLSINFO=(--tlsRootCertFiles "${!CA}")
    PEER_CONN_PARMS=("${PEER_CONN_PARMS[@]}" "${TLSINFO[@]}")
    # shift by one to get to the next organization
    shift
  done
}

verifyResult() {
  if [ $1 -ne 0 ]; then
    fatalln "$2"
  fi
}
