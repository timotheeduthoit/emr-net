#!/bin/bash

# Exit on any error
set -e

# Function to set environment variables for a specific user
set_user_env() {
    local org=$1
    local user=$2
    local port=$3
    
    export CORE_PEER_LOCALMSPID="${org}MSP"
    export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/${org,,}.example.com/tlsca/tlsca.${org,,}.example.com-cert.pem
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/${org,,}.example.com/users/${user}@${org,,}.example.com/msp
    export CORE_PEER_ADDRESS=localhost:${port}
}

# Set common environment variables
export ORDERER_CA=${PWD}/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem
export PEER0_ORG1_CA=${PWD}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export PEER0_ORG2_CA=${PWD}/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem

echo "1. Creating EMR as doctor1..."
set_user_env "Org1" "doctor1" "7051"
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA \
-C emrchannel -n emr --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0_ORG1_CA \
--peerAddresses localhost:9051 --tlsRootCertFiles $PEER0_ORG2_CA \
-c '{"Args":["CreateEMR","EMR111","not actually sick"]}'
sleep 3

echo -e "\n2. Testing access as patient2 (should fail)..."
set_user_env "Org2" "patient2" "9051"
if peer chaincode query -C emrchannel -n emr -c '{"Args":["ReadRecord","EMR111"]}' 2>/dev/null; then
    echo "Error: patient2 should not have access"
    exit 1
else
    echo "Success: patient2 access denied as expected"
fi

echo -e "\n3. Testing access as doctor2 (should fail)..."
set_user_env "Org1" "doctor2" "7051"
if peer chaincode query -C emrchannel -n emr -c '{"Args":["ReadRecord","EMR111"]}' 2>/dev/null; then
    echo "Error: doctor2 should not have access"
    exit 1
else
    echo "Success: doctor2 access denied as expected"
fi

echo -e "\n4. Sharing EMR with doctor2..."
set_user_env "Org1" "doctor1" "7051"
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA \
-C emrchannel -n emr --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0_ORG1_CA \
--peerAddresses localhost:9051 --tlsRootCertFiles $PEER0_ORG2_CA \
-c '{"Args":["ShareRecord","EMR111","doctor2@org1.example.com","doctor"]}'
sleep 3

echo -e "\n5. Verifying doctor2 access (should succeed)..."
set_user_env "Org1" "doctor2" "7051"
peer chaincode query -C emrchannel -n emr -c '{"Args":["ReadRecord","EMR111"]}'

echo -e "\n6. Sharing EMR with hospital2..."
set_user_env "Org1" "doctor1" "7051"
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA \
-C emrchannel -n emr --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0_ORG1_CA \
--peerAddresses localhost:9051 --tlsRootCertFiles $PEER0_ORG2_CA \
-c '{"Args":["ShareRecord","EMR111","hospital2@org1.example.com","hospital"]}'
sleep 3

echo -e "\n7. Verifying hospital2 access (should succeed)..."
set_user_env "Org1" "hospital2" "7051"
peer chaincode query -C emrchannel -n emr -c '{"Args":["ReadRecord","EMR111"]}'

echo -e "\nAll tests completed successfully!"
