package medplum

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"

	cr "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
)

func TestUpdateResource_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		if r.URL.Path != "/fhir/R4/Patient/123" {
			t.Errorf("Expected path /fhir/R4/Patient/123, got %s", r.URL.Path)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/fhir+json" {
			t.Errorf("Expected Content-Type application/fhir+json, got %s", contentType)
		}

		// Read request body to ensure it's valid JSON
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
		}

		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Errorf("Request body is not valid JSON: %v", err)
		}

		w.Header().Set("Content-Type", "application/fhir+json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"resourceType": "Patient",
			"id":           "123",
			"name": []map[string]interface{}{
				{
					"given":  []string{"Jane"},
					"family": "Doe",
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
			Patient: &patient_go_proto.Patient{
				Id: &datatypes_go_proto.Id{Value: "123"},
				Name: []*datatypes_go_proto.HumanName{
					{
						Given: []*datatypes_go_proto.String{
							{Value: "Jane"},
						},
						Family: &datatypes_go_proto.String{Value: "Doe"},
					},
				},
			},
		},
	}

	result, err := m.UpdateResource(nil, "123", resource)
	if err != nil {
		t.Fatalf("UpdateResource failed: %v", err)
	}

	if len(result.RawHTTPResponses) != 1 {
		t.Errorf("Expected 1 RawHTTPResponse, got %d", len(result.RawHTTPResponses))
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.RawHTTPResponses[0].StatusCode)
	}

	patient := result.ContainedResource.GetPatient()
	if patient == nil {
		t.Fatal("Expected Patient in result")
	}

	if patient.GetId().GetValue() != "123" {
		t.Errorf("Expected patient ID '123', got '%s'", patient.GetId().GetValue())
	}
}

func TestUpdateResource_NilResource(t *testing.T) {
	m := &Medplum{
		opts: &Options{
			MedplumURL: "http://localhost",
			Timezone:   "UTC",
		},
	}

	_, err := m.UpdateResource(nil, "123", nil)
	if err == nil {
		t.Fatal("Expected error for nil resource")
	}

	if err != ErrResourceCannotBeNil {
		t.Errorf("Expected ErrResourceCannotBeNil, got: %v", err)
	}
}

func TestUpdateResource_EmptyContainedResource(t *testing.T) {
	m := &Medplum{
		opts: &Options{
			MedplumURL: "http://localhost",
			Timezone:   "UTC",
		},
	}

	resource := &cr.ContainedResource{}

	_, err := m.UpdateResource(nil, "123", resource)
	if err == nil {
		t.Fatal("Expected error for empty ContainedResource")
	}

	if err != ErrInvalidResource {
		t.Errorf("Expected ErrInvalidResource, got: %v", err)
	}
}

func TestUpdateResource_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/fhir+json")
		w.WriteHeader(http.StatusNotFound)

		response := map[string]interface{}{
			"resourceType": "OperationOutcome",
			"issue": []map[string]interface{}{
				{
					"severity": "error",
					"code":     "not-found",
					"details": map[string]interface{}{
						"text": "Resource not found",
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
			Patient: &patient_go_proto.Patient{
				Id: &datatypes_go_proto.Id{Value: "nonexistent"},
			},
		},
	}

	result, err := m.UpdateResource(nil, "nonexistent", resource)
	if err != nil {
		t.Fatalf("UpdateResource should not return error for 404: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}

func TestUpdateResource_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/fhir+json")
		w.WriteHeader(http.StatusConflict)

		response := map[string]interface{}{
			"resourceType": "OperationOutcome",
			"issue": []map[string]interface{}{
				{
					"severity": "error",
					"code":     "conflict",
					"details": map[string]interface{}{
						"text": "Resource version conflict",
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
			Patient: &patient_go_proto.Patient{
				Id: &datatypes_go_proto.Id{Value: "123"},
			},
		},
	}

	result, err := m.UpdateResource(nil, "123", resource)
	if err != nil {
		t.Fatalf("UpdateResource should not return error for HTTP errors: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}

func TestUpdateResource_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/fhir+json")
		w.WriteHeader(http.StatusBadRequest)

		response := map[string]interface{}{
			"resourceType": "OperationOutcome",
			"issue": []map[string]interface{}{
				{
					"severity": "error",
					"code":     "invalid",
					"details": map[string]interface{}{
						"text": "Validation failed: missing required field",
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
			Patient: &patient_go_proto.Patient{
				Id: &datatypes_go_proto.Id{Value: "123"},
			},
		},
	}

	result, err := m.UpdateResource(nil, "123", resource)
	if err != nil {
		t.Fatalf("UpdateResource should not return error for HTTP errors: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}
