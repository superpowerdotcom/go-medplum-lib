package medplum

import (
	"errors"
	"fmt"
	"net/http"
)

type Options struct {
	ClientID      string
	ClientSecret  string
	TokenEndpoint string
	AuthEndpoint  string
}

type Medplum struct {
	client *http.Client
	opts   *Options
}

func New(opts *Options) (*Medplum, error) {
	if err := validateOptions(opts); err != nil {
		return nil, fmt.Errorf("failed to validate options: %s", err)
	}

	client, err := login(opts.ClientID, opts.ClientSecret, opts.TokenEndpoint, opts.AuthEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %s", err)
	}

	return &Medplum{
		client: client,
		opts:   opts,
	}, nil
}

func (m *Medplum) CreateResource(resourceType string, resource interface{}) error {
	return errors.New("not implemented")
}

func (m *Medplum) GetResource(resourceType, id string) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (m *Medplum) UpdateResource(resourceType, id string, resource interface{}) error {
	return errors.New("not implemented")
}

func (m *Medplum) DeleteResource(resourceType, id string) error {
	return errors.New("not implemented")
}

func validateOptions(opts *Options) error {
	if opts.ClientID == "" {
		return errors.New("ClientID is required")
	}

	if opts.ClientSecret == "" {
		return errors.New("ClientSecret is required")
	}

	if opts.TokenEndpoint == "" {
		return errors.New("TokenEndpoint is required")
	}

	if opts.AuthEndpoint == "" {
		return errors.New("AuthEndpoint is required")
	}

	return nil
}

func login(clientID, clientSecret, tokenEndpoint, authEndpoint string) (*http.Client, error) {
	return nil, errors.New("not implemented")
}
