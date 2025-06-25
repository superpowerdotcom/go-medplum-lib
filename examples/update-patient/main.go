package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	dt "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	cr "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"

	"github.com/superpowerdotcom/go-medplum-lib"
)

func main() {
	m, err := medplum.New(&medplum.Options{
		MedplumURL:   "http://localhost:8103",
		ClientID:     "3008218e-5de9-4398-a987-ca393e3e64b0",
		ClientSecret: "1b6b7708423fa6cc589d2996e40d35bc2ba38d6af366e16660bcfcecb5438896",
	})

	if err != nil {
		fmt.Println("unable to create medplum client: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("Successfully authenticated")

	patientResource := &patient_go_proto.Patient{
		Name: []*dt.HumanName{
			{
				Text: &dt.String{Value: "Update Example 1"},
			},
		},
	}

	patientID, patientName, err := createPatient(m, patientResource)
	if err != nil {
		fmt.Println("Unable to create patient resource: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("[created] Patient ID: " + patientID)
	fmt.Println("[created] Patient Name: " + patientName)

	// Update the patient
	// Must include ID of the patient we are updating in both the URL and body
	patientResource.Id = &dt.Id{Value: patientID}
	patientResource.Name = []*dt.HumanName{
		{
			Text: &dt.String{Value: "Update Example 2"},
		},
	}

	// Add the patient to a ContainedResource
	patientCR := &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Patient{
			Patient: patientResource,
		},
	}

	result, err := m.UpdateResource(nil, patientID, patientCR)
	if err != nil {
		fmt.Println("Unexpected error updating resource: " + err.Error())
		os.Exit(1)
	}

	if result.RawHTTPResponse.StatusCode < 200 || result.RawHTTPResponse.StatusCode >= 300 {
		fmt.Printf("Unable to update user (received %d status code)\n", result.RawHTTPResponse.StatusCode)
		spew.Dump(result)
		os.Exit(1)
	}

	returnedPatient := result.ContainedResource.GetPatient()
	if returnedPatient == nil {
		fmt.Println("Unexpected: patient resource is nil")
		os.Exit(1)
	}

	fmt.Println("[returned] Patient ID: " + returnedPatient.Id.Value)
	fmt.Println("[returned] Patient Name: " + returnedPatient.Name[0].Text.Value)

	os.Exit(0)
}

// Returns id, name, error
func createPatient(m *medplum.Medplum, patient *patient_go_proto.Patient) (string, string, error) {
	// Put the patient inside of a ContainedResource
	patientCR := &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Patient{
			Patient: patient,
		},
	}

	// Create it via medplum client
	result, err := m.CreateResource(nil, patientCR)
	if err != nil {
		return "", "", errors.New("Unable to create patient resource: " + err.Error())

	}

	if result.RawHTTPResponse.StatusCode < 200 || result.RawHTTPResponse.StatusCode >= 300 {
		return "", "", fmt.Errorf("failed to create patient resource (StatusCode: %d)", result.RawHTTPResponse.StatusCode)
	}

	patientResource := result.ContainedResource.GetPatient()
	if patientResource == nil {
		return "", "", errors.New("unexpected: returned patient resource is nil")
	}

	return patientResource.Id.Value, patientResource.Name[0].Text.Value, nil
}
