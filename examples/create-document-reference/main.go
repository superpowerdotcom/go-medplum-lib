package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	dt "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	cr "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/document_reference_go_proto"
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

	patientID, err := createPatientResource(m)
	if err != nil {
		fmt.Println("Unable to create patient resource: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("[created] Patient ID: " + patientID)

	// Create a binary resource
	binaryID, binaryURL, err := createBinaryResource(m)
	if err != nil {
		fmt.Println("Unable to create binary resource: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("[created] Binary ID: " + binaryID)
	fmt.Println("[created] Binary URL: " + binaryURL)

	// Now create a document reference (it is a bit of a slog)
	documentReference := &document_reference_go_proto.DocumentReference{
		Subject: &dt.Reference{
			Reference: &dt.Reference_PatientId{
				PatientId: &dt.ReferenceId{Value: patientID},
			},
		},
		Status: &document_reference_go_proto.DocumentReference_StatusCode{
			Value: codes_go_proto.DocumentReferenceStatusCode_CURRENT,
		},
		Type: &dt.CodeableConcept{
			Coding: []*dt.Coding{
				{
					System:  &dt.Uri{Value: "http://loinc.org"},
					Code:    &dt.Code{Value: "34108-1"},
					Display: &dt.String{Value: "Outpatient Note"},
				},
			},
		},
		Content: []*document_reference_go_proto.DocumentReference_Content{
			{
				Attachment: &dt.Attachment{
					ContentType: &dt.Attachment_ContentTypeCode{Value: "image/png"},
					Url:         &dt.Url{Value: "Binary/" + binaryID},
					Title:       &dt.String{Value: "bees.png"},
				},
			},
		},
	}

	// Put it in a contained resource
	documentReferenceCR := &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_DocumentReference{
			DocumentReference: documentReference,
		},
	}

	// Create the document reference resource
	result, err := m.CreateResource(context.Background(), documentReferenceCR)
	if err != nil {
		fmt.Println("Unable to create document reference resource: " + err.Error())
		os.Exit(1)
	}

	spew.Dump(result)
}

func createPatientResource(m *medplum.Medplum) (string, error) {
	patient := &patient_go_proto.Patient{
		Name: []*dt.HumanName{
			{
				Text: &dt.String{Value: "Document Reference Example"},
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

func createBinaryResource(m *medplum.Medplum) (string, string, error) {
	// Read file contents
	data, err := os.ReadFile("bees.png")
	if err != nil {
		return "", "", errors.New("unable to read file: " + err.Error())
	}

	// Create binary resource with convenience method
	result, err := m.CreateBinaryResource(data, "image/png")
	if err != nil {
		return "", "", errors.New("unable to create binary resource: " + err.Error())
	}

	// Did the create succeed?
	if result.RawHTTPResponse.StatusCode < 200 || result.RawHTTPResponse.StatusCode >= 300 {
		return "", "", fmt.Errorf("unexpected response status code: %d", result.RawHTTPResponse.StatusCode)
	}

	// Binary might not be able to get unmarshalled into an FHIR resource, so
	// we'll check MapResource instead.
	binaryIDInterface, ok := result.MapResource["id"]
	if !ok {
		return "", "", errors.New("unexpected 'id' not contained in MapResource")
	}

	binaryID, ok := binaryIDInterface.(string)
	if !ok {
		return "", "", errors.New("unable to type assert id to a string")
	}

	binaryURLInterface, ok := result.MapResource["url"]
	if !ok {
		return "", "", errors.New("unexpected 'url' not contained in MapResource")
	}

	binaryURL, ok := binaryURLInterface.(string)
	if !ok {
		return "", "", errors.New("unable to type assert URL to a string")
	}

	return binaryID, binaryURL, nil
}
