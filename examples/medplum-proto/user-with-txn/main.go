package main

import (
	"encoding/json"
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

// Transactions allow you to bundle multiple operations into a single, atomic
// request. If one of the operations fails, the entire transaction will fail and
// will be rolled back.
//
// Read more about Medplum transactions here:
// https://www.medplum.com/docs/migration/migration-pipelines#using-transactions-for-data-integrity

// testBundleMarshalling tests marshalling and unmarshalling of a bundle
func testBundleMarshalling(bundle *bcr_gp.ContainedResource, isRequest bool) *bcr_gp.ContainedResource {
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
		fmt.Printf("Unmarshalled Bundle contains %d entries\n", len(unmarshalledBundle.Entry))
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

	// Create multiple users in a transaction
	user1 := &u_gp.User{
		FirstName:     &dt.String{Value: "Transaction"},
		LastName:      &dt.String{Value: "User1"},
		Email:         &dt.String{Value: "txn.user1@test.com"},
		EmailVerified: &dt.Boolean{Value: true},
	}

	user2 := &u_gp.User{
		FirstName:     &dt.String{Value: "Transaction"},
		LastName:      &dt.String{Value: "User2"},
		Email:         &dt.String{Value: "txn.user2@test.com"},
		EmailVerified: &dt.Boolean{Value: false},
	}

	user3 := &u_gp.User{
		FirstName:     &dt.String{Value: "Transaction"},
		LastName:      &dt.String{Value: "User3"},
		Email:         &dt.String{Value: "txn.user3@test.com"},
		EmailVerified: &dt.Boolean{Value: true},
	}

	// Create a Bundle with multiple User resources
	bundle := &bcr_gp.Bundle{
		Type: &bcr_gp.Bundle_TypeCode{
			Value: c_gp.BundleTypeCode_TRANSACTION,
		},
		Entry: []*bcr_gp.Bundle_Entry{
			{
				Resource: &bcr_gp.ContainedResource{
					OneofResource: &bcr_gp.ContainedResource_User{
						User: user1,
					},
				},
				Request: &bcr_gp.Bundle_Entry_Request{
					Method: &bcr_gp.Bundle_Entry_Request_MethodCode{
						Value: c_gp.HTTPVerbCode_POST,
					},
					Url: &dt.Uri{Value: "User"},
				},
			},
			{
				Resource: &bcr_gp.ContainedResource{
					OneofResource: &bcr_gp.ContainedResource_User{
						User: user2,
					},
				},
				Request: &bcr_gp.Bundle_Entry_Request{
					Method: &bcr_gp.Bundle_Entry_Request_MethodCode{
						Value: c_gp.HTTPVerbCode_POST,
					},
					Url: &dt.Uri{Value: "User"},
				},
			},
			{
				Resource: &bcr_gp.ContainedResource{
					OneofResource: &bcr_gp.ContainedResource_User{
						User: user3,
					},
				},
				Request: &bcr_gp.Bundle_Entry_Request{
					Method: &bcr_gp.Bundle_Entry_Request_MethodCode{
						Value: c_gp.HTTPVerbCode_POST,
					},
					Url: &dt.Uri{Value: "User"},
				},
			},
		},
	}

	// Create the contained resource bundle for marshalling
	containedBundle := &bcr_gp.ContainedResource{
		OneofResource: &bcr_gp.ContainedResource_Bundle{
			Bundle: bundle,
		},
	}

	// Test marshalling and unmarshalling the transaction Bundle
	testBundleMarshalling(containedBundle, true)

	// Send bundle in contained resource
	result, err := m.ExecuteBatch(nil, containedBundle)
	if err != nil {
		fmt.Println("Unable to execute batch: " + err.Error())
		os.Exit(1)
	}

	// Test marshalling and unmarshalling the response Bundle
	testBundleMarshalling(result.ContainedResource, false)

	// Check transaction results
	resultBundle := result.ContainedResource.GetBundle()
	if resultBundle == nil {
		fmt.Println("Unexpected: result bundle is nil")
		os.Exit(1)
	}

	fmt.Printf("Transaction processed %d entries\n", len(resultBundle.Entry))

	successCount := 0
	failureCount := 0

	// Check each response entry
	for i, entry := range resultBundle.Entry {
		fmt.Printf("\n--- Entry %d Results ---\n", i+1)

		if entry.Response != nil {
			status := entry.Response.Status.Value
			fmt.Printf("HTTP Status: %s\n", status)

			if status == "201" || status == "200" {
				// Success case - extract created user
				successCount++
				if entry.GetResource() != nil {
					userRes := entry.GetResource().GetUser()
					if userRes != nil {
						fmt.Printf("✅ Created User: ID=%s, Name=%s %s, Email=%s, Verified=%v\n",
							userRes.Id.Value,
							userRes.FirstName.Value,
							userRes.LastName.Value,
							userRes.Email.Value,
							userRes.EmailVerified.Value,
						)
					} else {
						fmt.Println("⚠️  Success status but no user resource returned")
					}
				} else {
					fmt.Println("⚠️  Success status but no resource returned")
				}
			} else {
				// Failed
				failureCount++
				fmt.Printf("❌ Failed with status %s\n", status)

				if entry.Response.Outcome != nil {
					outcome := entry.Response.Outcome.GetOperationOutcome()
					if outcome != nil && len(outcome.Issue) > 0 {
						issue := outcome.Issue[0]
						fmt.Printf("Error: %s - %s\n",
							issue.Code.Value,
							issue.Details.Text.Value)
					}
				}

				if status == "403" {
					fmt.Println("This likely means the client doesn't have permission to create User resources")
				}
			}
		} else {
			failureCount++
			fmt.Println("❌ No response information available")
		}
	}

	fmt.Printf("\n=== Transaction Summary ===\n")
	fmt.Printf("Total entries: %d\n", len(resultBundle.Entry))
	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failureCount)

	if successCount > 0 {
		fmt.Println("✅ Some users were created successfully")
	}
	if failureCount > 0 {
		fmt.Println("❌ Some user creations failed")
		fmt.Println("This is likely due to permission restrictions on User resource creation")
	}

	os.Exit(0)
}
