# README.md

# EMR Network

This project implements a Hyperledger Fabric network for managing Electronic Medical Records (EMRs) with distinct organizations for hospitals, doctors, and patients.

## Project Structure

- **organizations/**: Contains the configuration and certificates for each organization.
  - **fabric-ca/**: Contains the Certificate Authority (CA) configurations and certificates.
    - **hospital1/**: Configuration and certificates for the hospital organization.
    - **doctor1/**: Configuration and certificates for the doctor organization.
    - **patient/**: Configuration and certificates for the patient organization.
  - **peerOrganizations/**: Contains the Membership Service Provider (MSP) files for each organization.
    - **hospital1.example.com/**: MSP files for the hospital organization.
    - **doctor1.example.com/**: MSP files for the doctor organization.
    - **patient.example.com/**: MSP files for the patient organization.
  
- **chaincode/**: Contains the chaincode for managing EMRs.
  - **emrChaincode.go**: The main chaincode file defining the EMRChaincode struct and its methods.

- **network.sh**: A script to manage the Hyperledger Fabric network, including starting/stopping the network, creating channels, and deploying chaincode.

## Certificates

Each organization has its own Certificate Authority (CA) and associated certificates for secure communication and identity verification:

- **Hospital Organization**:
  - CA Certificate: `organizations/fabric-ca/hospital1/ca-cert.pem`
  - TLS Certificate: `organizations/fabric-ca/hospital1/tls-cert.pem`
  
- **Doctor Organization**:
  - CA Certificate: `organizations/fabric-ca/doctor1/ca-cert.pem`
  - TLS Certificate: `organizations/fabric-ca/doctor1/tls-cert.pem`
  
- **Patient Organization**:
  - CA Certificate: `organizations/fabric-ca/patient/ca-cert.pem`
  - TLS Certificate: `organizations/fabric-ca/patient/tls-cert.pem`

## Setup Instructions

1. **Install Prerequisites**: Ensure you have Docker, Docker Compose, and Go installed.
2. **Clone the Repository**: Clone this repository to your local machine.
3. **Start the Network**: Run `./network.sh up createChannel -c channel1 -s couchdb` to start the network and create a channel.
4. **Deploy Chaincode**: Use `./network.sh deployCC -ccn emr -ccp chaincode/ -ccl go -c channel1` to deploy the EMR chaincode.
5. **Register Users**: Use the Fabric CA client to register doctors, hospitals, and patients as needed.

## Usage

After setting up the network and deploying the chaincode, you can interact with the chaincode using the Fabric CLI commands to create, read, and manage EMR records.