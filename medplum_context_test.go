package medplum

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	codes "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
)

type contextKey string

type contextObservingTransport struct {
	started  chan struct{}
	canceled chan struct{}
	once     sync.Once
	want     any
}

func (transport *contextObservingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if got := req.Context().Value(contextKey("request")); got != transport.want {
		return nil, errors.New("request context value was not preserved")
	}

	transport.once.Do(func() { close(transport.started) })
	<-req.Context().Done()
	close(transport.canceled)
	return nil, req.Context().Err()
}

func TestRequestContextReachesTransport(t *testing.T) {
	transport := &contextObservingTransport{
		started:  make(chan struct{}),
		canceled: make(chan struct{}),
		want:     "trace-context",
	}
	m := &Medplum{
		client: &http.Client{Transport: transport},
		opts:   &Options{MedplumURL: "http://medplum.test", Timezone: "UTC"},
	}

	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey("request"), transport.want))
	done := make(chan error, 1)
	go func() {
		_, err := m.ReadResource(ctx, "123", codes.ResourceTypeCode_PATIENT)
		done <- err
	}()

	select {
	case <-transport.started:
	case <-time.After(time.Second):
		t.Fatal("HTTP transport did not receive the request context")
	}

	cancel()

	select {
	case <-transport.canceled:
	case <-time.After(time.Second):
		t.Fatal("HTTP transport did not observe context cancellation")
	}

	if err := <-done; err == nil || (!errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), context.Canceled.Error())) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}
}

func TestNewRequestAcceptsNilContext(t *testing.T) {
	req, err := newRequest(nil, http.MethodGet, "http://medplum.test/fhir/R4/Patient/123", nil)
	if err != nil {
		t.Fatalf("newRequest returned an error: %v", err)
	}
	if req.Context() == nil {
		t.Fatal("expected a non-nil request context")
	}
	if err := req.Context().Err(); err != nil {
		t.Fatalf("expected a live background context, got %v", err)
	}
}

func TestInjectedNewRelicTransportPropagatesTraceparent(t *testing.T) {
	var traceparent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/oauth2/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "test-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case "/fhir/R4/Patient/123":
			traceparent = req.Header.Get("traceparent")
			w.Header().Set("Content-Type", "application/fhir+json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"resourceType": "Patient",
				"id":           "123",
			})
		default:
			http.NotFound(w, req)
		}
	}))
	defer server.Close()

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("go-medplum-lib-test"),
		newrelic.ConfigAppLogEnabled(false),
		func(config *newrelic.Config) {
			config.DistributedTracer.Enabled = true
			config.ServerlessMode.Enabled = true
			config.ServerlessMode.AccountID = "1"
			config.ServerlessMode.TrustedAccountKey = "1"
			config.ServerlessMode.PrimaryAppID = "1"
		},
	)
	if err != nil {
		t.Fatalf("failed to create New Relic test application: %v", err)
	}
	defer app.Shutdown(time.Second)

	baseClient := server.Client()
	instrumentedClient := &http.Client{
		Transport:     newrelic.NewRoundTripper(baseClient.Transport),
		CheckRedirect: baseClient.CheckRedirect,
		Jar:           baseClient.Jar,
		Timeout:       baseClient.Timeout,
	}
	m, err := New(&Options{
		MedplumURL:   server.URL,
		TokenURL:     server.URL + "/oauth2/token",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		HTTPClient:   instrumentedClient,
		Timezone:     "UTC",
	})
	if err != nil {
		t.Fatalf("failed to create Medplum client: %v", err)
	}

	txn := app.StartTransaction("test-medplum-read")
	ctx := newrelic.NewContext(context.Background(), txn)
	_, requestErr := m.ReadResource(ctx, "123", codes.ResourceTypeCode_PATIENT)
	txn.End()
	if requestErr != nil {
		t.Fatalf("ReadResource failed: %v", requestErr)
	}
	if traceparent == "" {
		t.Fatal("expected New Relic transport to propagate traceparent")
	}
}
