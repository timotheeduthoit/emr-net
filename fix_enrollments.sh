#!/bin/bash
set -e

# Exit on errors
trap 'echo "Error: There was an error executing the script."; exit 1' ERR

echo "Fix enrollments script started"

# Function to fix Org1 enrollments (hospitals)
function fixOrg1Enrollments() {
  echo "Fixing Org1 enrollments (hospitals)"

  # Set the Fabric CA client home directory for Org1
  export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org1.example.com/

  # Register new hospital identities with proper attributes
  echo "Registering hospital1_fixed with proper attributes"
  ../bin/fabric-ca-client register --caname ca-org1 --id.name hospital1_fixed --id.secret h1pass --id.type client --id.attrs "role=hospital:ecert" --tls.certfiles "${PWD}/organizations/fabric-ca/org1/ca-cert.pem" || true

  echo "Enrolling hospital1_fixed with correct attributes"
  # Set up MSP directory for the new identity
  rm -rf "${PWD}/organizations/peerOrganizations/org1.example.com/users/Hospital1_Fixed@org1.example.com/msp"
  mkdir -p "${PWD}/organizations/peerOrganizations/org1.example.com/users/Hospital1_Fixed@org1.example.com/msp"
  
  # Enroll with attribute 
  ../bin/fabric-ca-client enroll -u https://hospital1_fixed:h1pass@localhost:7054 --caname ca-org1 -M "${PWD}/organizations/peerOrganizations/org1.example.com/users/Hospital1_Fixed@org1.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/org1/ca-cert.pem"

  cp "${PWD}/organizations/peerOrganizations/org1.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/org1.example.com/users/Hospital1_Fixed@org1.example.com/msp/config.yaml"

  echo "Creating symbolic link for backward compatibility"
  ln -sf "${PWD}/organizations/peerOrganizations/org1.example.com/users/Hospital1_Fixed@org1.example.com/msp" "${PWD}/organizations/peerOrganizations/org1.example.com/users/Hospital1@org1.example.com/msp" || true

  # Register hospital2_fixed
  echo "Registering hospital2_fixed with proper attributes"
  ../bin/fabric-ca-client register --caname ca-org1 --id.name hospital2_fixed --id.secret h2pass --id.type client --id.attrs "role=hospital:ecert" --tls.certfiles "${PWD}/organizations/fabric-ca/org1/ca-cert.pem" || true

  echo "Enrolling hospital2_fixed with correct attributes"
  # Set up MSP directory for the new identity
  rm -rf "${PWD}/organizations/peerOrganizations/org1.example.com/users/Hospital2_Fixed@org1.example.com/msp"
  mkdir -p "${PWD}/organizations/peerOrganizations/org1.example.com/users/Hospital2_Fixed@org1.example.com/msp"
  
  # Enroll with attribute
  ../bin/fabric-ca-client enroll -u https://hospital2_fixed:h2pass@localhost:7054 --caname ca-org1 -M "${PWD}/organizations/peerOrganizations/org1.example.com/users/Hospital2_Fixed@org1.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/org1/ca-cert.pem"

  cp "${PWD}/organizations/peerOrganizations/org1.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/org1.example.com/users/Hospital2_Fixed@org1.example.com/msp/config.yaml"

  echo "Creating symbolic link for backward compatibility"
  ln -sf "${PWD}/organizations/peerOrganizations/org1.example.com/users/Hospital2_Fixed@org1.example.com/msp" "${PWD}/organizations/peerOrganizations/org1.example.com/users/Hospital2@org1.example.com/msp" || true
}

# Function to fix Org2 enrollments (patients)
function fixOrg2Enrollments() {
  echo "Fixing Org2 enrollments (patients)"

  # Set the Fabric CA client home directory for Org2
  export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org2.example.com/

  # Register new patient identities with proper attributes
  echo "Registering patient1_fixed with proper attributes"
  ../bin/fabric-ca-client register --caname ca-org2 --id.name patient1_fixed --id.secret p1pass --id.type client --id.attrs "role=patient:ecert" --tls.certfiles "${PWD}/organizations/fabric-ca/org2/ca-cert.pem" || true

  echo "Enrolling patient1_fixed with correct attributes"
  # Set up MSP directory for the new identity
  rm -rf "${PWD}/organizations/peerOrganizations/org2.example.com/users/Patient1_Fixed@org2.example.com/msp"
  mkdir -p "${PWD}/organizations/peerOrganizations/org2.example.com/users/Patient1_Fixed@org2.example.com/msp"
  
  # Enroll with attribute
  ../bin/fabric-ca-client enroll -u https://patient1_fixed:p1pass@localhost:
}

# Function to fix peer TLS issues and restart peers
function fixPeersAndRestart() {
  echo "Fixing peer TLS issues and restarting peers"
  
  # Stop all peer containers
  echo "Stopping peer containers..."
  docker stop peer0.org1.example.com peer0.org2.example.com peer1.org1.example.com peer1.org2.example.com 2>/dev/null || true
  docker rm peer0.org1.example.com peer0.org2.example.com peer1.org1.example.com peer1.org2.example.com 2>/dev/null || true

  # Verify TLS cert directories exist for peers and re-create if needed
  # For peer0.org2
  echo "Regenerating TLS certificates for Org2 peers"
  export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org2.example.com/
  
  # Fix peer0.org2.example.com TLS
  mkdir -p "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls"
  # Clean existing TLS directory
  rm -rf "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls"
  mkdir -p "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls"
  
  # Enroll with TLS profile
  ../bin/fabric-ca-client enroll -u https://peer0:peer0pw@localhost:8054 --caname ca-org2 -M "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls" --enrollment.profile tls --csr.hosts peer0.org2.example.com --csr.hosts localhost --tls.certfiles "${PWD}/organizations/fabric-ca/org2/ca-cert.pem"

  cp "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/tlscacerts/"* "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt"
  cp "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/signcerts/"* "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/server.crt"
  cp "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/keystore/"* "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/server.key"

  # Fix peer1.org2.example.com TLS
  mkdir -p "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/tls"
  # Clean existing TLS directory
  rm -rf "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/tls"
  mkdir -p "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/tls"
  
  # Enroll with TLS profile
  ../bin/fabric-ca-client enroll -u https://peer1:peer1pw@localhost:8054 --caname ca-org2 -M "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/tls" --enrollment.profile tls --csr.hosts peer1.org2.example.com --csr.hosts localhost --tls.certfiles "${PWD}/organizations/fabric-ca/org2/ca-cert.pem"

  cp "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/tls/tlscacerts/"* "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/tls/ca.crt"
  cp "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/tls/signcerts/"* "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/tls/server.crt"
  cp "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/tls/keystore/"* "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/tls/server.key"

  # Fix peer1.org1.example.com TLS
  echo "Regenerating TLS certificates for Org1 peers"
  export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org1.example.com/
  
  # We're only regenerating peer1 since peer0 seems to be working
  mkdir -p "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls"
  # Clean existing TLS directory
  rm -rf "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls"
  mkdir -p "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls"
  
  # Enroll with TLS profile
  ../bin/fabric-ca-client enroll -u https://peer1:peer1pw@localhost:7054 --caname ca-org1 -M "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls" --enrollment.profile tls --csr.hosts peer1.org1.example.com --csr.hosts localhost --tls.certfiles "${PWD}/organizations/fabric-ca/org1/ca-cert.pem"

  cp "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls/tlscacerts/"* "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls/ca.crt"
  cp "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls/signcerts/"* "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls/server.crt"
  cp "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls/keystore/"* "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls/server.key"

  # Restart the network using docker-compose
  echo "Restarting the network..."
  docker-compose -f compose/docker-compose.yaml -f compose/docker-compose-couch.yaml up -d

  # Wait for peers to start
  echo "Waiting for peers to start..."
  sleep 15
  
  # Check the status of containers
  echo "Checking container status:"
  docker ps -a | grep 'peer\|orderer'
}

# Main execution
echo "Starting enrollment fixes"

# Make sure the peer directories have the right permissions
chmod -R 755 ${PWD}/organizations/

# Call the functions to fix enrollments and restart peers
fixOrg1Enrollments
fixOrg2Enrollments
fixPeersAndRestart

echo "Fix enrollments script completed"
echo "You can now proceed with channel creation and chaincode deployment"
