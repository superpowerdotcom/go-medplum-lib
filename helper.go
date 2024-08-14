package medplum

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	cr "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	"golang.org/x/oauth2/clientcredentials"
)

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
	cfg := &clientcredentials.Config{
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
