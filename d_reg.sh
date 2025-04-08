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

# Set default environment variables for Org1 (doctors are in Org1)
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export CORE_PEER_ADDRESS=localhost:7051  # Org1's peer address

# First step: Register and enroll doctors with Fabric CA
echo -e "\n========== Step 1: Registering and enrolling doctors with Fabric CA ==========="

# Enroll admin first to get the necessary credentials
echo "Enrolling admin for Org1..."
export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org1.example.com/

# Register doctor1 with CA
echo "Registering doctor1 with Fabric CA..."
fabric-ca-client register --caname ca-org1 --id.name doctor1 --id.secret d1pass \
  --id.type client --id.affiliation org1 --id.attrs role=doctor:ecert \
  --tls.certfiles ${PWD}/organizations/fabric-ca/org1/ca-cert.pem

# Enroll doctor1
echo "Enrolling doctor1..."
fabric-ca-client enroll -u https://doctor1:d1pass@localhost:7054 --caname ca-org1 \
  -M ${PWD}/organizations/peerOrganizations/org1.example.com/users/doctor1@org1.example.com/msp \
  --enrollment.attrs role,hf.Affiliation \
  --tls.certfiles ${PWD}/organizations/fabric-ca/org1/ca-cert.pem

# Copy the config.yaml file to doctor1's MSP directory
echo "Copying config.yaml to doctor1's MSP directory"
cp ${PWD}/organizations/peerOrganizations/org1.example.com/msp/config.yaml ${PWD}/organizations/peerOrganizations/org1.example.com/users/doctor1@org1.example.com/msp/config.yaml

echo "Doctor1 registered and enrolled with Fabric CA"

# Register doctor2 with CA
echo "Registering doctor2 with Fabric CA..."
fabric-ca-client register --caname ca-org1 --id.name doctor2 --id.secret d2pass \
  --id.type client --id.affiliation org1 --id.attrs role=doctor:ecert \
  --tls.certfiles ${PWD}/organizations/fabric-ca/org1/ca-cert.pem

# Enroll doctor2
echo "Enrolling doctor2..."
fabric-ca-client enroll -u https://doctor2:d2pass@localhost:7054 --caname ca-org1 \
  -M ${PWD}/organizations/peerOrganizations/org1.example.com/users/doctor2@org1.example.com/msp \
  --enrollment.attrs role,hf.Affiliation \
  --tls.certfiles ${PWD}/organizations/fabric-ca/org1/ca-cert.pem

# Copy the config.yaml file to doctor2's MSP directory
echo "Copying config.yaml to doctor2's MSP directory"
cp ${PWD}/organizations/peerOrganizations/org1.example.com/msp/config.yaml ${PWD}/organizations/peerOrganizations/org1.example.com/users/doctor2@org1.example.com/msp/config.yaml

echo "Doctor2 registered and enrolled with Fabric CA"
# Function to set environment for a specific doctor
setup_doctor_env() {
  local doctor=$1
  if [ "$doctor" = "doctor1" ]; then
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/doctor1@org1.example.com/msp
    echo "Environment set for doctor1"
  elif [ "$doctor" = "doctor2" ]; then
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/doctor2@org1.example.com/msp
    echo "Environment set for doctor2"
  else
    echo "Invalid doctor name: $doctor"
    exit 1
  fi
}

# Function to check identity attributes
check_identity() {
  local doctor=$1
  echo "Checking role of $doctor"
  
  setup_doctor_env $doctor
  
  peer chaincode query -C emrchannel -n emr -c '{"Args":["GetIdentityAttributes"]}' || {
    echo "Failed to check identity for $doctor"
    return 1
  }
}

# Function to register user
register_user() {
  local doctor=$1
  echo "Registering $doctor"
  
  setup_doctor_env $doctor
  
  peer chaincode invoke -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile $ORDERER_CA \
    -C emrchannel -n emr \
    --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0_ORG1_CA \
    --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0_ORG2_CA \
    -c '{"Args":["RegisterUser"]}' || {
    echo "Failed to register $doctor"
    return 1
  }
}

# Function to verify user registration
verify_registration() {
  local doctor=$1
  local common_name="${doctor}@org1.example.com"
  echo "Checking if $doctor is registered"
  
  setup_doctor_env $doctor
  
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
    echo "Failed to verify registration for $doctor"
    return 1
  }
}

# Function to safely register and verify a doctor
safe_register_and_verify() {
  local doctor=$1
  
  echo -e "\n========== Testing $doctor ==========="
  
  # Check identity
  if ! check_identity "$doctor"; then
    echo "⚠️ Could not check identity for $doctor"
    return 1
  fi
  echo "✓ Identity check for $doctor successful"
  
  # Try to register
  echo "Attempting to register $doctor with chaincode..."
  if ! register_user "$doctor"; then
    echo "⚠️ $doctor registration failed"
    return 1
  fi
  echo "✓ $doctor registration command completed successfully"
  
  # Verify registration
  if ! verify_registration "$doctor"; then
    echo "⚠️ Could not verify $doctor registration"
    return 1
  fi
  echo "✓ $doctor registration verified"
  return 0
}

# Main execution
echo -e "\n========== Step 2: Registering doctors with chaincode ==========="

# Process doctor1
safe_register_and_verify "doctor1"
DOCTOR1_STATUS=$?

# Process doctor2
safe_register_and_verify "doctor2"
DOCTOR2_STATUS=$?

if [ $DOCTOR1_STATUS -ne 0 ] || [ $DOCTOR2_STATUS -ne 0 ]; then
  echo -e "\n⚠️ One or more doctor registrations failed. Cross-lookup tests skipped."
  exit 1
fi

echo -e "\n========== Testing Cross-Doctor Lookup ==========="
# Test if doctor2 can find doctor1's registration
setup_doctor_env "doctor2"
echo "Checking if doctor2 can find doctor1's registration"
if ! peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "doctor1@org1.example.com"]}' 2>/dev/null; then
  echo "First attempt failed, waiting 5 seconds and trying again..."
  sleep 5
  peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "doctor1@org1.example.com"]}' || {
    echo "⚠️ Doctor2 cannot find doctor1's registration"
    exit 1
  }
fi
echo "✓ Doctor2 can find doctor1's registration"

# Test if doctor1 can find doctor2's registration
setup_doctor_env "doctor1"
echo "Checking if doctor1 can find doctor2's registration"
if ! peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "doctor2@org1.example.com"]}' 2>/dev/null; then
  echo "First attempt failed, waiting 5 seconds and trying again..."
  sleep 5
  peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "doctor2@org1.example.com"]}' || {
    echo "⚠️ Doctor1 cannot find doctor2's registration"
    exit 1
  }
fi
echo "✓ Doctor1 can find doctor2's registration"

# Test cross-organization lookup
setup_doctor_env "doctor1"
echo "Checking if doctor1 can find hospital1's registration"
if ! peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "hospital1@org1.example.com"]}' 2>/dev/null; then
  echo "First attempt failed, waiting 5 seconds and trying again..."
  sleep 5
  if ! peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser", "hospital1@org1.example.com"]}' 2>/dev/null; then
    echo "⚠️ Doctor1 cannot find hospital1's registration"
  else
    echo "✓ Doctor1 can find hospital1's registration"
  fi
else
  echo "✓ Doctor1 can find hospital1's registration"
fi

echo -e "\nAll tests completed successfully!"

