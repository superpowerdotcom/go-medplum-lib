package main

import (
	"fmt"
	"os"

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

	// Create a patient
	patient := &patient_go_proto.Patient{
		Id: &dt.Id{Value: "12345"}, // Will be ignored by server and a new ID will be generated
		Name: []*dt.HumanName{
			{
				Text: &dt.String{Value: "Create Patient"},
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
	result, err := m.CreateResource(nil, patientCR)
	if err != nil {
		fmt.Println("Unable to create patient resource: " + err.Error())
		os.Exit(1)
	}

	if result == nil {
		fmt.Println("Result is nil")
		os.Exit(1)
	}

	if result.RawHTTPResponse.StatusCode < 200 || result.RawHTTPResponse.StatusCode >= 300 {
		fmt.Printf("Unable to create user (received %d status code)\n", result.RawHTTPResponse.StatusCode)
		os.Exit(1)
	}

	// This might be good enough but if you want to look further, you can check
	// the contents of the contained resource.

	// Check to see the type of the underlying resource
	switch res := result.ContainedResource.OneofResource.(type) {
	case *cr.ContainedResource_Patient:
		// This is a patient
		fmt.Println("[type switch] patient ID: " + res.Patient.Id.Value)
	case *cr.ContainedResource_OperationOutcome:
		// This is an operation outcome
		fmt.Println("[type switch] Operation Outcome issue: " + res.OperationOutcome.Issue[0].Details.String())
	default:
		fmt.Printf("Unknown resource type: %+v\n", res)
		os.Exit(1)
	}

	// Without a type switch, to _safely_ get access to the patient, do this:
	crPatient := result.ContainedResource.GetPatient()
	if crPatient == nil {
		fmt.Println("Contained resource is not a patient")
		os.Exit(1)
	}

	// You can now safely read patient data
	fmt.Printf("[safe] Patient ID: %s\n", crPatient.Id.Value)

	// This should be avoided - it has a very minor resource hit, as the patient
	// will be asserted again but more importantly, if result.ContainedResource
	// gets modified while we are reading it, it might return nil and cause
	// your program to panic.
	fmt.Printf("[unsafe] Patient ID: %s\n", result.ContainedResource.GetPatient().Id.Value)
}
