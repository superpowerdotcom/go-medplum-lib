package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

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

	// Access the Bundle and display user information
	unmarshalledBundle := unmarshalledResource.GetBundle()
	if unmarshalledBundle != nil && len(unmarshalledBundle.Entry) > 0 {
		for i, entry := range unmarshalledBundle.Entry {
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

	// Create a user to update
	userID, err := createUser(m)
	if err != nil {
		fmt.Println("Unable to create user for updating: " + err.Error())
		os.Exit(1)
	}

	fmt.Printf("Created user with ID: %s\n", userID)

	// Read the user to get the current version
	result, err := m.ReadResource(nil, userID, codes_go_proto.ResourceTypeCode_USER)
	if err != nil {
		fmt.Println("Unable to read user: " + err.Error())
		os.Exit(1)
	}

	userResource := result.ContainedResource.GetUser()
	if userResource == nil {
		fmt.Println("Expected User resource but got something else")
		os.Exit(1)
	}

	fmt.Printf("Current user: Name=%s %s, Email=%s\n",
		userResource.FirstName.Value,
		userResource.LastName.Value,
		userResource.Email.Value,
	)

	// Update the user - change the last name
	userResource.LastName = &dt.String{Value: "UpdatedLastName"}
	userResource.EmailVerified = &dt.Boolean{Value: false} // Change email verification status

	// Create update bundle
	updateBundle := &cr.Bundle{
		Type: &cr.Bundle_TypeCode{
			Value: codes_go_proto.BundleTypeCode_TRANSACTION,
		},
		Entry: []*cr.Bundle_Entry{
			{
				Resource: &cr.ContainedResource{
					OneofResource: &cr.ContainedResource_User{
						User: userResource,
					},
				},
				Request: &cr.Bundle_Entry_Request{
					Method: &cr.Bundle_Entry_Request_MethodCode{
						Value: codes_go_proto.HTTPVerbCode_PUT,
					},
					Url: &dt.Uri{Value: "User/" + userID},
				},
			},
		},
	}

	// Create the contained resource bundle for marshalling
	updateContainedBundle := &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Bundle{
			Bundle: updateBundle,
		},
	}

	// Test marshalling and unmarshalling the update Bundle
	testBundleMarshalling(updateContainedBundle, true)

	updateResult, err := m.ExecuteBatch(nil, updateContainedBundle)
	if err != nil {
		fmt.Println("Unable to execute update transaction: " + err.Error())
		os.Exit(1)
	}

	// Test marshalling and unmarshalling the response Bundle
	testBundleMarshalling(updateResult.ContainedResource, false)

	// Check update transaction response
	updateResultBundle := updateResult.ContainedResource.GetBundle()
	if updateResultBundle == nil || len(updateResultBundle.Entry) == 0 {
		fmt.Println("No entries in update transaction response")
		os.Exit(1)
	}

	entry := updateResultBundle.Entry[0]
	if entry.Response != nil {
		status := entry.Response.Status.Value
		fmt.Printf("Update transaction status: %s\n", status)

		if status != "200" && status != "201" {
			// Update failed
			if entry.Response.Outcome != nil {
				outcome := entry.Response.Outcome.GetOperationOutcome()
				if outcome != nil && len(outcome.Issue) > 0 {
					issue := outcome.Issue[0]
					fmt.Printf("ERROR: %s - %s\n",
						issue.Code.Value,
						issue.Details.Text.Value)
				}
			}
			fmt.Printf("FAILED: User update was rejected by server (status %s)\n", status)
			fmt.Println("This likely means the client doesn't have permission to update User resources")
			os.Exit(1)
		}
	}

	// Success case - extract updated user
	if entry.GetResource() != nil {
		updatedUser := entry.GetResource().GetUser()
		if updatedUser != nil {
			fmt.Printf("✅ Updated User: ID=%s, Name=%s %s, Email=%s, Verified=%v\n",
				updatedUser.Id.Value,
				updatedUser.FirstName.Value,
				updatedUser.LastName.Value,
				updatedUser.Email.Value,
				updatedUser.EmailVerified.Value,
			)
		}
	} else {
		fmt.Println("⚠️  Update completed but no updated resource returned")
	}

	fmt.Println("Update operation completed")
	os.Exit(0)
}

func createUser(m *medplum.Medplum) (string, error) {
	user := &u_gp.User{
		FirstName:     &dt.String{Value: "Update"},
		LastName:      &dt.String{Value: "TestUser"},
		Email:         &dt.String{Value: "update.testuser@example.com"},
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
