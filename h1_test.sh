#!/bin/bash

# Exit on first error
set -e

# Import utils
. scripts/utils.sh

export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/../config/

export ORDERER_CA=${PWD}/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem
export PEER0_ORG1_CA=${PWD}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export PEER0_ORG2_CA=${PWD}/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem

# Set default environment variables for Org1
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export CORE_PEER_ADDRESS=localhost:7051

# Function to set environment for a specific hospital
setup_hospital_env() {
  local hospital=$1
  if [ "$hospital" = "hospital1" ]; then
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/hospital1@org1.example.com/msp
    echo "Environment set for hospital1"
  elif [ "$hospital" = "hospital2" ]; then
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/hospital2@org1.example.com/msp
    echo "Environment set for hospital2"
  else
    echo "Invalid hospital name: $hospital"
    exit 1
  fi
}

# Function to check identity attributes
check_identity() {
  local hospital=$1
  echo "Checking role of $hospital"
  
  setup_hospital_env $hospital
  
  peer chaincode query -C emrchannel -n emr -c '{"Args":["GetIdentityAttributes"]}' || {
    echo "Failed to check identity for $hospital"
    return 1
  }
}

# Function to register user
register_user() {
  local hospital=$1
  echo "Registering $hospital"
  
  setup_hospital_env $hospital
  
  peer chaincode invoke -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile $ORDERER_CA \
    -C emrchannel -n emr \
    --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0_ORG1_CA \
    --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0_ORG2_CA \
    -c '{"Args":["RegisterUser"]}' || {
    echo "Failed to register $hospital"
    return 1
  }
}

# Function to verify user registration
verify_registration() {
  local hospital=$1
  local common_name="${hospital}@org1.example.com"
  echo "Checking if $hospital is registered"
  
  setup_hospital_env $hospital
  
  peer chaincode query -C emrchannel -n emr -c "{\"Args\":[\"GetUser\", \"$common_name\"]}" || {
    echo "Failed to verify registration for $hospital"
    return 1
  }
}

# Main execution

echo "========== Testing Hospital 1 ==========="
if ! check_identity "hospital1"; then
  echo "⚠️ Could not check identity for hospital1"
else
  echo "✓ Identity check for hospital1 successful"
fi

if ! register_user "hospital1"; then
  echo "⚠️ Hospital1 registration failed or already registered"
else
  echo "✓ Hospital1 registered successfully"
fi

if ! verify_registration "hospital1"; then
  echo "⚠️ Could not verify hospital1 registration"
else
  echo "✓ Hospital1 registration verified"
fi

echo -e "\n========== Testing Hospital 2 ==========="
if ! check_identity "hospital2"; then
  echo "⚠️ Could not check identity for hospital2"
else
  echo "✓ Identity check for hospital2 successful"
fi

if ! register_user "hospital2"; then
  echo "⚠️ Hospital2 registration failed or already registered"
else
  echo "✓ Hospital2 registered successfully"
fi

if ! verify_registration "hospital2"; then
  echo "⚠️ Could not verify hospital2 registration"
else
  echo "✓ Hospital2 registration verified"
fi

echo -e "\n========== Testing Cross-Hospital Lookup ==========="
# Test if hospital2 can find hospital1's registration
setup_hospital_env "hospital2"
echo "Checking if hospital2 can find hospital1's registration"
peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "hospital1@org1.example.com"]}' || {
  echo "⚠️ Hospital2 cannot find hospital1's registration"
  exit 1
}
echo "✓ Hospital2 can find hospital1's registration"

echo -e "\nAll tests completed successfully!"

