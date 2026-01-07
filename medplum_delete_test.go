package medplum

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
)

func TestDeleteResource_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		if r.URL.Path != "/fhir/R4/Patient/123" {
			t.Errorf("Expected path /fhir/R4/Patient/123, got %s", r.URL.Path)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/fhir+json" {
			t.Errorf("Expected Content-Type application/fhir+json, got %s", contentType)
		}

		w.Header().Set("Content-Type", "application/fhir+json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"resourceType": "OperationOutcome",
			"issue": []map[string]interface{}{
				{
					"severity": "information",
					"code":     "informational",
					"details": map[string]interface{}{
						"text": "Resource deleted",
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

	result, err := m.DeleteResource(nil, "123", codes_go_proto.ResourceTypeCode_PATIENT)
	if err != nil {
		t.Fatalf("DeleteResource failed: %v", err)
	}

	if len(result.RawHTTPResponses) != 1 {
		t.Errorf("Expected 1 RawHTTPResponse, got %d", len(result.RawHTTPResponses))
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}

func TestDeleteResource_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	m := &Medplum{
		client: server.Client(),
		opts: &Options{
			MedplumURL: server.URL,
			Timezone:   "UTC",
		},
	}

	result, err := m.DeleteResource(nil, "123", codes_go_proto.ResourceTypeCode_PATIENT)
	if err != nil {
		t.Fatalf("DeleteResource failed: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}

func TestDeleteResource_NotFound(t *testing.T) {
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

	result, err := m.DeleteResource(nil, "nonexistent", codes_go_proto.ResourceTypeCode_PATIENT)
	if err != nil {
		t.Fatalf("DeleteResource should not return error for 404: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}

func TestDeleteResource_DifferentResourceTypes(t *testing.T) {
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
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE method, got %s", r.Method)
				}

				if r.URL.Path != tc.expectedPath {
					t.Errorf("Expected path %s, got %s", tc.expectedPath, r.URL.Path)
				}

				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			m := &Medplum{
				client: server.Client(),
				opts: &Options{
					MedplumURL: server.URL,
					Timezone:   "UTC",
				},
			}

			result, err := m.DeleteResource(nil, "123", tc.resourceType)
			if err != nil {
				t.Fatalf("DeleteResource failed: %v", err)
			}

			if result.RawHTTPResponses[0].StatusCode != http.StatusNoContent {
				t.Errorf("Expected status 204, got %d", result.RawHTTPResponses[0].StatusCode)
			}
		})
	}
}

func TestDeleteResource_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/fhir+json")
		w.WriteHeader(http.StatusForbidden)

		response := map[string]interface{}{
			"resourceType": "OperationOutcome",
			"issue": []map[string]interface{}{
				{
					"severity": "error",
					"code":     "forbidden",
					"details": map[string]interface{}{
						"text": "Access denied",
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

	result, err := m.DeleteResource(nil, "123", codes_go_proto.ResourceTypeCode_PATIENT)
	if err != nil {
		t.Fatalf("DeleteResource should not return error for HTTP errors: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}
