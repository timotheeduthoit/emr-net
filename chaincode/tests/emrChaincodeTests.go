package main

import (
    "encoding/json"
    "testing"

    "github.com/hyperledger/fabric-contract-api-go/contractapi"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// MockTransactionContext is a mock for the TransactionContextInterface
type MockTransactionContext struct {
    mock.Mock
    Stub *MockStub
}

func (m *MockTransactionContext) GetStub() contractapi.ChaincodeStubInterface {
    return m.Stub
}

func (m *MockTransactionContext) GetClientIdentity() *contractapi.ClientIdentity {
    return &contractapi.ClientIdentity{}
}

// MockStub is a mock for the ChaincodeStubInterface
type MockStub struct {
    mock.Mock
    State map[string][]byte
}

func (m *MockStub) GetState(key string) ([]byte, error) {
    args := m.Called(key)
    return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStub) PutState(key string, value []byte) error {
    m.State[key] = value
    return nil
}

func (m *MockStub) GetQueryResult(query string) (contractapi.ResultsIterator, error) {
    args := m.Called(query)
    return args.Get(0).(contractapi.ResultsIterator), args.Error(1)
}

// TestCreateRecord tests the CreateRecord function
func TestCreateRecord(t *testing.T) {
    chaincode := new(EMRChaincode)
    ctx := new(MockTransactionContext)
    stub := new(MockStub)
    ctx.Stub = stub
    stub.State = make(map[string][]byte)

    err := chaincode.CreateRecord(ctx, "emr1", "patient1", "doctor1", "diagnosis1")
    assert.Nil(t, err)

    emrJSON, exists := stub.State["emr1"]
    assert.True(t, exists)

    var emr EMR
    err = json.Unmarshal(emrJSON, &emr)
    assert.Nil(t, err)
    assert.Equal(t, "emr1", emr.EMRID)
    assert.Equal(t, "patient1", emr.PatientID)
    assert.Equal(t, "doctor1", emr.DoctorID)
    assert.Equal(t, "diagnosis1", emr.Diagnosis)
}

// TestReadRecord tests the ReadRecord function
func TestReadRecord(t *testing.T) {
    chaincode := new(EMRChaincode)
    ctx := new(MockTransactionContext)
    stub := new(MockStub)
    ctx.Stub = stub
    stub.State = make(map[string][]byte)

    emr := EMR{
        EMRID:     "emr1",
        PatientID: "patient1",
        DoctorID:  "doctor1",
        Diagnosis: "diagnosis1",
    }
    emrJSON, _ := json.Marshal(emr)
    stub.State["emr1"] = emrJSON

    stub.On("GetState", "emr1").Return(emrJSON, nil)

    result, err := chaincode.ReadRecord(ctx, "emr1")
    assert.Nil(t, err)
    assert.Equal(t, "emr1", result.EMRID)
    assert.Equal(t, "patient1", result.PatientID)
    assert.Equal(t, "doctor1", result.DoctorID)
    assert.Equal(t, "diagnosis1", result.Diagnosis)
}

// TestShareRecord tests the ShareRecord function
func TestShareRecord(t *testing.T) {
    chaincode := new(EMRChaincode)
    ctx := new(MockTransactionContext)
    stub := new(MockStub)
    ctx.Stub = stub
    stub.State = make(map[string][]byte)

    emr := EMR{
        EMRID:     "emr1",
        PatientID: "patient1",
        DoctorID:  "doctor1",
        Diagnosis: "diagnosis1",
    }
    emrJSON, _ := json.Marshal(emr)
    stub.State["emr1"] = emrJSON

    stub.On("GetState", "emr1").Return(emrJSON, nil)

    err := chaincode.ShareRecord(ctx, "emr1", "doctor2", "doctor")
    assert.Nil(t, err)

    updatedEMRJSON := stub.State["emr1"]
    var updatedEMR EMR
    err = json.Unmarshal(updatedEMRJSON, &updatedEMR)
    assert.Nil(t, err)
    assert.Contains(t, updatedEMR.SharedWithDoctors, "doctor2")
}

// TestGetAllRecordsForPatient tests the GetAllRecordsForPatient function
func TestGetAllRecordsForPatient(t *testing.T) {
    chaincode := new(EMRChaincode)
    ctx := new(MockTransactionContext)
    stub := new(MockStub)
    ctx.Stub = stub
    stub.State = make(map[string][]byte)

    emr1 := EMR{
        EMRID:     "emr1",
        PatientID: "patient1",
        DoctorID:  "doctor1",
        Diagnosis: "diagnosis1",
    }
    emr2 := EMR{
        EMRID:     "emr2",
        PatientID: "patient1",
        DoctorID:  "doctor2",
        Diagnosis: "diagnosis2",
    }
    emr1JSON, _ := json.Marshal(emr1)
    emr2JSON, _ := json.Marshal(emr2)
    stub.State["emr1"] = emr1JSON
    stub.State["emr2"] = emr2JSON

    stub.On("GetQueryResult", mock.Anything).Return(&MockResultsIterator{
        Results: [][]byte{emr1JSON, emr2JSON},
    }, nil)

    results, err := chaincode.GetAllRecordsForPatient(ctx, "patient1")
    assert.Nil(t, err)
    assert.Len(t, results, 2)
    assert.Equal(t, "emr1", results[0].EMRID)
    assert.Equal(t, "emr2", results[1].EMRID)
}

// MockResultsIterator is a mock for the ResultsIterator
type MockResultsIterator struct {
    Results [][]byte
    Index   int
}

func (m *MockResultsIterator) HasNext() bool {
    return m.Index < len(m.Results)
}

func (m *MockResultsIterator) Next() (*contractapi.QueryResult, error) {
    if m.HasNext() {
        result := m.Results[m.Index]
        m.Index++
        return &contractapi.QueryResult{Value: result}, nil
    }
    return nil, nil
}

func (m *MockResultsIterator) Close() error {
    return nil
}