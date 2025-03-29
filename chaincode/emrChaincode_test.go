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
}

func TestShareRecordDoctorShareListToDoctor(t *testing.T) {
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
		SharedWithDoctors:   []string{"doctor2"},
		SharedWithHospitals: []string{},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentity.On("GetID").Return("doctor2", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil)
	mockStub.On("PutState", "emr1", mock.Anything).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.ShareRecord(ctx, "emr1", "doctor3", "doctor")
	assert.NoError(t, err)

	// Verify doctor3 can access the record
	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentity.On("GetID").Return("doctor3", nil)
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emr, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

func TestShareRecordDoctorOwnerToHospital(t *testing.T) {
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
	mockStub.On("PutState", "emr1", mock.Anything).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.ShareRecord(ctx, "emr1", "hospital2", "hospital")
	assert.NoError(t, err)

	// Verify hospital2 can access the record
	mockClientIdentity.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentity.On("GetID").Return("hospital2", nil)
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emr, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

func TestShareRecordHospitalOwnerToDoctor(t *testing.T) {
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
	mockStub.On("PutState", "emr1", mock.Anything).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.ShareRecord(ctx, "emr1", "doctor2", "doctor")
	assert.NoError(t, err)

	// Verify doctor2 can access the record
	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentity.On("GetID").Return("doctor2", nil)
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emr, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

func TestShareRecordHospitalShareListToHospital(t *testing.T) {
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
		SharedWithHospitals: []string{"hospital2"},
	}
	emrJSON, _ := json.Marshal(emr)

	mockClientIdentity.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentity.On("GetID").Return("hospital2", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil)
	mockStub.On("PutState", "emr1", mock.Anything).Return(nil)

	ctx := &mockTransactionContext{
		stub:           mockStub,
		clientIdentity: mockClientIdentity,
	}

	err := chaincode.ShareRecord(ctx, "emr1", "hospital3", "hospital")
	assert.NoError(t, err)

	// Verify hospital3 can access the record
	mockClientIdentity.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentity.On("GetID").Return("hospital3", nil)
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emr, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

func TestShareRecordPatientOwnerToDoctor(t *testing.T) {
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

	// Verify doctor2 can access the record
	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentity.On("GetID").Return("doctor2", nil)
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emr, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
}

func TestShareRecordPatientOwnerToHospital(t *testing.T) {
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

	err := chaincode.ShareRecord(ctx, "emr1", "hospital2", "hospital")
	assert.NoError(t, err)

	// Verify hospital2 can access the record
	mockClientIdentity.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentity.On("GetID").Return("hospital2", nil)
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emr, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
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

	// Verify doctor4 cannot access the record
	mockClientIdentityDoctor := new(MockClientIdentity)
	mockClientIdentityDoctor.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentityDoctor.On("GetID").Return("doctor4", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Once()
	ctx.clientIdentity = mockClientIdentityDoctor
	// Attempt to read the record
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "doctor is not authorized to read")

	// Verify hospital2 cannot access the record
	mockClientIdentityHospital := new(MockClientIdentity)
	mockClientIdentityHospital.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentityHospital.On("GetID").Return("hospital2", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Once()
	ctx.clientIdentity = mockClientIdentityHospital
	// Attempt to read the record
	result, err = chaincode.ReadRecord(ctx, "emr1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "hospital is not authorized to read")

	mockClientIdentity.AssertExpectations(t)
	mockClientIdentityDoctor.AssertExpectations(t)
	mockClientIdentityHospital.AssertExpectations(t)
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

	// Verify doctor3 cannot access the record
	mockClientIdentityDoctor := new(MockClientIdentity)
	mockClientIdentityDoctor.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentityDoctor.On("GetID").Return("doctor3", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Once()
	ctx.clientIdentity = mockClientIdentityDoctor
	// Attempt to read the record
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "doctor is not authorized to read")

	// Verify hospital4 cannot access the record
	mockClientIdentityHospital := new(MockClientIdentity)
	mockClientIdentityHospital.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentityHospital.On("GetID").Return("hospital4", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Once()
	ctx.clientIdentity = mockClientIdentityHospital
	// Attempt to read the record
	result, err = chaincode.ReadRecord(ctx, "emr1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "hospital is not authorized to read")

	mockClientIdentity.AssertExpectations(t)
	mockClientIdentityDoctor.AssertExpectations(t)
	mockClientIdentityHospital.AssertExpectations(t)
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

	// Verify doctor3 cannot access the record
	mockClientIdentityDoctor := new(MockClientIdentity)
	mockClientIdentityDoctor.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentityDoctor.On("GetID").Return("doctor3", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Once()
	ctx.clientIdentity = mockClientIdentityDoctor
	// Attempt to read the record
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "doctor is not authorized to read")

	// Verify hospital3 cannot access the record
	mockClientIdentityHospital := new(MockClientIdentity)
	mockClientIdentityHospital.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentityHospital.On("GetID").Return("hospital3", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Once()
	ctx.clientIdentity = mockClientIdentityHospital
	// Attempt to read the record
	result, err = chaincode.ReadRecord(ctx, "emr1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "hospital is not authorized to read")

	mockClientIdentity.AssertExpectations(t)
	mockClientIdentityDoctor.AssertExpectations(t)
	mockClientIdentityHospital.AssertExpectations(t)
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

	// Doctor tries to read the record
	// Setup mock
	mockClientIdentityDoctor := new(MockClientIdentity)
	mockClientIdentityDoctor.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentityDoctor.On("GetID").Return("doctor2", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Once()
	ctx.clientIdentity = mockClientIdentityDoctor
	// Attempt to read the record
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "doctor is not authorized to read")

	// Hospital tries to read the record
	// Setup mock
	mockClientIdentityHospital := new(MockClientIdentity)
	mockClientIdentityHospital.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentityHospital.On("GetID").Return("hospital2", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Once()
	ctx.clientIdentity = mockClientIdentityHospital
	// Attempt to read the record
	result, err = chaincode.ReadRecord(ctx, "emr1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "hospital is not authorized to read")

	// Assert expectations
	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
	mockClientIdentityDoctor.AssertExpectations(t)
	mockClientIdentityHospital.AssertExpectations(t)
}

func TestShareRecordRightIDWrongRole(t *testing.T) {
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

	// Share from doctor with access
	mockClientIdentity.On("GetAttributeValue", "role").Return("doctor", true, nil)
	mockClientIdentity.On("GetID").Return("doctor1", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Once()
	mockStub.On("PutState", "emr1", mock.Anything).Return(nil).Once()
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
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Once()
	ctx.clientIdentity = mockClientIdentityDoctor
	// Attempt to read the record
	result, err := chaincode.ReadRecord(ctx, "emr1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "doctor is not authorized to read")

	// Verify doctor3 can access the record as hospital
	mockClientIdentityHospital := new(MockClientIdentity)
	mockClientIdentityHospital.On("GetAttributeValue", "role").Return("hospital", true, nil)
	mockClientIdentityHospital.On("GetID").Return("doctor3", nil)
	mockStub.On("GetState", "emr1").Return(emrJSON, nil).Once()
	ctx.clientIdentity = mockClientIdentityHospital
	// Attempt to read the record
	result, err = chaincode.ReadRecord(ctx, "emr1")
	assert.NoError(t, err)
	assert.Equal(t, &emr, result)

	mockClientIdentity.AssertExpectations(t)
	mockStub.AssertExpectations(t)
	mockClientIdentityDoctor.AssertExpectations(t)
	mockClientIdentityHospital.AssertExpectations(t)
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
