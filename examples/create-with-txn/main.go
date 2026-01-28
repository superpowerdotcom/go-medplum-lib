package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	c_gp "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	dt_gp "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	bcr_gp "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	cp_gp "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/care_plan_go_proto"
	g_gp "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/goal_go_proto"
	p_gp "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"
	vs_gp "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/valuesets_go_proto"

	"github.com/superpowerdotcom/go-medplum-lib"
)

// Transactions allow you to bundle multiple operations into a single, atomic
// request. If one of the operations fails, the entire transaction will fail and
// will be rolled back.
//
// Medplum will also automatically replace reference IDs but you MUST use the
// "urn:uuid:$uuid" format in FullURL and Reference_Uri fields.
//
// Read more about Medplum transactions here:
//
// https://www.medplum.com/docs/migration/migration-pipelines#using-transactions-for-data-integrity

func main() {
	m, err := medplum.New(&medplum.Options{
		MedplumURL:   "http://localhost:8103",
		ClientID:     "foo",
		ClientSecret: "bar",
	})

	if err != nil {
		fmt.Println("unable to create medplum client: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("Successfully authenticated")

	// Create a dummy patient
	dummyPatient, err := createDummyPatient(m, "foo", "bar", "foo@bar.com")
	if err != nil {
		fmt.Println("Unable to create dummy patient: " + err.Error())
		os.Exit(1)
	}

	// Create Goals
	goal1 := &g_gp.Goal{
		Subject: &dt_gp.Reference{
			Reference: &dt_gp.Reference_PatientId{
				PatientId: &dt_gp.ReferenceId{Value: dummyPatient.Id.Value},
			},
		},
		LifecycleStatus: &g_gp.Goal_LifecycleStatusCode{
			Value: c_gp.GoalLifecycleStatusCode_ACTIVE,
		},
		Description: &dt_gp.CodeableConcept{
			Text: &dt_gp.String{Value: "Increase daily activity"},
		},
	}

	goal2 := &g_gp.Goal{
		Subject: &dt_gp.Reference{
			Reference: &dt_gp.Reference_PatientId{
				PatientId: &dt_gp.ReferenceId{Value: dummyPatient.Id.Value},
			},
		},
		LifecycleStatus: &g_gp.Goal_LifecycleStatusCode{
			Value: c_gp.GoalLifecycleStatusCode_ACTIVE,
		},
		Description: &dt_gp.CodeableConcept{
			Text: &dt_gp.String{Value: "Improve nutrition"},
		},
	}

	// Create CarePlan referencing the Goals
	carePlan := &cp_gp.CarePlan{
		Subject: &dt_gp.Reference{
			Reference: &dt_gp.Reference_PatientId{
				PatientId: &dt_gp.ReferenceId{Value: dummyPatient.Id.Value},
			},
		},
		Status: &cp_gp.CarePlan_StatusCode{
			Value: c_gp.RequestStatusCode_DRAFT,
		},
		Intent: &cp_gp.CarePlan_IntentCode{
			Value: vs_gp.CarePlanIntentValueSet_PLAN,
		},
		Goal: []*dt_gp.Reference{
			// !!! IMPORTANT !!!
			//
			// 1. Medplum WILL replace the IDs for the references BUT you MUST
			//    use the "urn:uuid:$uuid" format for references.
			// 2. You MUST use Reference_Uri - if not, proto lib will add an
			//    additional "$Resource/" prefix to the reference (you will end
			//    up with "Goal/Goal/$uuid" instead of "Goal/$uuid").
			{Reference: &dt_gp.Reference_Uri{Uri: &dt_gp.String{Value: "urn:uuid:ddc3e8de-da12-42ad-831e-f659ef5af8f1"}}},
			{Reference: &dt_gp.Reference_Uri{Uri: &dt_gp.String{Value: "urn:uuid:ddc3e8de-da12-42ad-831e-f659ef5af8f2"}}},
		},
	}

	// Construct Bundle with urn:uuid references
	bundle := &bcr_gp.Bundle{
		Type: &bcr_gp.Bundle_TypeCode{
			Value: c_gp.BundleTypeCode_TRANSACTION,
		},
		Entry: []*bcr_gp.Bundle_Entry{
			{
				// URL will be replaced by Medplum with the real ID in refs
				FullUrl: &dt_gp.Uri{Value: "urn:uuid:ddc3e8de-da12-42ad-831e-f659ef5af8f1"},
				Resource: &bcr_gp.ContainedResource{
					OneofResource: &bcr_gp.ContainedResource_Goal{
						Goal: goal1,
					},
				},
				Request: &bcr_gp.Bundle_Entry_Request{
					Method: &bcr_gp.Bundle_Entry_Request_MethodCode{
						Value: c_gp.HTTPVerbCode_POST,
					},
					Url: &dt_gp.Uri{Value: "Goal"},
				},
			},
			{
				// URL will be replaced by Medplum with the real ID in refs
				FullUrl: &dt_gp.Uri{Value: "urn:uuid:ddc3e8de-da12-42ad-831e-f659ef5af8f2"},
				Resource: &bcr_gp.ContainedResource{
					OneofResource: &bcr_gp.ContainedResource_Goal{
						Goal: goal2,
					},
				},
				Request: &bcr_gp.Bundle_Entry_Request{
					Method: &bcr_gp.Bundle_Entry_Request_MethodCode{
						Value: c_gp.HTTPVerbCode_POST,
					},
					Url: &dt_gp.Uri{Value: "Goal"},
				},
			},
			{

				Resource: &bcr_gp.ContainedResource{
					OneofResource: &bcr_gp.ContainedResource_CarePlan{
						CarePlan: carePlan,
					},
				},
				Request: &bcr_gp.Bundle_Entry_Request{
					Method: &bcr_gp.Bundle_Entry_Request_MethodCode{
						Value: c_gp.HTTPVerbCode_POST,
					},
					Url: &dt_gp.Uri{Value: "CarePlan"},
				},
			},
		},
	}

	// Send bundle in contained resource
	result, err := m.ExecuteBatch(nil, &bcr_gp.ContainedResource{
		OneofResource: &bcr_gp.ContainedResource_Bundle{
			Bundle: bundle,
		},
	})
	if err != nil {
		fmt.Println("Unable to execute batch: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("Successfully created CarePlan with Goals")
	medplum.PrettyPrintResult(result)
}

func createDummyPatient(m *medplum.Medplum, firstName, lastName, email string) (*p_gp.Patient, error) {
	// Create a patient
	patient := &p_gp.Patient{
		Name: []*dt_gp.HumanName{
			{
				Text: &dt_gp.String{Value: firstName + " " + lastName},
			},
		},
		Telecom: []*dt_gp.ContactPoint{
			{
				System: &dt_gp.ContactPoint_SystemCode{
					Value: c_gp.ContactPointSystemCode_EMAIL,
				},
				Value: &dt_gp.String{Value: email},
			},
		},
	}

	// Put the patient inside of a ContainedResource
	patientCR := &bcr_gp.ContainedResource{
		OneofResource: &bcr_gp.ContainedResource_Patient{
			Patient: patient,
		},
	}

	// Create it via medplum client
	result, err := m.CreateResource(nil, patientCR)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create patient resource")
	}

	if result == nil || result.ContainedResource == nil || result.ContainedResource.GetPatient() == nil {
		return nil, errors.New("unexpected response - patient resource is nil")
	}

	return result.ContainedResource.GetPatient(), nil
}
