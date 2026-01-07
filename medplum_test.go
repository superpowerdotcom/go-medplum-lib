package medplum

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
)

// TestSearchPagination tests that Search() automatically follows pagination links
func TestSearchPagination(t *testing.T) {
	// Track how many requests were made
	requestCount := 0

	// We need a pointer to store the server URL for use in the handler
	var serverURL string

	// Create a test server that returns paginated results
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		w.Header().Set("Content-Type", "application/fhir+json")

		// First page - return 2 entries with a "next" link
		if requestCount == 1 {
			response := map[string]interface{}{
				"resourceType": "Bundle",
				"type":         "searchset",
				"entry": []map[string]interface{}{
					{
						"fullUrl": "http://example.com/Patient/1",
						"resource": map[string]interface{}{
							"resourceType": "Patient",
							"id":           "1",
						},
					},
					{
						"fullUrl": "http://example.com/Patient/2",
						"resource": map[string]interface{}{
							"resourceType": "Patient",
							"id":           "2",
						},
					},
				},
				"link": []map[string]interface{}{
					{
						"relation": "self",
						"url":      r.URL.String(),
					},
					{
						"relation": "next",
						"url":      fmt.Sprintf("%s/fhir/R4/Patient?_page=2", serverURL),
					},
				},
			}

			json.NewEncoder(w).Encode(response)
			return
		}

		// Second page - return 1 entry with no "next" link
		if requestCount == 2 {
			response := map[string]interface{}{
				"resourceType": "Bundle",
				"type":         "searchset",
				"entry": []map[string]interface{}{
					{
						"fullUrl": "http://example.com/Patient/3",
						"resource": map[string]interface{}{
							"resourceType": "Patient",
							"id":           "3",
						},
					},
				},
				"link": []map[string]interface{}{
					{
						"relation": "self",
						"url":      r.URL.String(),
					},
				},
			}

			json.NewEncoder(w).Encode(response)
			return
		}

		// Unexpected request
		t.Errorf("Unexpected request count: %d", requestCount)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Set the server URL after server is created
	serverURL = server.URL

	// Create Medplum client pointing to test server
	m := &Medplum{
		client: server.Client(),
		opts: &Options{
			MedplumURL: server.URL,
			Timezone:   "UTC",
		},
	}

	// Execute search
	result, err := m.Search(nil, codes_go_proto.ResourceTypeCode_PATIENT, "")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Verify we made 2 requests (followed pagination)
	if requestCount != 2 {
		t.Errorf("Expected 2 requests, got %d", requestCount)
	}

	// Verify we got all 3 entries combined
	bundle := result.ContainedResource.GetBundle()
	if bundle == nil {
		t.Fatal("Expected bundle in result")
	}

	if len(bundle.Entry) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(bundle.Entry))
	}

	// Verify we collected both HTTP responses
	if len(result.RawHTTPResponses) != 2 {
		t.Errorf("Expected 2 RawHTTPResponses, got %d", len(result.RawHTTPResponses))
	}
}

// TestSearchNoPagination tests that Search() works correctly when there's no pagination
func TestSearchNoPagination(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		w.Header().Set("Content-Type", "application/fhir+json")

		response := map[string]interface{}{
			"resourceType": "Bundle",
			"type":         "searchset",
			"entry": []map[string]interface{}{
				{
					"fullUrl": "http://example.com/Patient/1",
					"resource": map[string]interface{}{
						"resourceType": "Patient",
						"id":           "1",
					},
				},
			},
			"link": []map[string]interface{}{
				{
					"relation": "self",
					"url":      r.URL.String(),
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

	result, err := m.Search(nil, codes_go_proto.ResourceTypeCode_PATIENT, "name=Test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should only make 1 request
	if requestCount != 1 {
		t.Errorf("Expected 1 request, got %d", requestCount)
	}

	bundle := result.ContainedResource.GetBundle()
	if bundle == nil {
		t.Fatal("Expected bundle in result")
	}

	if len(bundle.Entry) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(bundle.Entry))
	}

	if len(result.RawHTTPResponses) != 1 {
		t.Errorf("Expected 1 RawHTTPResponse, got %d", len(result.RawHTTPResponses))
	}
}

// TestSearchErrorResponse tests that Search() handles error responses correctly
func TestSearchErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/fhir+json")

		response := map[string]interface{}{
			"resourceType": "OperationOutcome",
			"issue": []map[string]interface{}{
				{
					"severity": "error",
					"code":     "invalid",
					"details": map[string]interface{}{
						"text": "Invalid search parameter",
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

	result, err := m.Search(nil, codes_go_proto.ResourceTypeCode_PATIENT, "invalid=param")
	if err != nil {
		t.Fatalf("Search should not return error for HTTP errors: %v", err)
	}

	// Should return result with error status
	if len(result.RawHTTPResponses) == 0 {
		t.Fatal("Expected at least one RawHTTPResponse")
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}

// TestSearchEmptyResult tests that Search() handles empty results correctly
func TestSearchEmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/fhir+json")

		response := map[string]interface{}{
			"resourceType": "Bundle",
			"type":         "searchset",
			"total":        0,
			"entry":        []map[string]interface{}{},
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

	result, err := m.Search(nil, codes_go_proto.ResourceTypeCode_PATIENT, "name=NonExistent")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	bundle := result.ContainedResource.GetBundle()
	if bundle == nil {
		t.Fatal("Expected bundle in result")
	}

	if len(bundle.Entry) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(bundle.Entry))
	}
}

// TestRawHTTPResponsesSlice tests that non-search methods also return slice
func TestRawHTTPResponsesSlice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/fhir+json")

		response := map[string]interface{}{
			"resourceType": "Patient",
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

	result, err := m.ReadResource(nil, "123", codes_go_proto.ResourceTypeCode_PATIENT)
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	// Should have exactly 1 response in slice
	if len(result.RawHTTPResponses) != 1 {
		t.Errorf("Expected 1 RawHTTPResponse, got %d", len(result.RawHTTPResponses))
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}
