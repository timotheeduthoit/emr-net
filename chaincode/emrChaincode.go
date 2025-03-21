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
	EMRID               string   `json:"emrID"`
	PatientID           string   `json:"patientID"`
	DoctorID            string   `json:"doctorID"`
	CreatedOn           string   `json:"createdOn"`
	LastModified        string   `json:"lastModified"`
	Diagnosis           string   `json:"diagnosis"`
	SharedWithDoctors   []string `json:"sharedWithDoctors"`
	SharedWithHospitals []string `json:"sharedWithHospitals"`
}

func (c *EMRChaincode) CreateRecord(ctx contractapi.TransactionContextInterface, emrID string, patientID string, doctorID string, diagnosis string) error {
	role, found, err := ctx.GetClientIdentity().GetAttributeValue("role")
	if err != nil {
		return fmt.Errorf("failed to get role attribute: %v", err)
	}
	if !found || (role != "doctor" && role != "hospital") {
		return fmt.Errorf("only doctors and hospitals can create records")
	}

	timestamp := time.Now().Format(time.RFC3339)
	emr := EMR{
		EMRID:               emrID,
		PatientID:           patientID,
		DoctorID:            doctorID,
		CreatedOn:           timestamp,
		LastModified:        timestamp,
		Diagnosis:           diagnosis,
		SharedWithDoctors:   []string{},
		SharedWithHospitals: []string{},
	}

	emrJSON, err := json.Marshal(emr)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(emrID, emrJSON)
}

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
		return nil, err
	}
	if emrJSON == nil {
		return nil, fmt.Errorf("record with ID %s does not exist", emrID)
	}

	var emr EMR
	err = json.Unmarshal(emrJSON, &emr)
	if err != nil {
		return nil, err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client ID: %v", err)
	}

	if (role == "patient" && clientID != emr.PatientID) ||
		(role == "doctor" && clientID != emr.DoctorID && !slices.Contains(emr.SharedWithDoctors, clientID)) ||
		(role == "hospital" && clientID != emr.DoctorID && !slices.Contains(emr.SharedWithHospitals, clientID)) {
		return nil, fmt.Errorf("this %s is not authorized to read this record", role)
	}

	return &emr, nil
}

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
		return err
	}
	if emrJSON == nil {
		return fmt.Errorf("record with ID %s does not exist", emrID)
	}

	var emr EMR
	err = json.Unmarshal(emrJSON, &emr)
	if err != nil {
		return err
	}

	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client ID: %v", err)
	}

	if (role == "patient" && clientID != emr.PatientID) ||
		(role == "doctor" && clientID != emr.DoctorID && !slices.Contains(emr.SharedWithDoctors, clientID)) ||
		(role == "hospital" && clientID != emr.DoctorID && !slices.Contains(emr.SharedWithHospitals, clientID)) {
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
		return err
	}

	return ctx.GetStub().PutState(emrID, emrJSON)
}

func (c *EMRChaincode) GetAllRecordsForPatient(ctx contractapi.TransactionContextInterface, patientID string) ([]*EMR, error) {
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
		return nil, err
	}
	defer resultsIterator.Close()

	var emrs []*EMR
	var skipped bool
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var emr EMR
		err = json.Unmarshal(queryResponse.Value, &emr)
		if err != nil {
			return nil, err
		}

		// Access control checks
		if (role == "patient" && clientID != emr.PatientID) ||
			(role == "doctor" && clientID != emr.DoctorID && !slices.Contains(emr.SharedWithDoctors, clientID)) ||
			(role == "hospital" && clientID != emr.DoctorID && !slices.Contains(emr.SharedWithHospitals, clientID)) {
			skipped = true
			continue // Skip records that the client is not authorized to access
		}

		emrs = append(emrs, &emr)
	}

	// Check if emrs is empty and if any records were skipped
	if len(emrs) == 0 && skipped {
		return nil, fmt.Errorf("records found for patient %s, but permission denied", patientID)
	}

	return emrs, nil
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
