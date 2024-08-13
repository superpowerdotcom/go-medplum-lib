package medplum

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/google/fhir/go/fhirversion"
	"github.com/google/fhir/go/jsonformat"
	cr "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	"github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	cc "golang.org/x/oauth2/clientcredentials"
)

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
	// custom http.Client. Default: context.Background()
	//
	// Read more about this here: https://pkg.go.dev/golang.org/x/oauth2/clientcredentials#Config.Client
	// Read about the oauth2.HTTPClient var: https://pkg.go.dev/golang.org/x/oauth2#pkg-variables
	ClientCtx context.Context

	// Optional: Timezone used when marshalling and unmarshalling responses from
	// Medplum API.
	//
	// The name corresponds to a file in the IANA Time Zone database, such as
	// "America/New_York". Default: "UTC".
	Timezone string
}

// Result is a common "wrapper" struct that is returned from some of the public
// methods in the Medplum library.
type Result struct {
	// ContainedResource is an FHIR "container" resource - it contains exactly
	// one other resource inside of it. When Medplum responds, the response will
	// be a ContainedResource and it is the responsibility of the caller to
	// "extract" the contained resource within it.
	//
	// You can see an example of what's involved in extracting a contained
	// resource in some of the examples in `./examples` dir.
	ContainedResource *cr.ContainedResource

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

func (m *Medplum) CreateResource(ctx context.Context, resource *cr.ContainedResource) (*Result, error) {
	if err := validResource(resource); err != nil {
		return nil, err
	}

	resourceName, err := getContainedResourceName(resource)
	if err != nil {
		return nil, fmt.Errorf("unable to get contained resource name: %s", err)
	}

	// Marshal contained resource oneof to JSON
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
		return nil, fmt.Errorf("unable to create POST request: %s", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")

	httpResp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send POST request: %s", err)
	}

	result, err := m.generateResult(httpResp)
	if err != nil {
		return nil, fmt.Errorf("unable to generate response: %s", err)
	}

	return result, nil
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
	unmarshaller, err := jsonformat.NewUnmarshaller(m.opts.Timezone, fhirversion.R4)
	if err != nil {
		return nil, fmt.Errorf("unable to create unmarshaler: %s", err)
	}

	containedResource, err := unmarshaller.UnmarshalR4(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal response body: %s", err)
	}

	return &Result{
		ContainedResource: containedResource,
		RawHTTPResponse:   httpResp,
	}, nil
}

func (m *Medplum) ReadResource(id string, code codes_go_proto.ResourceTypeCode_Value) (*Result, error) {
	if id == "" {
		return nil, errors.New("id cannot be empty")
	}

	resourceName, err := getResourceNameFromTypeCode(code)
	if err != nil {
		return nil, fmt.Errorf("unable to get resource name from type code: %s", err)
	}

	req, err := http.NewRequest("GET", m.url(resourceName)+"/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create POST request: %s", err)
	}

	req.Header.Set("Content-Type", "application/fhir+json")

	httpResp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send POST request: %s", err)
	}

	result, err := m.generateResult(httpResp)
	if err != nil {
		return nil, fmt.Errorf("unable to generate response: %s", err)
	}

	return result, nil
}

func (m *Medplum) UpdateResource(id string, resource *cr.ContainedResource) error {
	return errors.New("not implemented")
}

func (m *Medplum) DeleteResource(id string, rtc *codes_go_proto.ResourceTypeCode) error {
	return errors.New("not implemented")
}

func (m *Medplum) Search(rtc *codes_go_proto.ResourceTypeCode, query string) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (m *Medplum) url(resourceName string) string {
	return fmt.Sprintf("%s/fhir/R4/%s", m.opts.MedplumURL, resourceName)
}

func getResourceNameFromTypeCode(code codes_go_proto.ResourceTypeCode_Value) (string, error) {
	name, exists := codes_go_proto.ResourceTypeCode_Value_name[int32(code)]
	if !exists {
		return "", fmt.Errorf("resource name not found for code: %d", code)
	}

	return normalizeResourceName(name), nil
}

func normalizeResourceName(s string) string {
	if len(s) == 0 {
		return s
	}

	s = strings.ToLower(s)

	return strings.ToUpper(string(s[0])) + s[1:]
}

func getContainedResourceName(resource *cr.ContainedResource) (string, error) {
	if err := validResource(resource); err != nil {
		return "", err
	}

	res := resource.GetOneofResource()
	resourceType := reflect.TypeOf(res).Elem().Name()

	if resourceType == "" {
		return "", errors.New("resource name lookup resulted in an empty name")
	}

	result := strings.Split(resourceType, "_")

	if len(result) != 2 {
		return "", fmt.Errorf("resource name lookup resulted in unexpected name format (expected 2, got %d)", len(result))
	}

	return result[1], nil
}

func validateOptions(opts *Options) error {
	if opts.MedplumURL == "" {
		return errors.New("MedplumURL is required")
	}

	if opts.ClientID == "" {
		return errors.New("ClientID is required")
	}

	if opts.ClientSecret == "" {
		return errors.New("ClientSecret is required")
	}

	if opts.TokenURL == "" {
		return errors.New("TokenEndpoint is required")
	}

	if opts.ClientCtx == nil {
		opts.ClientCtx = context.Background()
	}

	if _, err := time.LoadLocation(opts.Timezone); err != nil {
		return fmt.Errorf("invalid timezone: %s", err)
	}

	return nil
}

func validResource(resource *cr.ContainedResource) error {
	if resource == nil {
		return ErrResourceCannotBeNil
	}

	// Check that the contained resource has a non-nil oneof
	if resource.OneofResource == nil {
		return ErrInvalidResource
	}

	return nil
}

func auth(clientID, clientSecret, tokenURL string, clientCtx context.Context) (*http.Client, error) {
	cfg := &cc.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
	}

	token, err := cfg.Token(clientCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %s", err)
	}

	if !token.Valid() {
		return nil, errors.New("token is invalid")
	}

	return cfg.Client(clientCtx), nil
}
