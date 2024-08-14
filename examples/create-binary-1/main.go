package main

import (
	"fmt"
	"os"

	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	"github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/binary_go_proto"
	cr "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"

	"github.com/superpowerdotcom/go-medplum-lib"
)

func main() {
	m, err := medplum.New(&medplum.Options{
		MedplumURL:   "http://localhost:8103",
		ClientID:     "3008218e-5de9-4398-a987-ca393e3e64b0",
		ClientSecret: "1b6b7708423fa6cc589d2996e40d35bc2ba38d6af366e16660bcfcecb5438896",
		TokenURL:     "http://localhost:8103/oauth2/token",
	})

	if err != nil {
		fmt.Println("unable to create medplum client: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("Successfully authenticated")

	// Read file contents
	data, err := os.ReadFile("bees.png")
	if err != nil {
		fmt.Println("unable to read file: " + err.Error())
		os.Exit(1)
	}

	// Create binary
	binaryResource := &binary_go_proto.Binary{
		ContentType: &binary_go_proto.Binary_ContentTypeCode{
			Value: "image/png",
		},
		Data: &datatypes_go_proto.Base64Binary{
			Value: data,
		},
	}

	// Create contained resource
	crBinary := &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Binary{
			Binary: binaryResource,
		},
	}

	// Create the resource
	result, err := m.CreateResource(crBinary)
	if err != nil {
		fmt.Println("unable to create resource: " + err.Error())
		os.Exit(1)
	}

	// Did the create succeed?
	if result.RawHTTPResponse.StatusCode < 200 || result.RawHTTPResponse.StatusCode >= 300 {
		fmt.Printf("unexpected response status code: %d\n", result.RawHTTPResponse.StatusCode)
		os.Exit(1)
	}

	// Binary might not be able to get unmarshalled into an FHIR resource, so
	// we'll check MapResource instead.
	binaryIDInterface, ok := result.MapResource["id"]
	if !ok {
		fmt.Println("Unexpected 'id' not contained in MapResource")
		os.Exit(1)
	}

	binaryID, ok := binaryIDInterface.(string)
	if !ok {
		fmt.Println("Unable to type assert id to a string")
		os.Exit(1)
	}

	binaryURLInterface, ok := result.MapResource["url"]
	if !ok {
		fmt.Println("Unexpected 'url' not contained in MapResource")
		os.Exit(1)
	}

	binaryURL, ok := binaryURLInterface.(string)
	if !ok {
		fmt.Println("Unable to type assert URL to a string")
		os.Exit(1)
	}

	fmt.Println("[created] Binary ID: " + binaryID)
	fmt.Println("[created] Binary URL: " + binaryURL)
}
