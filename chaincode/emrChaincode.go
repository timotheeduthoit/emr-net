package main

import (
	"encoding/json"
	"fmt"
	"time"

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

	if role == "patient" && clientID != emr.PatientID {
        return nil, fmt.Errorf("this patient is not authorized to read this record")
	}

	if role == "doctor" {
		if clientID != emr.DoctorID && !contains(emr.SharedWithDoctors, clientID) {
            return nil, fmt.Errorf("this doctor is not authorized to read this record")
		}
	}

	if role == "hospital" {
		if clientID != emr.DoctorID && !contains(emr.SharedWithHospitals, clientID) {
            return nil, fmt.Errorf("this hospital is not authorized to read this record")
		}
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

    if role == "patient" && clientID != emr.PatientID {
        return fmt.Errorf("patients can only share their own records")
	}

    if role == "doctor" {
        if clientID != emr.DoctorID && !contains(emr.SharedWithDoctors, clientID) {
            return fmt.Errorf("this doctor is not authorized to share this record")
        }
    }

    if role == "hospital" {
        if clientID != emr.DoctorID && !contains(emr.SharedWithHospitals, clientID) {
            return fmt.Errorf("this hospital is not authorized to share this record")
        }
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

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
