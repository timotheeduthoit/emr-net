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
func (c *EMRChaincode) CreateRecord(ctx contractapi.TransactionContextInterface, emrID string, patientID string, doctorID string, hospitalID string, diagnosis string) error {
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
func (c *EMRChaincode) ShareRecord(ctx contractapi.TransactionContextInterface, emrID string, shareWithID string, shareWithRole string) error {
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
		emr.SharedWithDoctors = append(emr.SharedWithDoctors, shareWithID)
	} else if shareWithRole == "hospital" {
		emr.SharedWithHospitals = append(emr.SharedWithHospitals, shareWithID)
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
func (c *EMRChaincode) GetAllRecordsForPatient(ctx contractapi.TransactionContextInterface, patientID string) ([]EMR, error) {
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

	queryString := fmt.Sprintf(`{"selector":{"patientID":"%s"}}`, patientID)

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

	// Add other attributes as needed
	// Example: department, organization, etc.
	// department, found, err := ctx.GetClientIdentity().GetAttributeValue("department")
	// if found {
	//     attributes["department"] = department
	// }

	return attributes, nil
}
