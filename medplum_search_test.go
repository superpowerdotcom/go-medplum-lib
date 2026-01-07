package medplum

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
)

func TestSearch_Pagination(t *testing.T) {
	requestCount := 0
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		w.Header().Set("Content-Type", "application/fhir+json")

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
					{"relation": "self", "url": r.URL.String()},
					{"relation": "next", "url": fmt.Sprintf("%s/fhir/R4/Patient?_page=2", serverURL)},
				},
			}

			json.NewEncoder(w).Encode(response)
			return
		}

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
					{"relation": "self", "url": r.URL.String()},
				},
			}

			json.NewEncoder(w).Encode(response)
			return
		}

		t.Errorf("Unexpected request count: %d", requestCount)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	serverURL = server.URL

	m := &Medplum{
		client: server.Client(),
		opts: &Options{
			MedplumURL: server.URL,
			Timezone:   "UTC",
		},
	}

	result, err := m.Search(nil, codes_go_proto.ResourceTypeCode_PATIENT, "")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if requestCount != 2 {
		t.Errorf("Expected 2 requests, got %d", requestCount)
	}

	bundle := result.ContainedResource.GetBundle()
	if bundle == nil {
		t.Fatal("Expected bundle in result")
	}

	if len(bundle.Entry) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(bundle.Entry))
	}

	if len(result.RawHTTPResponses) != 2 {
		t.Errorf("Expected 2 RawHTTPResponses, got %d", len(result.RawHTTPResponses))
	}
}

func TestSearch_NoPagination(t *testing.T) {
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
				{"relation": "self", "url": r.URL.String()},
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

func TestSearch_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/fhir+json")

		response := map[string]interface{}{
			"resourceType": "OperationOutcome",
			"issue": []map[string]interface{}{
				{
					"severity": "error",
					"code":     "invalid",
					"details":  map[string]interface{}{"text": "Invalid search parameter"},
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

	if len(result.RawHTTPResponses) == 0 {
		t.Fatal("Expected at least one RawHTTPResponse")
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}

func TestSearch_EmptyResult(t *testing.T) {
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
