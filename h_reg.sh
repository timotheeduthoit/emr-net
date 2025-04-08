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

  # First attempt - might fail due to transaction not yet committed
  echo "First attempt at verifying registration..."
  if peer chaincode query -C emrchannel -n emr -c "{\"Args\":[\"GetUser\", \"$common_name\"]}" 2>/dev/null; then
    echo "Registration verified on first attempt"
    return 0
  fi

  # Wait and try again
  echo "Waiting for transaction to be committed (5 seconds)..."
  sleep 5

  # Second attempt
  echo "Second attempt at verifying registration..."
  if peer chaincode query -C emrchannel -n emr -c "{\"Args\":[\"GetUser\", \"$common_name\"]}" 2>/dev/null; then
    echo "Registration verified on second attempt"
    return 0
  fi

  # Wait longer and try one more time
  echo "Waiting longer for transaction to be committed (10 seconds)..."
  sleep 10

  # Final attempt
  echo "Final attempt at verifying registration..."
  peer chaincode query -C emrchannel -n emr -c "{\"Args\":[\"GetUser\", \"$common_name\"]}" || {
    echo "Failed to verify registration for $hospital"
    return 1
  }
}

# Function to safely register and verify a hospital
safe_register_and_verify() {
  local hospital=$1

  echo -e "\n========== Testing $hospital ==========="

  # Check identity
  if ! check_identity "$hospital"; then
    echo "⚠️ Could not check identity for $hospital"
    return 1
  fi
  echo "✓ Identity check for $hospital successful"

  # Try to register
  echo "Attempting to register $hospital..."
  if ! register_user "$hospital"; then
    echo "⚠️ $hospital registration failed"
    return 1
  fi
  echo "✓ $hospital registration command completed successfully"

  # Verify registration
  if ! verify_registration "$hospital"; then
    echo "⚠️ Could not verify $hospital registration"
    return 1
  fi
  echo "✓ $hospital registration verified"
  return 0
}

# Main execution

# Make sure we're starting with Org1 environment for hospitals
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export CORE_PEER_ADDRESS=localhost:7051  # Org1's peer address

# Process hospital1
safe_register_and_verify "hospital1"
HOSPITAL1_STATUS=$?

# Process hospital2
safe_register_and_verify "hospital2"
HOSPITAL2_STATUS=$?

if [ $HOSPITAL1_STATUS -ne 0 ] || [ $HOSPITAL2_STATUS -ne 0 ]; then
  echo -e "\n⚠️ One or more hospital registrations failed. Cross-lookup tests skipped."
  exit 1
fi

echo -e "\n========== Testing Cross-Hospital Lookup ==========="
# Test if hospital2 can find hospital1's registration
setup_hospital_env "hospital2"
echo "Checking if hospital2 can find hospital1's registration"
if ! peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "hospital1@org1.example.com"]}' 2>/dev/null; then
  echo "First attempt failed, waiting 5 seconds and trying again..."
  sleep 5
  peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "hospital1@org1.example.com"]}' || {
    echo "⚠️ Hospital2 cannot find hospital1's registration"
    exit 1
  }
fi
echo "✓ Hospital2 can find hospital1's registration"

# Test if hospital1 can find hospital2's registration
setup_hospital_env "hospital1"
echo "Checking if hospital1 can find hospital2's registration"
if ! peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "hospital2@org1.example.com"]}' 2>/dev/null; then
  echo "First attempt failed, waiting 5 seconds and trying again..."
  sleep 5
  peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "hospital2@org1.example.com"]}' || {
    echo "⚠️ Hospital1 cannot find hospital2's registration"
    exit 1
  }
fi
echo "✓ Hospital1 can find hospital2's registration"

echo -e "\n========== Testing Cross-Organization Lookup ==========="
# Check if we have patient registrations in the system
setup_hospital_env "hospital1"

# Try to find a patient - switching to patient MSP path for reference
echo "Looking for patient1 registration from hospital1..."
if ! peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "patient1@org2.example.com"]}' 2>/dev/null; then
  echo "First attempt failed, waiting 5 seconds and trying again..."
  sleep 5
  if ! peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "patient1@org2.example.com"]}' 2>/dev/null; then
    echo "⚠️ Hospital1 cannot find patient1's registration (this is expected if patients aren't registered yet)"
  else
    echo "✓ Hospital1 can find patient1's registration"
  fi
else
  echo "✓ Hospital1 can find patient1's registration"
fi

echo -e "\nAll tests completed successfully!"

