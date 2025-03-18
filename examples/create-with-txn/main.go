package main

import (
	"fmt"
	"os"

	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	dt_gp "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	bcr_gp "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	cp_gp "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/care_plan_go_proto"
	g_gp "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/goal_go_proto"
	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"
	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/valuesets_go_proto"
	"github.com/pkg/errors"

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

	// Create a dummy patient
	dummyPatient, err := createDummyPatient(m, "foo", "bar", "foo@bar.com")
	if err != nil {
		fmt.Println("Unable to create dummy patient: " + err.Error())
		os.Exit(1)
	}

	// Create Goals
	goal1 := &g_gp.Goal{
		Id: &dt_gp.Id{Value: "temporary-goal-1"},
		Subject: &dt_gp.Reference{
			Reference: &dt_gp.Reference_PatientId{
				PatientId: &dt_gp.ReferenceId{Value: dummyPatient.Id.Value},
			},
		},
		LifecycleStatus: &g_gp.Goal_LifecycleStatusCode{
			Value: codes_go_proto.GoalLifecycleStatusCode_ACTIVE,
		},
		Description: &dt_gp.CodeableConcept{
			Text: &dt_gp.String{Value: "Increase daily activity"},
		},
	}

	goal2 := &g_gp.Goal{
		Id: &dt_gp.Id{Value: "temporary-goal-2"},
		Subject: &dt_gp.Reference{
			Reference: &dt_gp.Reference_PatientId{
				PatientId: &dt_gp.ReferenceId{Value: dummyPatient.Id.Value},
			},
		},
		LifecycleStatus: &g_gp.Goal_LifecycleStatusCode{
			Value: codes_go_proto.GoalLifecycleStatusCode_ACTIVE,
		},
		Description: &dt_gp.CodeableConcept{
			Text: &dt_gp.String{Value: "Improve nutrition"},
		},
	}

	// Create CarePlan referencing the Goals
	carePlan := &cp_gp.CarePlan{
		Id: &dt_gp.Id{Value: "temporary-careplan"},
		Subject: &dt_gp.Reference{
			Reference: &dt_gp.Reference_PatientId{
				PatientId: &dt_gp.ReferenceId{Value: dummyPatient.Id.Value},
			},
		},
		Status: &cp_gp.CarePlan_StatusCode{
			Value: codes_go_proto.RequestStatusCode_DRAFT,
		},
		Intent: &cp_gp.CarePlan_IntentCode{
			Value: valuesets_go_proto.CarePlanIntentValueSet_PLAN,
		},
		Goal: []*dt_gp.Reference{
			{Reference: &dt_gp.Reference_GoalId{GoalId: &dt_gp.ReferenceId{Value: "temporary-goal-1"}}},
			{Reference: &dt_gp.Reference_GoalId{GoalId: &dt_gp.ReferenceId{Value: "temporary-goal-2"}}},
		},
	}

	// Construct Bundle
	bundle := &bcr_gp.Bundle{
		Type: &bcr_gp.Bundle_TypeCode{
			Value: codes_go_proto.BundleTypeCode_TRANSACTION,
		},
		Entry: []*bcr_gp.Bundle_Entry{
			{
				FullUrl: &dt_gp.Uri{Value: "temporary-goal-1"},
				Resource: &bcr_gp.ContainedResource{
					OneofResource: &bcr_gp.ContainedResource_Goal{
						Goal: goal1,
					},
				},
				Request: &bcr_gp.Bundle_Entry_Request{
					Method: &bcr_gp.Bundle_Entry_Request_MethodCode{
						Value: codes_go_proto.HTTPVerbCode_POST,
					},
					Url: &dt_gp.Uri{Value: "Goal"},
				},
			},
			{
				FullUrl: &dt_gp.Uri{Value: "temporary-goal-2"},
				Resource: &bcr_gp.ContainedResource{
					OneofResource: &bcr_gp.ContainedResource_Goal{
						Goal: goal2,
					},
				},
				Request: &bcr_gp.Bundle_Entry_Request{
					Method: &bcr_gp.Bundle_Entry_Request_MethodCode{
						Value: codes_go_proto.HTTPVerbCode_POST,
					},
					Url: &dt_gp.Uri{Value: "Goal"},
				},
			},
			{
				FullUrl: &dt_gp.Uri{Value: "temporary-careplan"},
				Resource: &bcr_gp.ContainedResource{
					OneofResource: &bcr_gp.ContainedResource_CarePlan{
						CarePlan: carePlan,
					},
				},
				Request: &bcr_gp.Bundle_Entry_Request{
					Method: &bcr_gp.Bundle_Entry_Request_MethodCode{
						Value: codes_go_proto.HTTPVerbCode_POST,
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

func createDummyPatient(m *medplum.Medplum, firstName, lastName, email string) (*patient_go_proto.Patient, error) {
	// Create a patient
	patient := &patient_go_proto.Patient{
		Name: []*dt_gp.HumanName{
			{
				Text: &dt_gp.String{Value: firstName + " " + lastName},
			},
		},
		Telecom: []*dt_gp.ContactPoint{
			{
				System: &dt_gp.ContactPoint_SystemCode{
					Value: codes_go_proto.ContactPointSystemCode_EMAIL,
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

	if result == nil {
		return nil, errors.Wrap(err, "Result is nil - something went wrong")
	}

	if result.RawHTTPResponse.StatusCode < 200 || result.RawHTTPResponse.StatusCode >= 300 {
		fmt.Printf("Unable to create user (received %d status code)\n", result.RawHTTPResponse.StatusCode)
		return nil, fmt.Errorf("received non-2xx status code during create: %d", result.RawHTTPResponse.StatusCode)
	}

	if result.ContainedResource == nil || result.ContainedResource.GetPatient() == nil {
		return nil, errors.New("unexpected containedResponse or patient resource is nil")
	}

	return result.ContainedResource.GetPatient(), nil
}
