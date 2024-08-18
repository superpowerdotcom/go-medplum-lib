package medplum

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/google/fhir/go/fhirversion"
	"github.com/google/fhir/go/jsonformat"
	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/binary_go_proto"
	cr "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type IMedplum interface {
	CreateResource(ctx context.Context, resource *cr.ContainedResource) (*Result, error)
	CreateBinaryResource(ctx context.Context, data []byte, contentType string) (*Result, error)
	UpdateResource(ctx context.Context, id string, resource *cr.ContainedResource) (*Result, error)
	DeleteResource(ctx context.Context, id string, code codes_go_proto.ResourceTypeCode_Value) (*Result, error)
	ReadResource(ctx context.Context, id string, code codes_go_proto.ResourceTypeCode_Value) (*Result, error)
	Search(ctx context.Context, code codes_go_proto.ResourceTypeCode_Value, query string) (*Result, error)
}

type Options struct {
	// Required: MedplumURL is the URL to the Medplum API
	//
	// Example: http://localhost:8103
	MedplumURL string

	// Required: ClientID is the ID for a ClientApplication created in Medplum
	ClientID string

	// Required: ClientSecret is the secret for a ClientApplication created in Medplum
	ClientSecret string

	// Required: TokenURL is the URL to the token exchange endpoint
	//
	// Example: http://localhost:8103/oauth2/token
	TokenURL string

	// Optional: ClientCtx allows you to pass a context that can include a
	// custom http.Client.
	//
	// Read more about this here: https://pkg.go.dev/golang.org/x/oauth2/clientcredentials#Config.Client
	// Read about the oauth2.HTTPClient var: https://pkg.go.dev/golang.org/x/oauth2#pkg-variables
	//
	// Default: context.Background()
	ClientCtx context.Context

	// Optional: Timezone used when marshalling and unmarshalling responses from
	// Medplum API.
	//
	// The name corresponds to a file in the IANA Time Zone database, such as
	// "America/New_York".
	//
	// Default: "UTC".
	Timezone string

	// Optional: Whether to log errors (such as during unmarshal attempts).
	//
	// Default: false
	LogErrors bool
}

// Result is a common "wrapper" struct that is returned from some of the public
// methods in the Medplum library.
type Result struct {
	// ContainedResource is an FHIR "container" resource - it contains exactly
	// one other resource inside of it. When Medplum responds, the response will
	// be a ContainedResource and it is the responsibility of the caller to
	// "extract" the contained resource within it.
	//
	// NOTE: It is possible for the jsonformat library to fail to unmarshal the
	// Medplum response into a ContainedResource. To be able to handle this,
	// make sure to check that ContainedResource is not nil. If it's nil, you
	// can try using MapResource instead or look through the RawHTTPResponse.
	//
	// You can see an example of what's involved in extracting a contained
	// resource in some of the examples in `./examples` dir.
	ContainedResource *cr.ContainedResource

	// MapResource is used as a "basic"/"last-resort" storage structure for
	// storing responses from Medplum. If ContainedResource is nil, you can try
	// using MapResource instead.
	MapResource map[string]interface{}

	// All responses from Medplum will also include the "raw" HTTP response that
	// the library receives from the Medplum API.
	//
	// This is useful if you need to inspect headers, status codes, read the
	// body manually etc. It is especially useful if you do not need to extract
	// the wrapped resource from the ContainedResource. For example, if you are
	// deleting a resource, you might only want to check the status code.
	RawHTTPResponse *http.Response
}

type Medplum struct {
	client *http.Client
	opts   *Options
}

var (
	ErrResourceCannotBeNil = errors.New("resource cannot be nil")
	ErrInvalidResource     = errors.New("invalid resource")
)

func New(opts *Options) (*Medplum, error) {
	if err := validateOptions(opts); err != nil {
		return nil, fmt.Errorf("failed to validate options: %s", err)
	}

	client, err := auth(opts.ClientID, opts.ClientSecret, opts.TokenURL, opts.ClientCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to auth: %s", err)
	}

	return &Medplum{
		client: client,
		opts:   opts,
	}, nil
}

// CreateBinaryResource is a convenience method for creating a Binary resource
func (m *Medplum) CreateBinaryResource(ctx context.Context, data []byte, contentType string) (*Result, error) {
	if ctx != nil {
		segment := newrelic.FromContext(ctx).StartSegment("go-medplum-lib.CreateBinaryResource")
		defer segment.End()
	}

	if data == nil {
		return nil, errors.New("data cannot be nil")
	}

	if contentType == "" {
		return nil, errors.New("contentType cannot be empty")
	}

	resource := &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Binary{
			Binary: &binary_go_proto.Binary{
				ContentType: &binary_go_proto.Binary_ContentTypeCode{Value: contentType},
				Data:        &datatypes_go_proto.Base64Binary{Value: data},
			},
		},
	}

	return m.CreateResource(ctx, resource)
}

func (m *Medplum) CreateResource(ctx context.Context, resource *cr.ContainedResource) (*Result, error) {
	if ctx != nil {
		segment := newrelic.FromContext(ctx).StartSegment("go-medplum-lib.CreateResource")
		defer segment.End()
	}

	if err := validResource(resource); err != nil {
		return nil, err
	}

	resourceName, err := getContainedResourceName(resource)
	if err != nil {
		return nil, fmt.Errorf("unable to get contained resource name: %s", err)
	}

	// Marshal resource to JSON
	marshaller, err := jsonformat.NewPrettyMarshaller(fhirversion.R4)
	if err != nil {
		return nil, fmt.Errorf("unable to create proto -> json marshaler: %s", err)
	}

	data, err := marshaller.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal resource to JSON: %s", err)
	}

	// Send to Medplum API
	req, err := http.NewRequest("POST", m.url(resourceName), bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %s", err)
	}

	// It is incredibly important to set the content-type header correctly,
	// otherwise Medplum API will return 400 errors.
	req.Header.Set("Content-Type", "application/fhir+json")

	httpResp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send request: %s", err)
	}

	result, err := m.generateResult(httpResp)
	if err != nil {
		return nil, fmt.Errorf("unable to generate response: %s", err)
	}

	return result, nil
}

func (m *Medplum) ReadResource(ctx context.Context, id string, code codes_go_proto.ResourceTypeCode_Value) (*Result, error) {
	if ctx != nil {
		segment := newrelic.FromContext(ctx).StartSegment("go-medplum-lib.ReadResource")
		defer segment.End()
	}

	if id == "" {
		return nil, errors.New("id cannot be empty")
	}

	resourceName, err := getResourceNameFromTypeCode(code)
	if err != nil {
		return nil, fmt.Errorf("unable to get resource name from type code: %s", err)
	}

	req, err := http.NewRequest("GET", m.url(resourceName)+"/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %s", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")

	httpResp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send request: %s", err)
	}

	return m.generateResult(httpResp)
}

func (m *Medplum) UpdateResource(ctx context.Context, id string, resource *cr.ContainedResource) (*Result, error) {
	if ctx != nil {
		segment := newrelic.FromContext(ctx).StartSegment("go-medplum-lib.UpdateResource")
		defer segment.End()
	}

	if err := validResource(resource); err != nil {
		return nil, err
	}

	resourceName, err := getContainedResourceName(resource)
	if err != nil {
		return nil, fmt.Errorf("unable to get contained resource name: %s", err)
	}

	// Marshal resource to JSON
	marshaller, err := jsonformat.NewPrettyMarshaller(fhirversion.R4)
	if err != nil {
		return nil, fmt.Errorf("unable to create proto json marshaler: %s", err)
	}

	data, err := marshaller.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal resource to JSON: %s", err)
	}

	// Send to Medplum API
	req, err := http.NewRequest("PUT", m.url(resourceName)+"/"+id, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %s", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")

	httpResp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send request: %s", err)
	}

	return m.generateResult(httpResp)
}

func (m *Medplum) DeleteResource(ctx context.Context, id string, code codes_go_proto.ResourceTypeCode_Value) (*Result, error) {
	if ctx != nil {
		segment := newrelic.FromContext(ctx).StartSegment("go-medplum-lib.DeleteResource")
		defer segment.End()
	}

	resourceName, err := getResourceNameFromTypeCode(code)
	if err != nil {
		return nil, fmt.Errorf("unable to get resource name from type code: %s", err)
	}

	req, err := http.NewRequest("DELETE", m.url(resourceName)+"/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %s", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")

	httpResp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send request: %s", err)
	}

	return m.generateResult(httpResp)
}

// Search will use the provided query to search resources and return a Bundle
// resource type. If query is empty, Medplum will return all resources of the
// provided type; if query is not empty, it must be a valid FHIR search query.
//
// Refer: https://hl7.org/fhir/search.html
func (m *Medplum) Search(ctx context.Context, code codes_go_proto.ResourceTypeCode_Value, query string) (*Result, error) {
	if ctx != nil {
		segment := newrelic.FromContext(ctx).StartSegment("go-medplum-lib.Search")
		defer segment.End()
	}

	resourceName, err := getResourceNameFromTypeCode(code)
	if err != nil {
		return nil, fmt.Errorf("unable to get resource name from type code: %s", err)
	}

	req, err := http.NewRequest("GET", m.url(resourceName)+"?"+query, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %s", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")

	httpResp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send request: %s", err)
	}

	return m.generateResult(httpResp)
}

func (m *Medplum) url(resourceName string) string {
	return fmt.Sprintf("%s/fhir/R4/%s", m.opts.MedplumURL, resourceName)
}

func (m *Medplum) generateResult(httpResp *http.Response) (*Result, error) {
	if httpResp == nil {
		return nil, errors.New("http response cannot be nil")
	}

	// Read the body so we can create a ContainedResource
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %s", err)
	}

	defer httpResp.Body.Close()

	// Make sure to reset the body so that the caller can read it
	httpResp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Unmarshal the response body into a ContainedResource
	unmarshaller, err := jsonformat.NewUnmarshallerWithoutValidation(m.opts.Timezone, fhirversion.R4)
	if err != nil {
		return nil, fmt.Errorf("unable to create unmarshaler: %s", err)
	}

	containedResource, err := unmarshaller.UnmarshalR4(bodyBytes)
	if err != nil {
		if m.opts.LogErrors {
			log.Println("go-medplum-lib: unable to unmarshal response body using FHIR protos: " + err.Error())
		}
	}

	// If we failed to unmarshal response, create an empty ContainedResource to
	// prevent panics in caller code.
	if containedResource == nil {
		containedResource = &cr.ContainedResource{}
	}

	mapResource := make(map[string]interface{})

	if err := json.Unmarshal(bodyBytes, &mapResource); err != nil {
		if m.opts.LogErrors {
			fmt.Println("go-medplum-lib: unable to unmarshal response body using map: " + err.Error())
		}
	}

	return &Result{
		ContainedResource: containedResource,
		MapResource:       mapResource,
		RawHTTPResponse:   httpResp,
	}, nil
}
