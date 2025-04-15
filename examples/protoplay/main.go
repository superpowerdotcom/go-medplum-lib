package main

import (
	"fmt"
	"log"
	"os"

	"github.com/google/fhir/go/fhirversion"
	"github.com/google/fhir/go/jsonformat"
	bcr_gp "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
)

func main() {
	// Read contents of file
	f, err := os.ReadFile("careplan.json")
	if err != nil {
		log.Fatal("Unable to read file: ", err)
	}

	// bundle := &bcr_gp.Bundle{}

	unmarshaller, err := jsonformat.NewUnmarshaller("UTC", fhirversion.R4)
	if err != nil {
		log.Fatal("Unable to create unmarshaller: ", err)
	}

	protoMsg, err := unmarshaller.Unmarshal(f)
	if err != nil {
		log.Fatal("Unable to unmarshal JSON: ", err)
	}

	fmt.Printf("Proto message: %+v\n", protoMsg)

	cr, ok := protoMsg.(*bcr_gp.ContainedResource)
	if !ok {
		log.Fatal("Unable to type assert to ContainedResource")
	}

	if cr == nil {
		log.Fatal("ContainedResource is nil")
	}

	if cr.GetBundle() == nil {
		log.Fatal("Bundle is nil")
	}

	crBundle := cr.GetBundle()

	for _, entry := range crBundle.GetEntry() {
		if entry == nil {
			log.Fatal("Entry is nil")
		}

		resource := entry.GetResource()
		if resource == nil {
			log.Fatal("Resource is nil")
		}

		fmt.Printf("Resource: %+v\n", resource)
	}

	//cr := &bcr_gp.ContainedResource{
	//	OneofResource: &bcr_gp.ContainedResource_Bundle{
	//		Bundle:
	//	}
	//}
}
