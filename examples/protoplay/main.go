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
)

func main() {
	// Test 1: Create and marshal complex User to JSON file
	testCreateComplexUser()

	// Test 2: Unmarshal from JSON file
	testUnmarshalComplexUser()
}

func testCreateComplexUser() {
	fmt.Println("=== Test 1: Create and Marshal Complex User to JSON ===")

	// Create a complex User resource
	user := &u_gp.User{
		FirstName: &dt.String{Value: "Audric"},
		LastName:  &dt.String{Value: "Serador"},
		Email:     &dt.String{Value: "audric-admin@superpower.com"},
		Id:        &dt.Id{Value: "635d69da-4259-4c8e-952b-95926dce7674"},
		PasswordHash: &dt.String{
			Value: "$2a$10$pen5UIwTm/SndqwDb8PWIeWpeNIY9id1tWpLutHw08AVi42G992..",
		},
		Identifier: []*dt.Identifier{
			{
				System: &dt.Uri{Value: "https://superpower.com/fhir/StructureDefinition/user-type"},
				Value:  &dt.String{Value: "admin"},
			},
			{
				System: &dt.Uri{Value: "https://superpower.com/fhir/StructureDefinition/user-source"},
				Value:  &dt.String{Value: "seed"},
			},
		},
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
	}

	// Create a Bundle containing the User entry
	bundle := &bcr_gp.Bundle{
		Type: &bcr_gp.Bundle_TypeCode{
			Value: c_gp.BundleTypeCode_COLLECTION,
		},
		Entry: []*bcr_gp.Bundle_Entry{entry},
	}

	// Wrap the Bundle in a ContainedResource
	containedBundle := &bcr_gp.ContainedResource{
		OneofResource: &bcr_gp.ContainedResource_Bundle{
			Bundle: bundle,
		},
	}

	// Marshal the ContainedResource (Bundle) to JSON using the FHIR marshaller
	marshaller, err := jsonformat.NewMarshaller(false, "", "  ", fhirversion.R4)
	if err != nil {
		log.Fatal("Unable to create marshaller: ", err)
	}

	jsonBytes, err := marshaller.Marshal(containedBundle)
	if err != nil {
		log.Fatal("Unable to marshal Bundle: ", err)
	}

	fmt.Println("Marshalled JSON:")
	fmt.Println(string(jsonBytes))

	// Re-format the JSON with proper indentation for file writing
	var jsonObj map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonObj)
	if err != nil {
		log.Fatal("Unable to parse JSON for reformatting: ", err)
	}

	prettyJSON, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		log.Fatal("Unable to format JSON: ", err)
	}

	// Write formatted JSON to file
	err = os.WriteFile("complex-user.json", prettyJSON, 0644)
	if err != nil {
		log.Fatal("Unable to write JSON file: ", err)
	}
	fmt.Println("Formatted JSON written to complex-user.json")

	// Test unmarshaling the same JSON
	unmarshaller, err := jsonformat.NewUnmarshaller("UTC", fhirversion.R4)
	if err != nil {
		log.Fatal("Unable to create unmarshaller: ", err)
	}

	protoMsg, err := unmarshaller.Unmarshal(jsonBytes)
	if err != nil {
		log.Fatal("Unable to unmarshal JSON: ", err)
	}

	fmt.Printf("Unmarshalled proto message: %+v\n", protoMsg)

	// Type assert to ContainedResource and print the User fields
	cr, ok := protoMsg.(*bcr_gp.ContainedResource)
	if !ok {
		log.Fatal("Unable to type assert to ContainedResource")
	}

	if cr.GetBundle() == nil {
		log.Fatal("Bundle is nil")
	}

	for _, entry := range cr.GetBundle().GetEntry() {
		if entry == nil || entry.GetResource() == nil {
			continue
		}
		userRes := entry.GetResource().GetUser()
		if userRes != nil {
			fmt.Printf("User: FirstName=%s, LastName=%s, Email=%s, ID=%s\n",
				userRes.GetFirstName().GetValue(),
				userRes.GetLastName().GetValue(),
				userRes.GetEmail().GetValue(),
				userRes.GetId().GetValue(),
			)

			// Print identifiers
			for i, ident := range userRes.GetIdentifier() {
				fmt.Printf("  Identifier[%d]: System=%s, Value=%s\n",
					i,
					ident.GetSystem().GetValue(),
					ident.GetValue().GetValue(),
				)
			}
		}
	}
	fmt.Println()
}

func testUnmarshalComplexUser() {
	fmt.Println("=== Test 2: Unmarshal Complex User from JSON File ===")

	// Read JSON from file
	jsonBytes, err := os.ReadFile("complex-user.json")
	if err != nil {
		log.Fatal("Unable to read JSON file: ", err)
	}

	fmt.Println("Read JSON from file:")
	fmt.Println(string(jsonBytes))

	// Unmarshal the JSON back to a proto message
	unmarshaller, err := jsonformat.NewUnmarshaller("UTC", fhirversion.R4)
	if err != nil {
		log.Fatal("Unable to create unmarshaller: ", err)
	}

	protoMsg, err := unmarshaller.Unmarshal(jsonBytes)
	if err != nil {
		log.Fatal("Unable to unmarshal JSON: ", err)
	}

	fmt.Printf("Unmarshalled proto message: %+v\n", protoMsg)

	// Type assert to ContainedResource and print the User fields
	cr, ok := protoMsg.(*bcr_gp.ContainedResource)
	if !ok {
		log.Fatal("Unable to type assert to ContainedResource")
	}

	if cr.GetBundle() == nil {
		log.Fatal("Bundle is nil")
	}

	fmt.Printf("Bundle type: %s\n", cr.GetBundle().GetType().GetValue())

	for _, entry := range cr.GetBundle().GetEntry() {
		if entry == nil || entry.GetResource() == nil {
			continue
		}
		userRes := entry.GetResource().GetUser()
		if userRes != nil {
			fmt.Printf("User: FirstName=%s, LastName=%s, Email=%s, ID=%s\n",
				userRes.GetFirstName().GetValue(),
				userRes.GetLastName().GetValue(),
				userRes.GetEmail().GetValue(),
				userRes.GetId().GetValue(),
			)

			// Print identifiers
			for i, ident := range userRes.GetIdentifier() {
				fmt.Printf("  Identifier[%d]: System=%s, Value=%s\n",
					i,
					ident.GetSystem().GetValue(),
					ident.GetValue().GetValue(),
				)
			}

			// Print meta information
			if userRes.GetMeta() != nil {
				meta := userRes.GetMeta()
				fmt.Printf("  Meta: VersionId=%s\n",
					meta.GetVersionId().GetValue(),
				)
			}
		}
	}
}
