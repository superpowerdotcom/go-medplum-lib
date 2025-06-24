package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	dt "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	cr "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
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

	// Create update transaction
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

	updateResult, err := m.ExecuteBatch(nil, &cr.ContainedResource{
		OneofResource: &cr.ContainedResource_Bundle{
			Bundle: updateBundle,
		},
	})
	if err != nil {
		fmt.Println("Unable to execute update transaction: " + err.Error())
		os.Exit(1)
	}

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
