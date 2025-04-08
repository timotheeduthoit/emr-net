package main

import (
	"encoding/json"
	"fmt"
	"time"

	"slices"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type EMRChaincode struct {
	contractapi.Contract
}

type User struct {
	UserID     string `json:"userId"`
	Role       string `json:"role"`
	CommonName string `json:"CommonName"`
}

type EMR struct {
	EMRID               string   `json:"emrId"`
	PatientID           string   `json:"patientId"`
	DoctorID            string   `json:"doctorId"`
	HospitalID          string   `json:"hospitalId,omitempty"` // Optional field
	Diagnosis           string   `json:"diagnosis"`
	CreatedOn           string   `json:"createdOn"`
	LastModified        string   `json:"lastModified"`
	SharedWithDoctors   []string `json:"sharedWithDoctors"`
	SharedWithHospitals []string `json:"sharedWithHospitals"`
}

// CreateRecord creates a new EMR record
// patientCommonName should be the CommonName of the patient with patient@orgName.example.com
func (c *EMRChaincode) CreateRecord(ctx contractapi.TransactionContextInterface, emrID string, patientCommonName string, doctorCommonName string, hospitalCommonName string, diagnosis string) error {
	role, found, err := ctx.GetClientIdentity().GetAttributeValue("role")
	if err != nil {
		return fmt.Errorf("failed to get role attribute: %v", err)
	}
	if !found || (role != "doctor" && role != "hospital") {
		return fmt.Errorf("only doctors and hospitals can create records")
	}

	// Get ID from ctx
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client ID: %v", err)
	}

	doctorID := ""
	hospitalID := ""

	if role == "doctor" {
		doctorID = clientID
	} else if role == "hospital" {
		hospitalID = clientID
	} else {
		return fmt.Errorf("invalid role: %s", role)
	}

	// Check if the EMR ID already exists
	existingEMR, err := ctx.GetStub().GetState(emrID)
	if err != nil {
		return fmt.Errorf("failed to check if EMR ID exists: %v", err)
	}
	if existingEMR != nil {
		return fmt.Errorf("EMR with ID %s already exists", emrID)
	}

	// Retrieve doctor or hospita ID
	if role == "doctor" { // DoctorID has already been set to UserID
		// Check if hospital exists
		hospital, err := c.GetUser(ctx, hospitalCommonName)
		if err != nil || hospital == nil {
			// Failed to get hospital or hospital does not exist (create without hospital)
			hospitalID = ""
		} else {
			hospitalID = hospital.UserID
		}
	} else if role == "hospital" { // HospitalID has already been set to UserID
		// Check if doctor exists
		doctor, err := c.GetUser(ctx, doctorCommonName)
		if err != nil || doctor == nil {
			// Failed to get doctor or doctor does not exist (create without doctor)
			doctorID = ""
		} else {
			doctorID = doctor.UserID
		}
	}

	patient, err := c.GetUser(ctx, patientCommonName)
	if err != nil {
		return fmt.Errorf("failed to get patient: %v", err)
	}
	if patient == nil {
		return fmt.Errorf("patient with CommonName %s does not exist", patientCommonName)
	}
	patientID := patient.UserID
	if patient.Role != "patient" {
		return fmt.Errorf("user with CommonName %s is not a patient", patientCommonName)
	}

	timestamp := time.Now().Format(time.RFC3339)
	emr := EMR{
		EMRID:               emrID,
		PatientID:           patientID,
		DoctorID:            doctorID,
		HospitalID:          hospitalID,
		CreatedOn:           timestamp,
		LastModified:        timestamp,
		Diagnosis:           diagnosis,
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{},
	}

	emrJSON, err := json.Marshal(emr)
	if err != nil {
		return fmt.Errorf("failed to marshal EMR: %v", err)
	}

	return ctx.GetStub().PutState(emrID, emrJSON)
}

// ReadRecord retrieves an EMR record by ID
func (c *EMRChaincode) ReadRecord(ctx contractapi.TransactionContextInterface, emrID string) (*EMR, error) {
	role, found, err := ctx.GetClientIdentity().GetAttributeValue("role")
	if err != nil {
		return nil, fmt.Errorf("failed to get role attribute: %v", err)
	}
	if !found {
		return nil, fmt.Errorf("role attribute not found")
	}

	emrJSON, err := ctx.GetStub().GetState(emrID)
	if err != nil {
		return nil, fmt.Errorf("failed to get state for EMR ID %s: %v", emrID, err)
	}
	if emrJSON == nil {
		return nil, fmt.Errorf("record with ID %s does not exist", emrID)
	}

	var emr EMR
	err = json.Unmarshal(emrJSON, &emr)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal EMR: %v", err)
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client ID: %v", err)
	}

	if !c.isAuthorizedToRead(role, clientID, &emr) {
		return nil, fmt.Errorf("this %s is not authorized to read this record", role)
	}

	return &emr, nil
}

// ShareRecord shares an EMR record with another entity
func (c *EMRChaincode) ShareRecord(ctx contractapi.TransactionContextInterface, emrID string, shareWithCommonName string, shareWithRole string) error {
	role, found, err := ctx.GetClientIdentity().GetAttributeValue("role")
	if err != nil {
		return fmt.Errorf("failed to get role attribute: %v", err)
	}
	if !found {
		return fmt.Errorf("role attribute not found")
	}

	emrJSON, err := ctx.GetStub().GetState(emrID)
	if err != nil {
		return fmt.Errorf("failed to get state for EMR ID %s: %v", emrID, err)
	}
	if emrJSON == nil {
		return fmt.Errorf("record with ID %s does not exist", emrID)
	}

	var emr EMR
	err = json.Unmarshal(emrJSON, &emr)
	if err != nil {
		return fmt.Errorf("failed to unmarshal EMR: %v", err)
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client ID: %v", err)
	}

	if !c.isAuthorizedToShare(role, clientID, &emr) {
		return fmt.Errorf("this %s is not authorized to share this record", role)
	}

	if shareWithRole == "doctor" {
		// Find the doctor ID from the CommonName
		doctor, err := c.GetUser(ctx, shareWithCommonName)
		if err != nil || doctor == nil {
			return fmt.Errorf("failed to get doctor: %v for sharing emr with ID %s", err, emrID)
		}
		emr.SharedWithDoctors = append(emr.SharedWithDoctors, doctor.UserID)
	} else if shareWithRole == "hospital" {
		// Find the hospital ID from the CommonName
		hospital, err := c.GetUser(ctx, shareWithCommonName)
		if err != nil || hospital == nil {
			return fmt.Errorf("failed to get hospital: %v for sharing emr with ID %s", err, emrID)
		}
		emr.SharedWithHospitals = append(emr.SharedWithHospitals, hospital.UserID)
	} else {
		return fmt.Errorf("invalid role to share with: %s", shareWithRole)
	}

	emrJSON, err = json.Marshal(emr)
	if err != nil {
		return fmt.Errorf("failed to marshal EMR: %v", err)
	}

	return ctx.GetStub().PutState(emrID, emrJSON)
}

// GetAllRecordsForPatient retrieves all EMR records for a given patient
func (c *EMRChaincode) GetAllRecordsForPatient(ctx contractapi.TransactionContextInterface, patientCommonName string) ([]EMR, error) {
	role, found, err := ctx.GetClientIdentity().GetAttributeValue("role")
	if err != nil {
		return nil, fmt.Errorf("failed to get role attribute: %v", err)
	}
	if !found {
		return nil, fmt.Errorf("role attribute not found")
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client ID: %v", err)
	}

	// Retrieve the patient ID from the CommonName
	patient, err := c.GetUser(ctx, patientCommonName)
	if err != nil || patient == nil {
		return nil, fmt.Errorf("failed to get patient: %v", err)
	}
	if patient.Role != "patient" {
		return nil, fmt.Errorf("user with CommonName %s is not a patient", patientCommonName)
	}

	queryString := fmt.Sprintf(`{"selector":{"patientID":"%s"}}`, patient.UserID)

	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, fmt.Errorf("failed to get query result: %v", err)
	}
	defer resultsIterator.Close()

	var emrs []EMR
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next query result: %v", err)
		}

		var emr EMR
		err = json.Unmarshal(queryResponse.Value, &emr)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal EMR: %v", err)
		}

		if !c.isAuthorizedToRead(role, clientID, &emr) {
			continue // Skip records that the client is not authorized to access
		}

		emrs = append(emrs, emr)
	}

	return emrs, nil
}

// isAuthorizedToRead checks if the client is authorized to read the EMR
func (c *EMRChaincode) isAuthorizedToRead(role string, clientID string, emr *EMR) bool {
	if role == "hospital" && (clientID == "" || emr.HospitalID == "") {
		// Explicitly deny access if either clientID or HospitalID is empty
		return false
	}

	return (role == "patient" && clientID == emr.PatientID) ||
		(role == "doctor" && (clientID == emr.DoctorID || slices.Contains(emr.SharedWithDoctors, clientID))) ||
		(role == "hospital" && (clientID == emr.HospitalID || slices.Contains(emr.SharedWithHospitals, clientID)))
}

// isAuthorizedToShare checks if the client is authorized to share the EMR
func (c *EMRChaincode) isAuthorizedToShare(role string, clientID string, emr *EMR) bool {
	return (role == "patient" && clientID == emr.PatientID) ||
		(role == "doctor" && (clientID == emr.DoctorID || slices.Contains(emr.SharedWithDoctors, clientID))) ||
		(role == "hospital" && (clientID == emr.HospitalID || slices.Contains(emr.SharedWithHospitals, clientID)))
}

func main() {
	chaincode, err := contractapi.NewChaincode(new(EMRChaincode))
	if err != nil {
		fmt.Printf("Error create EMRChaincode: %s", err.Error())
		return
	}

	// Register the chaincode
	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting EMRChaincode: %s", err.Error())
	}
}

// GetIdentityAttributes retrieves all attributes of the invoking client identity
func (c *EMRChaincode) GetIdentityAttributes(ctx contractapi.TransactionContextInterface) (map[string]string, error) {
	attributes := make(map[string]string)

	// Get the client ID
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client ID: %v", err)
	}
	attributes["clientID"] = clientID

	// Get the role attribute
	role, found, err := ctx.GetClientIdentity().GetAttributeValue("role")
	if err != nil {
		return nil, fmt.Errorf("failed to get role attribute: %v", err)
	}
	if !found {
		return nil, fmt.Errorf("role attribute not found for client ID: %s", clientID)
	}
	attributes["role"] = role

	// Get the organization affiliation
	orgName, found, err := ctx.GetClientIdentity().GetAttributeValue("hf.Affiliation")
	if err != nil {
		return nil, fmt.Errorf("failed to get organization affiliation: %v", err)
	}
	if !found {
		return nil, fmt.Errorf("organization affiliation not found for client ID: %s", clientID)
	}
	attributes["organization"] = orgName

	// Get the CommonName from the X.509 certificate
	cert, err := ctx.GetClientIdentity().GetX509Certificate()
	if err != nil {
		return nil, fmt.Errorf("failed to get X.509 certificate: %v", err)
	}
	attributes["CommonName"] = cert.Subject.CommonName

	return attributes, nil
}

func (c *EMRChaincode) RegisterUser(ctx contractapi.TransactionContextInterface) error {
	// Get the client ID
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client ID: %v", err)
	}

	// Extract the CommonName from the X.509 certificate
	cert, err := ctx.GetClientIdentity().GetX509Certificate()
	if err != nil {
		return fmt.Errorf("failed to get X.509 certificate: %v", err)
	}

	// Extract the organization name from the client identity attributes
	orgName, found, err := ctx.GetClientIdentity().GetAttributeValue("hf.Affiliation")
	if err != nil {
		return fmt.Errorf("failed to get organization affiliation: %v", err)
	}
	if !found {
		return fmt.Errorf("organization affiliation not found for client ID: %s", clientID)
	}

	// Construct the fullName dynamically
	fullName := fmt.Sprintf("%s@%s.example.com", cert.Subject.CommonName, orgName)

	// Check if the user is already registered
	existingUser, err := ctx.GetStub().GetState(fullName)
	if err != nil {
		return fmt.Errorf("failed to check if user is already registered: %v", err)
	}
	if existingUser != nil {
		return fmt.Errorf("user with CommonName %s is already registered", fullName)
	}

	// Get the role attribute
	role, found, err := ctx.GetClientIdentity().GetAttributeValue("role")
	if err != nil {
		return fmt.Errorf("failed to get role attribute: %v", err)
	}
	if !found {
		return fmt.Errorf("role attribute not found for client ID: %s", clientID)
	}

	// Create a new user object
	user := User{
		UserID:     clientID,
		Role:       role,
		CommonName: fullName,
	}

	// Serialize the user object to JSON
	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %v", err)
	}

	// Store the user in the ledger
	return ctx.GetStub().PutState(fullName, userJSON)
}

func (c *EMRChaincode) GetUser(ctx contractapi.TransactionContextInterface, commonName string) (*User, error) {
	userJSON, err := ctx.GetStub().GetState(commonName)

	if err != nil {
		return nil, fmt.Errorf("failed to get user with CommonName %s: %v", commonName, err)
	}

	if userJSON == nil {
		return nil, fmt.Errorf("user with CommonName %s does not exist", commonName)
	}

	var user User
	err = json.Unmarshal(userJSON, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %v", err)
	}

	return &user, nil
}
