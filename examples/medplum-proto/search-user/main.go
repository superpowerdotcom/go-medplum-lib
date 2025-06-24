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

	// Create a user to search for
	testEmail := "search.testuser@example.com"
	userID, err := createUser(m, testEmail)
	if err != nil {
		fmt.Println("Unable to create user for searching: " + err.Error())
		os.Exit(1)
	}

	fmt.Printf("Created user with ID: %s and email: %s\n", userID, testEmail)

	// Search for the user by email
	result, err := m.Search(nil, codes_go_proto.ResourceTypeCode_USER, "email="+testEmail)
	if err != nil {
		fmt.Println("Unable to search for users: " + err.Error())
		os.Exit(1)
	}

	searchBundle := result.ContainedResource.GetBundle()
	if searchBundle == nil {
		fmt.Println("Search did not return a Bundle")
		os.Exit(1)
	}

	fmt.Printf("Search returned %d results\n", len(searchBundle.Entry))

	// Print search results
	for i, entry := range searchBundle.Entry {
		if entry.GetResource() != nil {
			userRes := entry.GetResource().GetUser()
			if userRes != nil {
				fmt.Printf("Result %d: ✅ Found User: ID=%s, Name=%s %s, Email=%s\n",
					i+1,
					userRes.Id.Value,
					userRes.FirstName.Value,
					userRes.LastName.Value,
					userRes.Email.Value,
				)
			}
		}
	}

	if len(searchBundle.Entry) == 0 {
		fmt.Println("⚠️  No users found with that email. This may be due to indexing delays or search permissions.")
	}

	os.Exit(0)
}

func createUser(m *medplum.Medplum, email string) (string, error) {
	user := &u_gp.User{
		FirstName:     &dt.String{Value: "Search"},
		LastName:      &dt.String{Value: "TestUser"},
		Email:         &dt.String{Value: email},
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
