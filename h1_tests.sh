# 1. Start the Hyperledger Fabric network with certificate authorities for Org1, Org2, and Orderer Org:
#    - Execute: ./network.sh up -ca -c emrchannel -s couchdb
#    - This command also creates the "emrchannel" by default.

# 2. Verify the channel creation:
#    - Confirm that the emrchannel has been created and that the organizations are joined to it.

# 3. Deploy the "emrChaincode" to the channel:
#    - Run: ./network.sh deployCC -ccn emr -ccp ./chaincode -ccl go -c emrchannel
#    - Wait for successful chaincode packaging, installing, approving, and committing.

# 4. Validate the deployment:
#    - Check Docker containers with docker ps -a to verify that peers, orderer, and CA services are running.
#    - Confirm the chaincode commit by querying the channel’s chaincode definition or logs.

# 5. List all relevant identities:
#    - Identify hospital1 and hospital2 users in org1 with the "hospital" role.
#    - Identify patient1 and patient2 users in org2 with the "patient" role.
#    - Ensure that admin users for each organization and peer identities are all enrolled.

# 6. Troubleshoot common issues:
#    - If enrollment errors appear, re-run registerEnroll.sh or fix configurations in Fabric-CA.
#    - If chaincode install fails, review logs for error messages, check Go version/dependencies, and re-try the deployment steps.

# ./network.sh down && ./network.sh up createChannel -ca && ./network.sh deployCC -ccn emr -ccp ./chaincode -ccl go -c emrchannel

# Check if the network is up if not call ./network.sh up -ca -c emrchannel
if ! docker ps | grep -q "orderer.example.com"; then
  echo "Network is not up. Starting the network..."
  ./network.sh down && ./network.sh up createChannel -ca && ./network.sh deployCC -ccn emr -ccp ./chaincode -ccl go -c emrchannel
  sleep 3
fi
echo "Network is up and running."

# Check if the channel is created
if [ ! -d "channel-artifacts" ]; then
  echo "Please create the channel first. Use ./network.sh createChannel -c emrchannel"
  exit 1
fi

# Check if the chaincode is deployed
if [ ! -d "chaincode" ]; then
  echo "Please deploy the chaincode first. Use ./network.sh deployCC -ccn emr -ccp ./chaincode -ccl go -c emrchannel"
  exit 1
fi
# Check if the chaincode is installed
if [ ! -d "organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls" ]; then
  echo "Please install the chaincode first. Use ./network.sh deployCC -ccn emr -ccp ./chaincode -ccl go -c emrchannel"
  exit 1
fi

export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/hospital1@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
export ORDERER_CA=${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem

echo "Checking role of hospital1"
peer chaincode query -C emrchannel -n emr -c '{"Args":["GetIdentityAttributes"]}'
sleep 3

# Call to chaincode to register hospital1 as a user in the ledger
echo "Registering hospital1"
infoln "Invoking chaincode to register hospital1"
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C emrchannel -n emr -c '{"Args":["RegisterUser","hospital1"]}'
sleep 3
echo "Checking if hospital1 is registered"
peer chaincode query -C emrchannel -n emr -c '{"Args":["GetUser","hospital1"]}'
sleep 3
exit 0




# stop here for now
exit 0





export FABRIC_CFG_PATH=${PWD}/compose/docker/peercfg
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/hospital3@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem -C emrchannel -n emr --peerAddresses localhost:7051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt --peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt -c '{"function":"RegisterUser","Args":["hospital3"]}'








echo "Creating EMR001 with hospital1"
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile ${ORDERER_CA} -C emrchannel -n emr --peerAddresses localhost:7051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt --peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt -c '{"Args":["CreateRecord","EMR001","patient1","doctor1","hospital1","Common Cold"]}'
sleep 3

echo "Attempting query with Hospital1's MSP..."
peer chaincode query -C emrchannel -n emr -c '{"Args":["ReadRecord","EMR001"]}'\
sleep 3

echo "Attempting query with Hospital2's MSP..."
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls/ca.crt


