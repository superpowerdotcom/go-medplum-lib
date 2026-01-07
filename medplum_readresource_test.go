package medplum

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
)

func TestReadResource_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		if r.URL.Path != "/fhir/R4/Patient/123" {
			t.Errorf("Expected path /fhir/R4/Patient/123, got %s", r.URL.Path)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/fhir+json" {
			t.Errorf("Expected Content-Type application/fhir+json, got %s", contentType)
		}

		w.Header().Set("Content-Type", "application/fhir+json")

		response := map[string]interface{}{
			"resourceType": "Patient",
			"id":           "123",
			"name": []map[string]interface{}{
				{
					"given":  []string{"John"},
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

	result, err := m.ReadResource(nil, "123", codes_go_proto.ResourceTypeCode_PATIENT)
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
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

func TestReadResource_EmptyID(t *testing.T) {
	m := &Medplum{
		opts: &Options{
			MedplumURL: "http://localhost",
			Timezone:   "UTC",
		},
	}

	_, err := m.ReadResource(nil, "", codes_go_proto.ResourceTypeCode_PATIENT)
	if err == nil {
		t.Fatal("Expected error for empty ID")
	}
}

func TestReadResource_NotFound(t *testing.T) {
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

	result, err := m.ReadResource(nil, "nonexistent", codes_go_proto.ResourceTypeCode_PATIENT)
	if err != nil {
		t.Fatalf("ReadResource should not return error for 404: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}

func TestReadResource_DifferentResourceTypes(t *testing.T) {
	testCases := []struct {
		name         string
		resourceType codes_go_proto.ResourceTypeCode_Value
		expectedPath string
	}{
		{"Patient", codes_go_proto.ResourceTypeCode_PATIENT, "/fhir/R4/Patient/123"},
		{"Practitioner", codes_go_proto.ResourceTypeCode_PRACTITIONER, "/fhir/R4/Practitioner/123"},
		{"Organization", codes_go_proto.ResourceTypeCode_ORGANIZATION, "/fhir/R4/Organization/123"},
		{"Observation", codes_go_proto.ResourceTypeCode_OBSERVATION, "/fhir/R4/Observation/123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.expectedPath {
					t.Errorf("Expected path %s, got %s", tc.expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/fhir+json")

				response := map[string]interface{}{
					"resourceType": tc.name,
					"id":           "123",
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

			result, err := m.ReadResource(nil, "123", tc.resourceType)
			if err != nil {
				t.Fatalf("ReadResource failed: %v", err)
			}

			if result.RawHTTPResponses[0].StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", result.RawHTTPResponses[0].StatusCode)
			}
		})
	}
}

func TestReadResourceHistory_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		if r.URL.Path != "/fhir/R4/Patient/123/_history" {
			t.Errorf("Expected path /fhir/R4/Patient/123/_history, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/fhir+json")

		response := map[string]interface{}{
			"resourceType": "Bundle",
			"type":         "history",
			"entry": []map[string]interface{}{
				{
					"fullUrl": "http://example.com/Patient/123/_history/2",
					"resource": map[string]interface{}{
						"resourceType": "Patient",
						"id":           "123",
						"meta": map[string]interface{}{
							"versionId": "2",
						},
					},
				},
				{
					"fullUrl": "http://example.com/Patient/123/_history/1",
					"resource": map[string]interface{}{
						"resourceType": "Patient",
						"id":           "123",
						"meta": map[string]interface{}{
							"versionId": "1",
						},
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

	result, err := m.ReadResourceHistory(nil, "123", codes_go_proto.ResourceTypeCode_PATIENT)
	if err != nil {
		t.Fatalf("ReadResourceHistory failed: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.RawHTTPResponses[0].StatusCode)
	}

	bundle := result.ContainedResource.GetBundle()
	if bundle == nil {
		t.Fatal("Expected Bundle in result")
	}

	if len(bundle.Entry) != 2 {
		t.Errorf("Expected 2 history entries, got %d", len(bundle.Entry))
	}
}

func TestReadResourceHistory_EmptyID(t *testing.T) {
	m := &Medplum{
		opts: &Options{
			MedplumURL: "http://localhost",
			Timezone:   "UTC",
		},
	}

	_, err := m.ReadResourceHistory(nil, "", codes_go_proto.ResourceTypeCode_PATIENT)
	if err == nil {
		t.Fatal("Expected error for empty ID")
	}
}
