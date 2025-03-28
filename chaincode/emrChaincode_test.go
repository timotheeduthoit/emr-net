package main

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockStub struct {
	mock.Mock
	shim.ChaincodeStubInterface
}

type MockClientIdentity struct {
	mock.Mock
	cid.ClientIdentity
}

type MockResultsIterator struct {
	mock.Mock
	shim.StateQueryIteratorInterface
}

func (m *MockStub) PutState(key string, value []byte) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockStub) GetState(key string) ([]byte, error) {
	args := m.Called(key)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStub) GetQueryResult(query string) (shim.StateQueryIteratorInterface, error) {
	args := m.Called(query)
	return args.Get(0).(shim.StateQueryIteratorInterface), args.Error(1)
}

func (m *MockClientIdentity) GetAttributeValue(attrName string) (string, bool, error) {
	args := m.Called(attrName)
	return args.String(0), args.Bool(1), args.Error(2)
}

func (m *MockClientIdentity) GetID() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockResultsIterator) HasNext() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockResultsIterator) Next() (*queryresult.KV, error) {
	args := m.Called()
	return args.Get(0).(*queryresult.KV), args.Error(1)
}

func (m *MockResultsIterator) Close() error {
	args := m.Called()
	return args.Error(0)
}

type mockTransactionContext struct {
	contractapi.TransactionContextInterface
	stub           *MockStub
	clientIdentity *MockClientIdentity
}

func (ctx *mockTransactionContext) GetStub() shim.ChaincodeStubInterface {
	return ctx.stub
}

func (ctx *mockTransactionContext) GetClientIdentity() cid.ClientIdentity {
	return ctx.clientIdentity
}

// Doctors should be able to create records for patients
func TestCreateRecordDoctor(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockStub.On("PutState", "emr1", mock.Anything).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.CreateRecord(ctx, "emr1", "patient1", "doctor1", "hospital1", "diagnosis1")
	assert.NoError(t, err)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

// Hospitals should be able to create records for patients
func TestCreateRecordHospital(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	mockClientIdentity.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockStub.On("PutState", "emr1", mock.Anything).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.CreateRecord(ctx, "emr1", "patient1", "doctor1", "hospital1", "diagnosis1")
	assert.NoError(t, err)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

// Patients should not be able to create records
func TestCreateRecordPatient(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	mockClientIdentity.On("GetAttributeValue", "role").Return("patient", true, nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.CreateRecord(ctx, "emr1", "patient1", "doctor1", "hospital1", "diagnosis1")
	assert.Error(t, err)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

// Doctors should be able to read records they created
func TestReadRecordDoctorOwner(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emr := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		Diagnosis:           "diagnosis1",
		HospitalID:          "hospital1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentity.On("GetID").Return("doctor1", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emr, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

// Doctors should not be able to read records they did not create
func TestReadRecordDoctorNotOwner(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emr := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		Diagnosis:           "diagnosis1",
		HospitalID:          "hospital1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentity.On("GetID").Return("doctor2", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.Error(t, err)
	assert.Nil(t, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

// Hospitals should be able to read records they created
func TestReadRecordHospitalOwner(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emr := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		Diagnosis:           "diagnosis1",
		HospitalID:          "hospital1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentity.On("GetID").Return("hospital1", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emr, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)

}

func TestReadRecordPatientOwner(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emr := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		Diagnosis:           "diagnosis1",
		HospitalID:          "hospital1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("patient", true, nil)
	mockClientIdentity.On("GetID").Return("patient1", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emr, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

func TestReadRecordPatientNotOwner(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emr := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		Diagnosis:           "diagnosis1",
		HospitalID:          "hospital1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("patient", true, nil)
	mockClientIdentity.On("GetID").Return("patient2", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.Error(t, err)
	assert.Nil(t, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

func TestShareRecord(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emr := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		Diagnosis:           "diagnosis1",
		HospitalID:          "hospital1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("patient", true, nil)
	mockClientIdentity.On("GetID").Return("patient1", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil)
	mockStub.On("PutState", "emr1", mock.Anything).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.ShareRecord(ctx, "emr1", "doctor2", "doctor")
	assert.NoError(t, err)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

func TestGetAllRecordsForPatient(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)
	mockResultsIterator := new(MockResultsIterator)

	emr1 := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		HospitalID:          "hospital1",
		Diagnosis:           "diagnosis1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{},
	}
	emr1JSON, _ := json.Marshal(emr1)

	emr2 := EMR{
		EMRID:               "emr2",
		PatientID:           "patient1",
		DoctorID:            "doctor2",
		HospitalID:          "hospital2",
		Diagnosis:           "diagnosis2",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{},
	}
	emr2JSON, _ := json.Marshal(emr2)

	// Setup query results
	kv1 := &queryresult.KV{
		Key:   "emr1",
		Value: emr1JSON,
	}
	kv2 := &queryresult.KV{
		Key:   "emr2",
		Value: emr2JSON,
	}

	mockClientIdentity.On("GetAttributeValue", "role").Return("patient", true, nil)
	mockClientIdentity.On("GetID").Return("patient1", nil)
	mockStub.On("GetQueryResult", mock.Anything).Return(mockResultsIterator, nil)
	mockResultsIterator.On("HasNext").Return(true).Once()
	mockResultsIterator.On("HasNext").Return(true).Once()
	mockResultsIterator.On("HasNext").Return(false)
	mockResultsIterator.On("Next").Return(kv1, nil).Once()
	mockResultsIterator.On("Next").Return(kv2, nil).Once()
	mockResultsIterator.On("Close").Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	results, err := chaincode.GetAllRecordsForPatient(ctx, "patient1")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, &emr1, results[0])
	assert.Equal(t, &emr2, results[1])

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
	mockResultsIterator.AssertExpectations(t)
}
