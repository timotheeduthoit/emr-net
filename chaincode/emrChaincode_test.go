package main

import (
	"encoding/json"
	"fmt"
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
	mockClientIdentity.On("GetID").Return("doctor1", nil)
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
	mockClientIdentity.On("GetID").Return("hospital1", nil)
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

// Hospitals should not be able to read records they did not create
func TestReadRecordHospitalNotOwner(t *testing.T) {
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
	mockClientIdentity.On("GetID").Return("hospital2", nil)
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

// Test read for hospital when no hospital ID is present
func TestReadRecordHospitalNoID(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emr := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		Diagnosis:           "diagnosis1",
		HospitalID:          "",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentity.On("GetID").Return("hospital2", nil)
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

// Empty hospital ID should not be allowed
func TestReadRecordHospitalEmptyID(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emr := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		Diagnosis:           "diagnosis1",
		HospitalID:          "",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentity.On("GetID").Return("", nil)
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

// Patients should be able to read their own records
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

// Patients should not be able to read records they do not own
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

func TestShareRecordDoctorOwnerToDoctor(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emrBase := EMR{
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
	emrBaseJSON, _ := json.Marshal(emrBase)

	emrExpected := emrBase
	emrExpected.SharedWithDoctors = []string{"doctor2"}
	emrExpectedJSON, _ := json.Marshal(emrExpected)

	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentity.On("GetID").Return("doctor1", nil)
	mockStub.On("GetState", "emr1").Return(emrBaseJSON, nil)
	mockStub.On("PutState", "emr1", emrExpectedJSON).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	// Share the record with another doctor
	err := chaincode.ShareRecord(ctx, "emr1", "doctor2", "doctor")
	assert.NoError(t, err)

	// Verify doctor2 can access the record
	mockClientIdentityDoctor2 := new(MockClientIdentity)
	mockClientIdentityDoctor2.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentityDoctor2.On("GetID").Return("doctor2", nil)
	mockStubDoctor := new(MockStub)
	mockStubDoctor.On("GetState", "emr1").Return(emrExpectedJSON, nil)
	ctx.clientIdentity = mockClientIdentityDoctor2
	ctx.stub = mockStubDoctor
	// Attempt to read the record
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emrExpected, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
	mockClientIdentityDoctor2.AssertExpectations(t)
	mockStubDoctor.AssertExpectations(t)
}

func TestShareRecordDoctorShareListToDoctor(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emrBase := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		Diagnosis:           "diagnosis1",
		HospitalID:          "hospital1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{"doctor2"},
		SharedWithHospitals: []string{},
	}
	emrBaseJSON, _ := json.Marshal(emrBase)

	emrExpected := emrBase
	emrExpected.SharedWithDoctors = []string{"doctor2", "doctor3"}
	emrExpectedJSON, _ := json.Marshal(emrExpected)

	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentity.On("GetID").Return("doctor2", nil)
	mockStub.On("GetState", "emr1").Return(emrBaseJSON, nil)
	mockStub.On("PutState", "emr1", emrExpectedJSON).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.ShareRecord(ctx, "emr1", "doctor3", "doctor")
	assert.NoError(t, err)

	// Verify doctor3 can access the record
	mockClientIdentityDoctor3 := new(MockClientIdentity)
	mockClientIdentityDoctor3.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentityDoctor3.On("GetID").Return("doctor3", nil)
	mockStubDoctor := new(MockStub)
	mockStubDoctor.On("GetState", "emr1").Return(emrExpectedJSON, nil)
	ctx.clientIdentity = mockClientIdentityDoctor3
	ctx.stub = mockStubDoctor
	// Attempt to read the record
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emrExpected, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
	mockClientIdentityDoctor3.AssertExpectations(t)
	mockStubDoctor.AssertExpectations(t)
}

func TestShareRecordDoctorOwnerToHospital(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emrBase := EMR{
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
	emrBaseJSON, _ := json.Marshal(emrBase)

	emrExpected := emrBase
	emrExpected.SharedWithHospitals = []string{"hospital2"}
	emrExpectedJSON, _ := json.Marshal(emrExpected)

	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentity.On("GetID").Return("doctor1", nil)
	mockStub.On("GetState", "emr1").Return(emrBaseJSON, nil)
	mockStub.On("PutState", "emr1", emrExpectedJSON).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.ShareRecord(ctx, "emr1", "hospital2", "hospital")
	assert.NoError(t, err)

	// Verify hospital2 can access the record
	mockClientIdentityHospital := new(MockClientIdentity)
	mockClientIdentityHospital.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentityHospital.On("GetID").Return("hospital2", nil)
	mockStubHospital := new(MockStub)
	mockStubHospital.On("GetState", "emr1").Return(emrExpectedJSON, nil)
	ctx.clientIdentity = mockClientIdentityHospital
	ctx.stub = mockStubHospital
	// Attempt to read the record
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emrExpected, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
	mockClientIdentityHospital.AssertExpectations(t)
	mockStubHospital.AssertExpectations(t)
}

func TestShareRecordHospitalOwnerToDoctor(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emrBase := EMR{
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
	emrBaseJSON, _ := json.Marshal(emrBase)

	emrExpected := emrBase
	emrExpected.SharedWithDoctors = []string{"doctor2"}
	emrExpectedJSON, _ := json.Marshal(emrExpected)

	mockClientIdentity.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentity.On("GetID").Return("hospital1", nil)
	mockStub.On("GetState", "emr1").Return(emrBaseJSON, nil)
	mockStub.On("PutState", "emr1", emrExpectedJSON).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.ShareRecord(ctx, "emr1", "doctor2", "doctor")
	assert.NoError(t, err)

	// Verify doctor2 can access the record
	mockClientIdentityDoctor := new(MockClientIdentity)
	mockClientIdentityDoctor.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentityDoctor.On("GetID").Return("doctor2", nil)
	mockStubDoctor := new(MockStub)
	mockStubDoctor.On("GetState", "emr1").Return(emrExpectedJSON, nil)
	ctx.clientIdentity = mockClientIdentityDoctor
	ctx.stub = mockStubDoctor
	// Attempt to read the record
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emrExpected, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
	mockClientIdentityDoctor.AssertExpectations(t)
	mockStubDoctor.AssertExpectations(t)
}

func TestShareRecordHospitalShareListToHospital(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emrBase := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		Diagnosis:           "diagnosis1",
		HospitalID:          "hospital1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{"hospital2"},
	}
	emrBaseJSON, _ := json.Marshal(emrBase)

	emrExpected := emrBase
	emrExpected.SharedWithHospitals = []string{"hospital2", "hospital3"}
	emrExpectedJSON, _ := json.Marshal(emrExpected)

	mockClientIdentity.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentity.On("GetID").Return("hospital2", nil)
	mockStub.On("GetState", "emr1").Return(emrBaseJSON, nil)
	mockStub.On("PutState", "emr1", emrExpectedJSON).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.ShareRecord(ctx, "emr1", "hospital3", "hospital")
	assert.NoError(t, err)

	// Verify hospital3 can access the record
	mockClientIdentityHospital := new(MockClientIdentity)
	mockClientIdentityHospital.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentityHospital.On("GetID").Return("hospital3", nil)
	mockStubHospital := new(MockStub)
	mockStubHospital.On("GetState", "emr1").Return(emrExpectedJSON, nil)
	ctx.clientIdentity = mockClientIdentityHospital
	ctx.stub = mockStubHospital
	// Attempt to read the record
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emrExpected, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
	mockClientIdentityHospital.AssertExpectations(t)
	mockStubHospital.AssertExpectations(t)
}

func TestShareRecordPatientOwnerToDoctor(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emrBase := EMR{
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
	emrBaseJSON, _ := json.Marshal(emrBase)

	emrExpected := emrBase
	emrExpected.SharedWithDoctors = []string{"doctor2"}
	emrExpectedJSON, _ := json.Marshal(emrExpected)

	mockClientIdentity.On("GetAttributeValue", "role").Return("patient", true, nil)
	mockClientIdentity.On("GetID").Return("patient1", nil)
	mockStub.On("GetState", "emr1").Return(emrBaseJSON, nil)
	mockStub.On("PutState", "emr1", emrExpectedJSON).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.ShareRecord(ctx, "emr1", "doctor2", "doctor")
	assert.NoError(t, err)

	// Verify doctor2 can access the record
	mockClientIdentityDoctor := new(MockClientIdentity)
	mockClientIdentityDoctor.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentityDoctor.On("GetID").Return("doctor2", nil)
	mockStubDoctor := new(MockStub)
	mockStubDoctor.On("GetState", "emr1").Return(emrExpectedJSON, nil)
	ctx.clientIdentity = mockClientIdentityDoctor
	ctx.stub = mockStubDoctor
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emrExpected, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
	mockClientIdentityDoctor.AssertExpectations(t)
	mockStubDoctor.AssertExpectations(t)
}

func TestShareRecordPatientOwnerToHospital(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emrBase := EMR{
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
	emrBaseJSON, _ := json.Marshal(emrBase)

	emrExpected := emrBase
	emrExpected.SharedWithHospitals = []string{"hospital2"}
	emrExpectedJSON, _ := json.Marshal(emrExpected)

	mockClientIdentity.On("GetAttributeValue", "role").Return("patient", true, nil)
	mockClientIdentity.On("GetID").Return("patient1", nil)
	mockStub.On("GetState", "emr1").Return(emrBaseJSON, nil)
	mockStub.On("PutState", "emr1", emrExpectedJSON).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.ShareRecord(ctx, "emr1", "hospital2", "hospital")
	assert.NoError(t, err)

	// Verify hospital2 can access the record
	mockClientIdentityHospital := new(MockClientIdentity)
	mockClientIdentityHospital.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentityHospital.On("GetID").Return("hospital2", nil)
	mockStubHospital := new(MockStub)
	mockStubHospital.On("GetState", "emr1").Return(emrExpectedJSON, nil)
	ctx.clientIdentity = mockClientIdentityHospital
	ctx.stub = mockStubHospital
	// Attempt to read the record
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emrExpected, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
	mockClientIdentityHospital.AssertExpectations(t)
	mockStubHospital.AssertExpectations(t)
}

func TestShareRecordDoctorNotAuthorizedToDoctorAndHospital(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)
	emr := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		HospitalID:          "hospital1",
		Diagnosis:           "diagnosis1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{"doctor2"},
		SharedWithHospitals: []string{},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentity.On("GetID").Return("doctor3", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Times(2) // Once for doctor, once for hospital
	// Do not set PutState expectation here since sharing should fail
	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	// Attempt to share with a doctor
	err := chaincode.ShareRecord(ctx, "emr1", "doctor4", "doctor")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "doctor is not authorized to share")

	// Attempt to share with a hospital
	err = chaincode.ShareRecord(ctx, "emr1", "hospital2", "hospital")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "doctor is not authorized to share")

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

func TestShareRecordHospitalNotAuthorizedToDoctorAndHospital(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)
	emr := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		HospitalID:          "hospital1",
		Diagnosis:           "diagnosis1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{"doctor2"},
		SharedWithHospitals: []string{"hospital2"},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentity.On("GetID").Return("hospital3", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Times(2) // Once for doctor, once for hospital
	// Do not set PutState expectation here since sharing should fail
	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	// Attempt to share with a doctor
	err := chaincode.ShareRecord(ctx, "emr1", "doctor3", "doctor")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hospital is not authorized to share")

	// Attempt to share with a hospital
	err = chaincode.ShareRecord(ctx, "emr1", "hospital4", "hospital")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hospital is not authorized to share")

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

func TestShareRecordPatientNotAuthorizedToDoctorAndHospital(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)
	emr := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		HospitalID:          "hospital1",
		Diagnosis:           "diagnosis1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{"doctor2"},
		SharedWithHospitals: []string{"hospital2"},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("patient", true, nil)
	mockClientIdentity.On("GetID").Return("patient2", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Times(2) // Once for doctor, once for hospital
	// Do not set PutState expectation here since sharing should fail
	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	// Attempt to share with a doctor
	err := chaincode.ShareRecord(ctx, "emr1", "doctor3", "doctor")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "patient is not authorized to share")

	// Attempt to share with a hospital
	err = chaincode.ShareRecord(ctx, "emr1", "hospital3", "hospital")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "patient is not authorized to share")

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

func TestShareRecordUnauthorizedRole(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)

	emr := EMR{
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
	emrJSON, _ := json.Marshal(emr)

	// Nurse tries to share the record
	mockClientIdentity.On("GetAttributeValue", "role").Return("nurse", true, nil)
	mockClientIdentity.On("GetID").Return("nurse1", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Times(2) // Once for doctor, once for hospital
	// Do not set PutState expectation here since sharing should fail
	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	// Attempt to share with a doctor
	err := chaincode.ShareRecord(ctx, "emr1", "doctor2", "doctor")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nurse is not authorized to share")

	// Attempt to share with a hospital
	err = chaincode.ShareRecord(ctx, "emr1", "hospital2", "hospital")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nurse is not authorized to share")

	// Assert expectations
	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

func TestShareRecordRightIDWrongRole(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)
	emrBase := EMR{
		EMRID:               "emr1",
		PatientID:           "patient1",
		DoctorID:            "doctor1",
		HospitalID:          "hospital1",
		Diagnosis:           "diagnosis1",
		CreatedOn:           "2025-03-27T12:00:00Z",
		LastModified:        "2025-03-27T12:00:00Z",
		SharedWithDoctors:   []string{"doctor2"},
		SharedWithHospitals: []string{"hospital2"},
	}
	emrBaseJSON, _ := json.Marshal(emrBase)

	emrExpected := emrBase
	emrExpected.SharedWithHospitals = []string{"hospital2", "doctor3"}
	emrExpectedJSON, _ := json.Marshal(emrExpected)

	// Share from doctor with access
	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentity.On("GetID").Return("doctor1", nil)
	mockStub.On("GetState", "emr1").Return(emrBaseJSON, nil).Once()
	mockStub.On("PutState", "emr1", emrExpectedJSON).Return(nil).Once()
	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	// Share with a doctor using right ID but wrong role
	err := chaincode.ShareRecord(ctx, "emr1", "doctor3", "hospital")
	assert.NoError(t, err)

	// Verify doctor3 cannot access the record as doctor
	mockClientIdentityDoctor := new(MockClientIdentity)
	mockClientIdentityDoctor.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentityDoctor.On("GetID").Return("doctor3", nil)
	mockStubDoctor := new(MockStub)
	mockStubDoctor.On("GetState", "emr1").Return(emrExpectedJSON, nil).Once()
	ctx.clientIdentity = mockClientIdentityDoctor
	ctx.stub = mockStubDoctor
	// Attempt to read the record
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "doctor is not authorized to read")

	// Verify doctor3 can access the record as hospital
	mockClientIdentityHospital := new(MockClientIdentity)
	mockClientIdentityHospital.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentityHospital.On("GetID").Return("doctor3", nil)
	mockStubHospital := new(MockStub)
	mockStubHospital.On("GetState", "emr1").Return(emrExpectedJSON, nil).Once()
	ctx.clientIdentity = mockClientIdentityHospital
	ctx.stub = mockStubHospital
	// Attempt to read the record
	result, err = chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emrExpected, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
	mockClientIdentityDoctor.AssertExpectations(t)
	mockStubDoctor.AssertExpectations(t)
	mockClientIdentityHospital.AssertExpectations(t)
	mockStubHospital.AssertExpectations(t)
}

func TestGetAllRecordsForPatient(t *testing.T) {
	chaincode := new(EMRChaincode)
	mockStub := new(MockStub)
	mockClientIdentity := new(MockClientIdentity)
	mockResultsIterator := new(MockResultsIterator)

	// Create total of 10 EMRs
	var emrs []EMR
	var expectedEMRs []EMR
	// Create 5 EMRs for patient2, 2 for patient7, and 3 for patient19
	for i := 0; i < 10; i++ {
		patientID := "patient2" // The patient we will retrieve records for
		if i == 3 || i == 9 {
			patientID = "patient7"
		} else if i%2 != 0 {
			patientID = "patient19"
		}

		emr := EMR{
			EMRID:               fmt.Sprintf("emr%d", i),
			PatientID:           patientID,
			DoctorID:            fmt.Sprintf("doctor%d", i%5),
			HospitalID:          fmt.Sprintf("hospital%d", i%2),
			Diagnosis:           fmt.Sprintf("diagnosis%d", i),
			CreatedOn:           "2025-03-27T12:00:00Z",
			LastModified:        "2025-03-27T12:00:00Z",
			SharedWithDoctors:   []string{},
			SharedWithHospitals: []string{},
		}
		emrs = append(emrs, emr)
		emrJSON, _ := json.Marshal(emr)
		mockResultsIterator.On("Next").Return(&queryresult.KV{
			Key:   emr.EMRID,
			Value: emrJSON,
		}, nil).Once()

		// Collect expected EMRs for patient2
		if patientID == "patient2" {
			expectedEMRs = append(expectedEMRs, emr)
		}
	}
	mockResultsIterator.On("HasNext").Return(true).Times(len(emrs))
	mockResultsIterator.On("HasNext").Return(false).Once()
	mockResultsIterator.On("Close").Return(nil)

	mockStub.On("GetQueryResult", mock.Anything).Return(mockResultsIterator, nil)
	mockClientIdentity.On("GetAttributeValue", "role").Return("patient", true, nil)
	mockClientIdentity.On("GetID").Return("patient2", nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	// Call the function to retrieve all records for patient2
	results, err := chaincode.GetAllRecordsForPatient(ctx, "patient2")
	assert.NoError(t, err)
	assert.Len(t, results, len(expectedEMRs))

	// Compare each expected EMR with the result
	for i, expected := range expectedEMRs {
		assert.Equal(t, expected, results[i])
	}

	mockResultsIterator.AssertExpectations(t)
	mockStub.AssertExpectations(t)
	mockClientIdentity.AssertExpectations(t)
}
