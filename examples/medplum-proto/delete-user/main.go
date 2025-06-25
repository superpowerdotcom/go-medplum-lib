package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/superpowerdotcom/fhir/go/fhirversion"
	"github.com/superpowerdotcom/fhir/go/jsonformat"
	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	dt "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	cr "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	u_gp "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/user_go_proto"
	"github.com/superpowerdotcom/go-medplum-lib"
)

// testBundleMarshalling tests marshalling and unmarshalling of a bundle
func testBundleMarshalling(bundle *cr.ContainedResource, isRequest bool) *cr.ContainedResource {
	bundleType := "response"
	if isRequest {
		bundleType = "request"
	}

	fmt.Printf("Testing %s bundle marshalling/unmarshalling...\n", bundleType)

	marshaller, err := jsonformat.NewMarshaller(false, "  ", "  ", fhirversion.R4)
	if err != nil {
		log.Fatalf("Failed to create marshaller: %v", err)
	}

	data, err := marshaller.Marshal(bundle)
	if err != nil {
		log.Fatalf("Unable to marshal %s bundle: %v", bundleType, err)
	}

	// Pretty-print the JSON using standard library
	var jsonObj interface{}
	err = json.Unmarshal(data, &jsonObj)
	if err != nil {
		log.Fatalf("Unable to parse JSON for pretty-printing: %v", err)
	}

	prettyData, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		log.Fatalf("Unable to pretty-print JSON: %v", err)
	}

	fmt.Printf("Marshalled %s bundle JSON:\n", bundleType)
	fmt.Println(string(prettyData))

	// Write pretty-printed JSON to file
	filename := fmt.Sprintf("./%s_bundle.json", bundleType)
	err = os.WriteFile(filename, prettyData, 0644)
	if err != nil {
		log.Printf("Warning: Failed to write JSON to file %s: %v", filename, err)
	} else {
		fmt.Printf("💾 Saved %s bundle JSON to: %s\n", bundleType, filename)
	}

	unmarshaller, err := jsonformat.NewUnmarshaller("UTC", fhirversion.R4)
	if err != nil {
		log.Fatalf("Failed to create unmarshaller: %v", err)
	}

	unmarshalledResource, err := unmarshaller.UnmarshalR4(data)
	if err != nil {
		log.Fatalf("Unable to unmarshal %s bundle JSON: %v", bundleType, err)
	}

	fmt.Printf("Successfully marshalled and unmarshalled %s bundle!\n", bundleType)

	// Access the Bundle and display information
	unmarshalledBundle := unmarshalledResource.GetBundle()
	if unmarshalledBundle != nil && len(unmarshalledBundle.Entry) > 0 {
		for i, entry := range unmarshalledBundle.Entry {
			// Check for delete requests
			if entry.Request != nil {
				fmt.Printf("Unmarshalled request %d: Method=%s, URL=%s\n",
					i+1,
					entry.Request.Method.Value,
					entry.Request.Url.Value,
				)
			}
			// Check for user resources
			userResource := entry.GetResource().GetUser()
			if userResource != nil {
				fmt.Printf("Unmarshalled User %d: Name=%s %s, Email=%s, Verified=%v\n",
					i+1,
					userResource.FirstName.Value,
					userResource.LastName.Value,
					userResource.Email.Value,
					userResource.EmailVerified.Value,
				)
			}
		}
	}
	fmt.Println()

	return unmarshalledResource
}

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

	// Create a user to delete
	userID, err := createUser(m)
	if err != nil {
		fmt.Println("Unable to create user for deletion: " + err.Error())
		os.Exit(1)
	}

	fmt.Printf("Created user with ID: %s\n", userID)

	// Wait a moment to ensure the user is indexed
	time.Sleep(time.Millisecond * 100)

	// Verify the user exists before deletion
	result, err := m.ReadResource(nil, userID, codes_go_proto.ResourceTypeCode_USER)
	if err != nil {
		fmt.Println("Unable to read user before deletion: " + err.Error())
		os.Exit(1)
	}

	userResource := result.ContainedResource.GetUser()
	if userResource == nil {
		fmt.Println("Expected User resource but got something else")
		os.Exit(1)
	}

	fmt.Printf("Confirmed user exists: Name=%s %s, Email=%s\n",
		userResource.FirstName.Value,
		userResource.LastName.Value,
		userResource.Email.Value,
	)

	// Delete the user using transaction
	deleteBundle := &cr.Bundle{
		Type: &cr.Bundle_TypeCode{
			Value: codes_go_proto.BundleTypeCode_TRANSACTION,
		},
		Entry: []*cr.Bundle_Entry{
			{
				Request: &cr.Bundle_Entry_Request{
					Method: &cr.Bundle_Entry_Request_MethodCode{
						Value: codes_go_proto.HTTPVerbCode_DELETE,
					},
					Url: &dt.Uri{Value: "User/" + userID},
				},
			},
		},
	}

	// Create the contained resource bundle for marshalling
	deleteContainedBundle := &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Bundle{
			Bundle: deleteBundle,
		},
	}

	// Test marshalling and unmarshalling the delete Bundle
	testBundleMarshalling(deleteContainedBundle, true)

	deleteResult, err := m.ExecuteBatch(nil, deleteContainedBundle)
	if err != nil {
		fmt.Println("Unable to execute delete transaction: " + err.Error())
		os.Exit(1)
	}

	// Test marshalling and unmarshalling the response Bundle
	testBundleMarshalling(deleteResult.ContainedResource, false)

	// Check delete transaction response
	deleteResultBundle := deleteResult.ContainedResource.GetBundle()
	if deleteResultBundle == nil || len(deleteResultBundle.Entry) == 0 {
		fmt.Println("No entries in delete transaction response")
		os.Exit(1)
	}

	entry := deleteResultBundle.Entry[0]
	if entry.Response != nil {
		status := entry.Response.Status.Value
		fmt.Printf("Delete transaction status: %s\n", status)

		if status != "200" && status != "204" && status != "404" {
			// Delete failed (404 is OK - already deleted)
			if entry.Response.Outcome != nil {
				outcome := entry.Response.Outcome.GetOperationOutcome()
				if outcome != nil && len(outcome.Issue) > 0 {
					issue := outcome.Issue[0]
					fmt.Printf("ERROR: %s - %s\n",
						issue.Code.Value,
						issue.Details.Text.Value)
				}
			}
			fmt.Printf("FAILED: User deletion was rejected by server (status %s)\n", status)
			fmt.Println("This likely means the client doesn't have permission to delete User resources")
			os.Exit(1)
		}

		if status == "404" {
			fmt.Println("⚠️  User was already deleted or not found")
		} else {
			fmt.Printf("✅ User deletion successful (status %s)\n", status)
		}
	}

	// Wait a moment then try to read the user to confirm deletion
	time.Sleep(time.Millisecond * 100)

	verifyResult, err := m.ReadResource(nil, userID, codes_go_proto.ResourceTypeCode_USER)
	if err != nil {
		fmt.Printf("✅ Confirmed: User no longer exists (expected error: %s)\n", err.Error())
	} else {
		// User still exists
		if verifyResult.ContainedResource.GetUser() != nil {
			fmt.Println("⚠️  Warning: User still exists after deletion attempt")
		}
	}

	fmt.Println("Delete operation completed")
	os.Exit(0)
}

func createUser(m *medplum.Medplum) (string, error) {
	user := &u_gp.User{
		FirstName:     &dt.String{Value: "Delete"},
		LastName:      &dt.String{Value: "TestUser"},
		Email:         &dt.String{Value: "delete.testuser@example.com"},
		EmailVerified: &dt.Boolean{Value: true},
	}

	// Create transaction bundle
	bundle := &cr.Bundle{
		Type: &cr.Bundle_TypeCode{
			Value: codes_go_proto.BundleTypeCode_TRANSACTION,
		},
		Entry: []*cr.Bundle_Entry{
			{
				Resource: &cr.ContainedResource{
					OneofResource: &cr.ContainedResource_User{
						User: user,
					},
				},
				Request: &cr.Bundle_Entry_Request{
					Method: &cr.Bundle_Entry_Request_MethodCode{
						Value: codes_go_proto.HTTPVerbCode_POST,
					},
					Url: &dt.Uri{Value: "User"},
				},
			},
		},
	}

	result, err := m.ExecuteBatch(nil, &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Bundle{
			Bundle: bundle,
		},
	})
	if err != nil {
		return "", err
	}

	// Check transaction response
	resultBundle := result.ContainedResource.GetBundle()
	if resultBundle == nil || len(resultBundle.Entry) == 0 {
		return "", errors.New("no entries in transaction response")
	}

	entry := resultBundle.Entry[0]
	if entry.Response != nil {
		status := entry.Response.Status.Value
		if status != "201" && status != "200" {
			// Extract error details
			errorMsg := fmt.Sprintf("server returned status %s", status)
			if entry.Response.Outcome != nil {
				outcome := entry.Response.Outcome.GetOperationOutcome()
				if outcome != nil && len(outcome.Issue) > 0 {
					issue := outcome.Issue[0]
					errorMsg = fmt.Sprintf("%s - %s", issue.Code.Value, issue.Details.Text.Value)
				}
			}
			return "", fmt.Errorf("FAILED: User creation was rejected by server: %s\nThis likely means the client doesn't have permission to create User resources", errorMsg)
		}
	}

	// Extract created user
	if entry.GetResource() != nil {
		userRes := entry.GetResource().GetUser()
		if userRes != nil {
			return userRes.Id.Value, nil
		}
	}

	return "", errors.New("unexpected: returned user resource is nil")
}
