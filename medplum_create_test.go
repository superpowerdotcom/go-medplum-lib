package medplum

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"

	cr "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
)

func TestCreateResource_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.URL.Path != "/fhir/R4/Patient" {
			t.Errorf("Expected path /fhir/R4/Patient, got %s", r.URL.Path)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/fhir+json" {
			t.Errorf("Expected Content-Type application/fhir+json, got %s", contentType)
		}

		w.Header().Set("Content-Type", "application/fhir+json")
		w.WriteHeader(http.StatusCreated)

		response := map[string]interface{}{
			"resourceType": "Patient",
			"id":           "new-patient-123",
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	m := &Medplum{
		client: server.Client(),
		opts: &Options{
			MedplumURL: server.URL,
			Timezone:   "UTC",
		},
	}

	resource := &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Patient{
			Patient: &patient_go_proto.Patient{
				Name: []*datatypes_go_proto.HumanName{
					{
						Given: []*datatypes_go_proto.String{
							{Value: "John"},
						},
						Family: &datatypes_go_proto.String{Value: "Doe"},
					},
				},
			},
		},
	}

	result, err := m.CreateResource(nil, resource)
	if err != nil {
		t.Fatalf("CreateResource failed: %v", err)
	}

	if len(result.RawHTTPResponses) != 1 {
		t.Errorf("Expected 1 RawHTTPResponse, got %d", len(result.RawHTTPResponses))
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", result.RawHTTPResponses[0].StatusCode)
	}

	patient := result.ContainedResource.GetPatient()
	if patient == nil {
		t.Fatal("Expected Patient in result")
	}

	if patient.GetId().GetValue() != "new-patient-123" {
		t.Errorf("Expected patient ID 'new-patient-123', got '%s'", patient.GetId().GetValue())
	}
}

func TestCreateResource_NilResource(t *testing.T) {
	m := &Medplum{
		opts: &Options{
			MedplumURL: "http://localhost",
			Timezone:   "UTC",
		},
	}

	_, err := m.CreateResource(nil, nil)
	if err == nil {
		t.Fatal("Expected error for nil resource")
	}

	if err != ErrResourceCannotBeNil {
		t.Errorf("Expected ErrResourceCannotBeNil, got: %v", err)
	}
}

func TestCreateResource_EmptyContainedResource(t *testing.T) {
	m := &Medplum{
		opts: &Options{
			MedplumURL: "http://localhost",
			Timezone:   "UTC",
		},
	}

	resource := &cr.ContainedResource{}

	_, err := m.CreateResource(nil, resource)
	if err == nil {
		t.Fatal("Expected error for empty ContainedResource")
	}

	if err != ErrInvalidResource {
		t.Errorf("Expected ErrInvalidResource, got: %v", err)
	}
}

func TestCreateResource_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/fhir+json")
		w.WriteHeader(http.StatusInternalServerError)

		response := map[string]interface{}{
			"resourceType": "OperationOutcome",
			"issue": []map[string]interface{}{
				{
					"severity": "error",
					"code":     "exception",
					"details": map[string]interface{}{
						"text": "Internal server error",
					},
				},
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	m := &Medplum{
		client: server.Client(),
		opts: &Options{
			MedplumURL: server.URL,
			Timezone:   "UTC",
		},
	}

	resource := &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Patient{
			Patient: &patient_go_proto.Patient{},
		},
	}

	result, err := m.CreateResource(nil, resource)
	if err != nil {
		t.Fatalf("CreateResource should not return error for server errors: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}

func TestCreateBinaryResource_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.URL.Path != "/fhir/R4/Binary" {
			t.Errorf("Expected path /fhir/R4/Binary, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/fhir+json")
		w.WriteHeader(http.StatusCreated)

		response := map[string]interface{}{
			"resourceType": "Binary",
			"id":           "binary-123",
			"contentType":  "application/pdf",
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	m := &Medplum{
		client: server.Client(),
		opts: &Options{
			MedplumURL: server.URL,
			Timezone:   "UTC",
		},
	}

	testData := []byte("test binary data")

	result, err := m.CreateBinaryResource(nil, testData, "application/pdf")
	if err != nil {
		t.Fatalf("CreateBinaryResource failed: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", result.RawHTTPResponses[0].StatusCode)
	}

	binary := result.ContainedResource.GetBinary()
	if binary == nil {
		t.Fatal("Expected Binary in result")
	}

	if binary.GetId().GetValue() != "binary-123" {
		t.Errorf("Expected binary ID 'binary-123', got '%s'", binary.GetId().GetValue())
	}
}

func TestCreateBinaryResource_NilData(t *testing.T) {
	m := &Medplum{
		opts: &Options{
			MedplumURL: "http://localhost",
			Timezone:   "UTC",
		},
	}

	_, err := m.CreateBinaryResource(nil, nil, "application/pdf")
	if err == nil {
		t.Fatal("Expected error for nil data")
	}
}

func TestCreateBinaryResource_EmptyContentType(t *testing.T) {
	m := &Medplum{
		opts: &Options{
			MedplumURL: "http://localhost",
			Timezone:   "UTC",
		},
	}

	_, err := m.CreateBinaryResource(nil, []byte("data"), "")
	if err == nil {
		t.Fatal("Expected error for empty contentType")
	}
}
