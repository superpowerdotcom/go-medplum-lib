package medplum

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/google/fhir/go/fhirversion"
	"github.com/google/fhir/go/jsonformat"
	cr "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	"github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	cc "golang.org/x/oauth2/clientcredentials"
)

type Options struct {
	// MedplumURL is the URL to the Medplum API
	MedplumURL string

	// ClientID is the ID for the ClientApplication created in Medplum
	ClientID string

	// ClientSecret is the secret for the ClientApplication created in Medplum
	ClientSecret string

	// TokenURL is the URL to the token exchange endpoint
	TokenURL string // ie. http://localhost:8103/oauth2/token

	// ClientCtx allows you to pass an optional context that can include a
	// custom http.Client. The context will
	//
	// Read more about this here: https://pkg.go.dev/golang.org/x/oauth2/clientcredentials#Config.Client
	// Read about the oauth2.HTTPClient var: https://pkg.go.dev/golang.org/x/oauth2#pkg-variables
	ClientCtx context.Context
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

func (m *Medplum) CreateResource(ctx context.Context, resource *cr.ContainedResource) (*http.Response, error) {
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

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send POST request: %s", err)
	}

	return resp, nil
}

func (m *Medplum) ReadResource(id string, rtc *codes_go_proto.ResourceTypeCode) (interface{}, error) {
	if id == "" {
		return nil, errors.New("id cannot be empty")
	}

	if err := validResourceCode(rtc); err != nil {
		return nil, err
	}

	fmt.Printf("%+v\n", rtc)

	return nil, errors.New("not implemented")
}

func validResourceCode(rtc *codes_go_proto.ResourceTypeCode) error {
	if rtc == nil {
		return errors.New("ResourceTypeCode cannot be nil")
	}

	return nil
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
