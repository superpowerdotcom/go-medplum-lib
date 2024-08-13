package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	dt "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	cr "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"

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

	fmt.Println("This is our patient ID: " + patientID)

	// TODO: Now get the patient by that ID
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
	resp, err := m.CreateResource(context.Background(), patientCR)
	if err != nil {
		return "", errors.New("Unable to create patient resource: " + err.Error())

	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("failed to create patient resource (StatusCode: %d)", resp.StatusCode)
	}

	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %s", err)
	}

	respJSON := &ResponseJSON{}

	if err := json.Unmarshal(respData, respJSON); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %s", err)
	}

	return respJSON.ID, nil
}
