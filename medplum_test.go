package medplum

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"

	cr "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
)

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

	if len(result.RawHTTPResponses) != 1 {
		t.Errorf("Expected 1 RawHTTPResponse, got %d", len(result.RawHTTPResponses))
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}

func TestExecuteBatch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.URL.Path != "/fhir/R4" {
			t.Errorf("Expected path /fhir/R4, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/fhir+json")

		response := map[string]interface{}{
			"resourceType": "Bundle",
			"type":         "transaction-response",
			"entry": []map[string]interface{}{
				{
					"response": map[string]interface{}{
						"status":   "201 Created",
						"location": "Patient/123/_history/1",
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

	bundle := &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Bundle{
			Bundle: &cr.Bundle{
				Type: &cr.Bundle_TypeCode{
					Value: codes_go_proto.BundleTypeCode_TRANSACTION,
				},
				Entry: []*cr.Bundle_Entry{
					{
						Request: &cr.Bundle_Entry_Request{
							Method: &cr.Bundle_Entry_Request_MethodCode{
								Value: codes_go_proto.HTTPVerbCode_POST,
							},
							Url: &datatypes_go_proto.Uri{Value: "Patient"},
						},
						Resource: &cr.ContainedResource{
							OneofResource: &cr.ContainedResource_Patient{
								Patient: &patient_go_proto.Patient{
									Name: []*datatypes_go_proto.HumanName{
										{
											Given:  []*datatypes_go_proto.String{{Value: "Test"}},
											Family: &datatypes_go_proto.String{Value: "User"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := m.ExecuteBatch(nil, bundle)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.RawHTTPResponses[0].StatusCode)
	}

	responseBundle := result.ContainedResource.GetBundle()
	if responseBundle == nil {
		t.Fatal("Expected Bundle in result")
	}
}

func TestPost_Success(t *testing.T) {
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
			"id":           "new-123",
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
						Given:  []*datatypes_go_proto.String{{Value: "Test"}},
						Family: &datatypes_go_proto.String{Value: "User"},
					},
				},
			},
		},
	}

	result, err := m.Post(nil, resource, "Patient")
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}

func TestPost_NoEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fhir/R4" {
			t.Errorf("Expected path /fhir/R4, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/fhir+json")

		response := map[string]interface{}{
			"resourceType": "Bundle",
			"type":         "transaction-response",
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

	bundle := &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Bundle{
			Bundle: &cr.Bundle{
				Type: &cr.Bundle_TypeCode{
					Value: codes_go_proto.BundleTypeCode_TRANSACTION,
				},
			},
		},
	}

	result, err := m.Post(nil, bundle)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if result.RawHTTPResponses[0].StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.RawHTTPResponses[0].StatusCode)
	}
}

func TestPost_NilResource(t *testing.T) {
	m := &Medplum{
		opts: &Options{
			MedplumURL: "http://localhost",
			Timezone:   "UTC",
		},
	}

	_, err := m.Post(nil, nil, "Patient")
	if err == nil {
		t.Fatal("Expected error for nil resource")
	}
}

func TestValidateOptions(t *testing.T) {
	tests := []struct {
		name        string
		opts        *Options
		wantErr     bool
		errContains string
		checkFunc   func(*testing.T, *Options)
	}{
		{
			name: "missing MedplumURL",
			opts: &Options{
				ClientID:     "test-id",
				ClientSecret: "test-secret",
			},
			wantErr:     true,
			errContains: "MedplumURL",
		},
		{
			name: "missing ClientID",
			opts: &Options{
				MedplumURL:   "http://localhost",
				ClientSecret: "test-secret",
			},
			wantErr:     true,
			errContains: "ClientID",
		},
		{
			name: "missing ClientSecret",
			opts: &Options{
				MedplumURL: "http://localhost",
				ClientID:   "test-id",
			},
			wantErr:     true,
			errContains: "ClientSecret",
		},
		{
			name: "invalid timezone",
			opts: &Options{
				MedplumURL:   "http://localhost",
				ClientID:     "test-id",
				ClientSecret: "test-secret",
				Timezone:     "Invalid/Timezone",
			},
			wantErr:     true,
			errContains: "timezone",
		},
		{
			name: "empty timezone is valid",
			opts: &Options{
				MedplumURL:   "http://localhost",
				ClientID:     "test-id",
				ClientSecret: "test-secret",
				Timezone:     "",
			},
			wantErr: false,
		},
		{
			name: "sets default TokenURL",
			opts: &Options{
				MedplumURL:   "http://localhost:8103",
				ClientID:     "test-id",
				ClientSecret: "test-secret",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, opts *Options) {
				expected := "http://localhost:8103/oauth2/token"
				if opts.TokenURL != expected {
					t.Errorf("Expected TokenURL '%s', got '%s'", expected, opts.TokenURL)
				}
			},
		},
		{
			name: "valid options",
			opts: &Options{
				MedplumURL:   "http://localhost",
				ClientID:     "test-id",
				ClientSecret: "test-secret",
				Timezone:     "America/New_York",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptions(tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}

				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errContains, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.opts)
			}
		})
	}
}

func TestGenerateResult(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		m := &Medplum{opts: &Options{Timezone: "UTC"}}

		_, err := m.generateResult(nil)
		if err == nil {
			t.Fatal("Expected error for nil response")
		}
	})

	t.Run("valid response", func(t *testing.T) {
		m := &Medplum{opts: &Options{Timezone: "UTC"}}

		body := `{"resourceType":"Patient","id":"123"}`
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}
		resp.Header.Set("Content-Type", "application/fhir+json")

		result, err := m.generateResult(resp)
		if err != nil {
			t.Fatalf("generateResult failed: %v", err)
		}

		if result.ContainedResource == nil {
			t.Fatal("Expected ContainedResource in result")
		}

		if len(result.RawHTTPResponses) != 1 {
			t.Errorf("Expected 1 RawHTTPResponse, got %d", len(result.RawHTTPResponses))
		}

		if result.MapResource["resourceType"] != "Patient" {
			t.Errorf("Expected resourceType 'Patient', got '%v'", result.MapResource["resourceType"])
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		m := &Medplum{opts: &Options{Timezone: "UTC", LogErrors: false}}

		body := `not valid json`
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}

		result, err := m.generateResult(resp)
		if err != nil {
			t.Fatalf("generateResult should not fail for invalid JSON: %v", err)
		}

		if result.ContainedResource == nil {
			t.Fatal("Expected non-nil ContainedResource")
		}
	})
}

func TestPrettyPrintResult_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrettyPrintResult panicked: %v", r)
		}
	}()

	PrettyPrintResult(nil)
	PrettyPrintResult(&Result{})
	PrettyPrintResult(&Result{
		ContainedResource: &cr.ContainedResource{},
		RawHTTPResponses:  []*http.Response{},
	})
}

func TestOnResponse_CalledOnSuccess(t *testing.T) {
	var callbackCalled bool
	var callbackResp *http.Response
	var callbackBody []byte
	var callbackErr error

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
			OnResponse: func(resp *http.Response, body []byte, err error) {
				callbackCalled = true
				callbackResp = resp
				callbackBody = body
				callbackErr = err
			},
		},
	}

	_, err := m.ReadResource(nil, "123", codes_go_proto.ResourceTypeCode_PATIENT)
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if !callbackCalled {
		t.Error("Expected OnResponse callback to be called")
	}

	if callbackResp == nil {
		t.Error("Expected callback to receive non-nil response")
	}

	if callbackBody == nil {
		t.Error("Expected callback to receive non-nil body")
	}

	if !strings.Contains(string(callbackBody), "Patient") {
		t.Errorf("Expected body to contain 'Patient', got: %s", string(callbackBody))
	}

	if callbackErr != nil {
		t.Errorf("Expected callback error to be nil, got: %v", callbackErr)
	}
}

func TestOnResponse_CalledOnUnmarshalError(t *testing.T) {
	var callbackCalled bool
	var callbackBody []byte
	var callbackErr error

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/fhir+json")
		w.Write([]byte(`{"resourceType": "InvalidResource", "customField": "value"}`))
	}))
	defer server.Close()

	m := &Medplum{
		client: server.Client(),
		opts: &Options{
			MedplumURL: server.URL,
			Timezone:   "UTC",
			OnResponse: func(resp *http.Response, body []byte, err error) {
				callbackCalled = true
				callbackBody = body
				callbackErr = err
			},
		},
	}

	_, err := m.ReadResource(nil, "123", codes_go_proto.ResourceTypeCode_PATIENT)
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if !callbackCalled {
		t.Error("Expected OnResponse callback to be called")
	}

	if callbackBody == nil {
		t.Error("Expected callback to receive body bytes")
	}

	if callbackErr == nil {
		t.Error("Expected callback to receive unmarshal error")
	}
}

func TestOnResponse_CalledOnNilResponse(t *testing.T) {
	var callbackCalled bool
	var callbackResp *http.Response
	var callbackErr error

	m := &Medplum{
		opts: &Options{
			Timezone: "UTC",
			OnResponse: func(resp *http.Response, body []byte, err error) {
				callbackCalled = true
				callbackResp = resp
				callbackErr = err
			},
		},
	}

	_, err := m.generateResult(nil)
	if err == nil {
		t.Fatal("Expected error for nil response")
	}

	if !callbackCalled {
		t.Error("Expected OnResponse callback to be called even on nil response")
	}

	if callbackResp != nil {
		t.Error("Expected callback to receive nil response")
	}

	if callbackErr == nil {
		t.Error("Expected callback to receive error")
	}
}

func TestOnResponse_NotCalledWhenNil(t *testing.T) {
	m := &Medplum{
		opts: &Options{
			Timezone:   "UTC",
			OnResponse: nil,
		},
	}

	body := `{"resourceType":"Patient","id":"123"}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "application/fhir+json")

	_, err := m.generateResult(resp)
	if err != nil {
		t.Fatalf("generateResult failed: %v", err)
	}
}
