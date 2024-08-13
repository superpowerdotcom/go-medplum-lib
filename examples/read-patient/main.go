package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	dt "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	cr "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"
	"github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"

	"github.com/superpowerdotcom/go-medplum-lib"
)

type ResponseJSON struct {
	ID string `json:"id"`
}

func main() {
	m, err := medplum.New(&medplum.Options{
		MedplumURL:   "http://localhost:8103",
		ClientID:     "3008218e-5de9-4398-a987-ca393e3e64b0",
		ClientSecret: "1b6b7708423fa6cc589d2996e40d35bc2ba38d6af366e16660bcfcecb5438896",
		TokenURL:     "http://localhost:8103/oauth2/token",
	})

	if err != nil {
		fmt.Println("unable to create medplum client: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("Successfully authenticated")

	patientID, err := createPatient(m)
	if err != nil {
		fmt.Println("Unable to create patient resource: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("Created a patient with ID: " + patientID)

	// Now read the patient
	result, err := m.ReadResource(patientID, codes_go_proto.ResourceTypeCode_PATIENT)
	if err != nil {
		fmt.Println("Unable to read patient resource: " + err.Error())
		os.Exit(1)
	}

	// Did it succeed?
	if result.RawHTTPResponse.StatusCode < 200 || result.RawHTTPResponse.StatusCode >= 300 {
		fmt.Printf("Unable to read patient - unexpected response status code: %d\n", result.RawHTTPResponse.StatusCode)
		os.Exit(1)
	}

	// Print patient info
	patientResource := result.ContainedResource.GetPatient()

	if patientResource == nil {
		fmt.Println("Unexpected: patient resource is nil")
		os.Exit(1)
	}

	if len(patientResource.Name) < 1 {
		fmt.Println("Unexpected: patient resource has no name")
		os.Exit(1)
	}

	fmt.Printf("[read] Patient Name: %s\n", patientResource.Name[0].Text.Value)

	// Uncomment to see the raw result
	//fmt.Printf("[raw result] ")
	//spew.Dump(result) // Library for pretty printing complex data structures

	os.Exit(0)
}

func createPatient(m *medplum.Medplum) (string, error) {
	patient := &patient_go_proto.Patient{
		Name: []*dt.HumanName{
			{
				Text: &dt.String{Value: "Foo Bar"},
			},
		},
	}

	// Put the patient inside of a ContainedResource
	patientCR := &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Patient{
			Patient: patient,
		},
	}

	// Create it via medplum client
	result, err := m.CreateResource(context.Background(), patientCR)
	if err != nil {
		return "", errors.New("Unable to create patient resource: " + err.Error())

	}

	if result.RawHTTPResponse.StatusCode < 200 || result.RawHTTPResponse.StatusCode >= 300 {
		return "", fmt.Errorf("failed to create patient resource (StatusCode: %d)", result.RawHTTPResponse.StatusCode)
	}

	patientResource := result.ContainedResource.GetPatient()
	if patientResource == nil {
		return "", errors.New("unexpected: returned patient resource is nil")
	}

	return patientResource.Id.Value, nil
}
