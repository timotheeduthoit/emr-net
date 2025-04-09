#!/bin/bash

# Exit on first error
set -e

# Check for argument
if [ $# -ne 1 ]; then
    echo "Usage: $0 <hospital-number>"
    echo "Example: $0 5"
    exit 1
fi

# Store hospital number and construct names
HNUM=$1
HOSPITAL_NAME="hospital${HNUM}"
HOSPITAL_PASS="h${HNUM}pass"
HOSPITAL_MSP="${PWD}/organizations/peerOrganizations/org1.example.com/users/${HOSPITAL_NAME}@org1.example.com/msp"

# Import utils if needed
. scripts/utils.sh

# Set environment variables
export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/../config/
export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org1.example.com/

# Set TLS certificates
export ORDERER_CA=${PWD}/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem
export PEER0_ORG1_CA=${PWD}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export PEER0_ORG2_CA=${PWD}/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem

echo "Registering ${HOSPITAL_NAME} with Fabric CA..."
fabric-ca-client register --caname ca-org1 \
    --id.name ${HOSPITAL_NAME} \
    --id.secret ${HOSPITAL_PASS} \
    --id.type client \
    --id.affiliation org1 \
    --id.attrs "role=hospital:ecert" \
    --tls.certfiles "${PWD}/organizations/fabric-ca/org1/ca-cert.pem" || {
    echo "Failed to register ${HOSPITAL_NAME}"
    exit 1
}

echo "Enrolling ${HOSPITAL_NAME}..."
fabric-ca-client enroll \
    -u https://${HOSPITAL_NAME}:${HOSPITAL_PASS}@localhost:7054 \
    --caname ca-org1 \
    -M "${HOSPITAL_MSP}" \
    --enrollment.attrs "role,hf.Affiliation" \
    --tls.certfiles "${PWD}/organizations/fabric-ca/org1/ca-cert.pem" || {
    echo "Failed to enroll ${HOSPITAL_NAME}"
    exit 1
}

# Copy the config.yaml to the new hospital's MSP directory
cp ${PWD}/organizations/peerOrganizations/org1.example.com/msp/config.yaml ${HOSPITAL_MSP}/config.yaml

# Set environment for chaincode operations
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export CORE_PEER_MSPCONFIGPATH=${HOSPITAL_MSP}
export CORE_PEER_ADDRESS=localhost:7051

echo "Registering ${HOSPITAL_NAME} with chaincode..."
peer chaincode invoke -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile "$ORDERER_CA" \
    -C emrchannel -n emr \
    --peerAddresses localhost:7051 --tlsRootCertFiles "$PEER0_ORG1_CA" \
    --peerAddresses localhost:9051 --tlsRootCertFiles "$PEER0_ORG2_CA" \
    -c '{"Args":["RegisterUser"]}' || {
    echo "Failed to register ${HOSPITAL_NAME} with chaincode"
    exit 1
}

echo "Verifying registration..."
# Function to verify registration
verify_registration() {
    local common_name="${HOSPITAL_NAME}@org1.example.com"
    # First attempt
    if peer chaincode query -C emrchannel -n emr -c "{\"Args\":[\"GetUser\",\"$common_name\"]}" 2>/dev/null; then
        return 0
    fi
    return 1
}

# Try verification with retries
max_retries=5
retry_delay=5
retry_count=1

while [ $retry_count -le $max_retries ]; do
    echo "Verification attempt $retry_count of $max_retries..."
    if verify_registration; then
        echo "✓ ${HOSPITAL_NAME} registration verified"
        exit 0
    fi
    echo "Waiting $retry_delay seconds before next attempt..."
    sleep $retry_delay
    retry_count=$((retry_count + 1))
done

echo "❌ Failed to verify ${HOSPITAL_NAME} registration after $max_retries attempts"
exit 1

