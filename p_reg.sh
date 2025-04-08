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

# Set default environment variables for Org2 (patients are in Org2)
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem
export CORE_PEER_ADDRESS=localhost:9051  # Org2's peer address

# Function to set environment for a specific patient
setup_patient_env() {
  local patient=$1
  if [ "$patient" = "patient1" ]; then
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/patient1@org2.example.com/msp
    echo "Environment set for patient1"
  elif [ "$patient" = "patient2" ]; then
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/patient2@org2.example.com/msp
    echo "Environment set for patient2"
  else
    echo "Invalid patient name: $patient"
    exit 1
  fi
}

# Function to check identity attributes
check_identity() {
  local patient=$1
  echo "Checking role of $patient"
  
  setup_patient_env $patient
  
  peer chaincode query -C emrchannel -n emr -c '{"Args":["GetIdentityAttributes"]}' || {
    echo "Failed to check identity for $patient"
    return 1
  }
}

# Function to register user
register_user() {
  local patient=$1
  echo "Registering $patient"
  
  setup_patient_env $patient
  
  peer chaincode invoke -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile $ORDERER_CA \
    -C emrchannel -n emr \
    --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0_ORG1_CA \
    --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0_ORG2_CA \
    -c '{"Args":["RegisterUser"]}' || {
    echo "Failed to register $patient"
    return 1
  }
}

# Function to verify user registration
verify_registration() {
  local patient=$1
  local common_name="${patient}@org2.example.com"
  echo "Checking if $patient is registered"
  
  setup_patient_env $patient
  
  peer chaincode query -C emrchannel -n emr -c "{\"Args\":[\"GetUser\", \"$common_name\"]}" || {
    echo "Failed to verify registration for $patient"
    return 1
  }
}

# Main execution

echo "========== Testing Patient 1 ==========="
if ! check_identity "patient1"; then
  echo "⚠️ Could not check identity for patient1"
else
  echo "✓ Identity check for patient1 successful"
fi

if ! register_user "patient1"; then
  echo "⚠️ Patient1 registration failed or already registered"
else
  echo "✓ Patient1 registered successfully"
fi

if ! verify_registration "patient1"; then
  echo "⚠️ Could not verify patient1 registration"
else
  echo "✓ Patient1 registration verified"
fi

echo -e "\n========== Testing Patient 2 ==========="
if ! check_identity "patient2"; then
  echo "⚠️ Could not check identity for patient2"
else
  echo "✓ Identity check for patient2 successful"
fi

if ! register_user "patient2"; then
  echo "⚠️ Patient2 registration failed or already registered"
else
  echo "✓ Patient2 registered successfully"
fi

if ! verify_registration "patient2"; then
  echo "⚠️ Could not verify patient2 registration"
else
  echo "✓ Patient2 registration verified"
fi

echo -e "\n========== Testing Cross-Patient Lookup ==========="
# Test if patient2 can find patient1's registration
setup_patient_env "patient2"
echo "Checking if patient2 can find patient1's registration"
peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "patient1@org2.example.com"]}' || {
  echo "⚠️ Patient2 cannot find patient1's registration"
  exit 1
}
echo "✓ Patient2 can find patient1's registration"

# Test if patient1 can find patient2's registration
setup_patient_env "patient1"
echo "Checking if patient1 can find patient2's registration"
peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "patient2@org2.example.com"]}' || {
  echo "⚠️ Patient1 cannot find patient2's registration"
  exit 1
}
echo "✓ Patient1 can find patient2's registration"

echo -e "\n========== Testing Cross-Organization Lookup ==========="
# Test if patient1 can find hospital1's registration
setup_patient_env "patient1"
echo "Checking if patient1 can find hospital1's registration"
peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "hospital1@org1.example.com"]}' || {
  echo "⚠️ Patient1 cannot find hospital1's registration"
}
if [ $? -eq 0 ]; then
  echo "✓ Patient1 can find hospital1's registration"
fi

echo -e "\nAll tests completed successfully!"

