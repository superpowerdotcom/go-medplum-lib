package main

import (
	"fmt"
	"log"
	"os"

	"github.com/superpowerdotcom/fhir/go/fhirversion"
	"github.com/superpowerdotcom/fhir/go/jsonformat"
	c_gp "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	dt "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	bcr_gp "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	u_gp "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/user_go_proto"
	"github.com/superpowerdotcom/go-medplum-lib"
)

func main() {
	m, err := medplum.New(&medplum.Options{
		MedplumURL:   "http://localhost:8103",
		ClientID:     "a787c2f4-0ca0-4abe-b9fa-2a36d628b67d",
		ClientSecret: "61ac1d01d0a414e0f4d051ff30227765984367edcc316e70bc2f7d6e9f3260af",
	})
	if err != nil {
		fmt.Println("unable to create medplum client: " + err.Error())
		os.Exit(1)
	}
	fmt.Println("Successfully authenticated")

	// Create a User resource according to Medplum FHIR spec
	user := &u_gp.User{
		FirstName:     &dt.String{Value: "Alice"},
		LastName:      &dt.String{Value: "Smith"},
		Email:         &dt.String{Value: "alice.smith@example.com"},
		EmailVerified: &dt.Boolean{Value: true},
	}

	// Create a ContainedResource for the User
	containedUser := &bcr_gp.ContainedResource{
		OneofResource: &bcr_gp.ContainedResource_User{
			User: user,
		},
	}

	// Create a Bundle.Entry with the User resource
	entry := &bcr_gp.Bundle_Entry{
		Resource: containedUser,
		Request: &bcr_gp.Bundle_Entry_Request{
			Method: &bcr_gp.Bundle_Entry_Request_MethodCode{
				Value: c_gp.HTTPVerbCode_POST,
			},
			Url: &dt.Uri{Value: "User"},
		},
	}

	// Create a Bundle with the required type field set to TRANSACTION
	bundle := &bcr_gp.Bundle{
		Type: &bcr_gp.Bundle_TypeCode{
			Value: c_gp.BundleTypeCode_TRANSACTION,
		},
		Entry: []*bcr_gp.Bundle_Entry{entry},
	}

	// Create the contained resource bundle for marshalling
	containedBundle := &bcr_gp.ContainedResource{
		OneofResource: &bcr_gp.ContainedResource_Bundle{
			Bundle: bundle,
		},
	}

	// Test marshalling and unmarshalling the Bundle
	fmt.Println("Testing Bundle marshalling/unmarshalling...")

	marshaller, err := jsonformat.NewMarshaller(false, "", "  ", fhirversion.R4)
	if err != nil {
		log.Fatal("Failed to create marshaller:", err)
	}

	data, err := marshaller.Marshal(containedBundle)
	if err != nil {
		log.Fatal("Unable to marshal bundle:", err)
	}

	fmt.Println("Marshalled JSON:")
	fmt.Println(string(data))

	// For unmarshalling, the JSON should represent a Bundle, not a ContainedResource
	unmarshaller, err := jsonformat.NewUnmarshaller("UTC", fhirversion.R4)
	if err != nil {
		log.Fatal("Failed to create unmarshaller:", err)
	}

	unmarshalledResource, err := unmarshaller.UnmarshalR4(data)
	if err != nil {
		log.Fatal("Unable to unmarshal JSON:", err)
	}

	fmt.Println("Successfully marshalled and unmarshalled!")

	// Access the User fields from the unmarshalled bundle
	unmarshalledBundle := unmarshalledResource.GetBundle()
	if unmarshalledBundle != nil && len(unmarshalledBundle.Entry) > 0 {
		userResource := unmarshalledBundle.Entry[0].GetResource().GetUser()
		if userResource != nil {
			fmt.Printf("Unmarshalled User: Name=%s %s, Email=%s, Verified=%v\n",
				userResource.FirstName.Value,
				userResource.LastName.Value,
				userResource.Email.Value,
				userResource.EmailVerified.Value,
			)
		}
	}

	// Attempt to create user via transaction
	result, err := m.ExecuteBatch(nil, containedBundle)
	if err != nil {
		fmt.Println("Unable to execute transaction: " + err.Error())
		os.Exit(1)
	}

	// Check transaction response
	resultBundle := result.ContainedResource.GetBundle()
	if resultBundle == nil {
		fmt.Println("ERROR: Transaction response bundle is nil")
		os.Exit(1)
	}

	fmt.Printf("Transaction processed %d entries\n", len(resultBundle.Entry))

	// Check each response entry
	for i, entry := range resultBundle.Entry {
		if entry.Response != nil {
			status := entry.Response.Status.Value
			fmt.Printf("Entry %d: HTTP Status %s\n", i, status)

			if status != "201" && status != "200" {
				// Transaction failed
				if entry.Response.Outcome != nil {
					outcome := entry.Response.Outcome.GetOperationOutcome()
					if outcome != nil && len(outcome.Issue) > 0 {
						issue := outcome.Issue[0]
						fmt.Printf("ERROR: %s - %s\n",
							issue.Code.Value,
							issue.Details.Text.Value)
					}
				}
				fmt.Printf("FAILED: User creation was rejected by server (status %s)\n", status)
				fmt.Println("This likely means the client doesn't have permission to create User resources")
				os.Exit(1)
			}
		}

		// Success case - extract created user
		if entry.GetResource() != nil {
			userRes := entry.GetResource().GetUser()
			if userRes != nil {
				fmt.Printf("✅ Created User: ID=%s, Name=%s %s, Email=%s\n",
					userRes.Id.Value,
					userRes.FirstName.Value,
					userRes.LastName.Value,
					userRes.Email.Value,
				)
			}
		}
	}

	os.Exit(0)
}
