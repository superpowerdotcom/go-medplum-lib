package main

import (
	"context"
	"fmt"
	"os"

	dt "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	cr "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"

	"github.com/superpowerdotcom/go-medplum-lib"
)

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

	// Create a patient
	patient := &patient_go_proto.Patient{
		Id: &dt.Id{Value: "12345"},
		Name: []*dt.HumanName{
			{
				Text: &dt.String{Value: "Bat Man"},
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
	resp, err := m.CreateResource(context.Background(), patientCR)
	if err != nil {
		fmt.Println("Unable to create patient resource: " + err.Error())
		os.Exit(1)
	}

	fmt.Printf("Got response code: %d\n", resp.StatusCode)
}
