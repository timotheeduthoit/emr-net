# EMR Network Latency Testing

This document explains the latency testing script for the Hyperledger Fabric EMR Network.

## About the Network

The EMR (Electronic Medical Record) network is built on Hyperledger Fabric with the following characteristics:

- **Multiple Organizations**: Hospital organization (Org1) and Patient organization (Org2)
- **User Types**: Hospitals, patients, and doctors with defined roles and permissions
- **Core Operations**: CreateRecord, ReadRecord, and ShareRecord with role-based access control
- **Security**: TLS-enabled communication, attribute-based access control, and cross-organization authorization

## Latency Testing Script

The `test_latency.sh` script tests transaction latency for the primary chaincode functions across different user loads.

### Features of the Testing Script

1. **Configuration**:
   - Tests peer communication on configured ports (7051 for Org1, 9051 for Org2)
   - Targets the "emrchannel" channel and "emr" chaincode
   - Configurable user counts for testing scalability

2. **Measured Operations**:
   - `CreateRecord`: Creation of medical records (by hospitals and doctors)
   - `ReadRecord`: Access to medical records (by all roles, with permission checks)
   - `ShareRecord`: Sharing of medical records (by record owners)

3. **Role-Specific Testing**:
   - Tests hospital operations in Org1
   - Tests doctor operations in Org1
   - Tests patient operations in Org2

4. **Output Metrics**:
   - Minimum latency
   - Maximum latency
   - Average latency
   - Error tracking

### Usage

Run the script to test network performance with increasing user loads:

```bash
./test_latency.sh
```

The script automatically runs tests for 6, 12, 18, and 24 users (distributed equally among roles) and appends results to `latency_res.txt` in CSV format.

### Example Output

```
=== EMR Network Latency Test ===
Date: 2025-04-08 22:30:15
Number of Users: 6
Channel: emrchannel
Chaincode: emr
----------------------------------------
Role,Operation,Min Latency,Max Latency,Avg Latency
hospital,CreateRecord,1.423,2.105,1.752
hospital,ReadRecord,0.876,1.245,1.021
hospital,ShareRecord,1.312,1.987,1.654
doctor,CreateRecord,1.456,2.234,1.845
doctor,ReadRecord,0.912,1.324,1.118
doctor,ShareRecord,1.376,2.123,1.725
patient,ReadRecord,0.934,1.354,1.137
----------------------------------------
```

## Performance Considerations

When analyzing results, consider:

1. **Cross-Organization Transactions**: Operations between Org1 and Org2 may have higher latency
2. **Endorsement Policy Impact**: The default OR policy between organizations affects validation time
3. **CouchDB Query Performance**: Rich queries may add latency compared to key-based lookups
4. **Network Contention**: Higher user counts may increase latency non-linearly

## Modifying the Script

To test with different configurations:

1. Edit the user counts in the main loop
2. Modify the operations being tested
3. Change peer addresses for testing different network topologies

