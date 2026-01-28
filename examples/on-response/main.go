package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"

	"github.com/superpowerdotcom/go-medplum-lib"
)

// This example demonstrates how to use the OnResponse callback to debug
// protobuf unmarshal errors.
//
// When Medplum returns FHIR data that doesn't conform to protobuf schemas,
// the library fails to unmarshal and you may see errors like:
//
//   go-medplum-lib: unable to unmarshal response body using FHIR protos:
//   error at "Bundle.entry[190].resource.ofType(QuestionnaireResponse).item[5].item[5].answer[0].valueInteger"
//
// If you are doing a Search() that returns many resources, it can be *really*
// difficult to find the specific resource that is causing the problem.
//
// The OnResponse callback gives you access to the raw response body so you
// can inspect what Medplum actually returned.

func main() {
	m, err := medplum.New(&medplum.Options{
		MedplumURL:   "http://localhost:8103",
		ClientID:     "3008218e-5de9-4398-a987-ca393e3e64b0",
		ClientSecret: "1b6b7708423fa6cc589d2996e40d35bc2ba38d6af366e16660bcfcecb5438896",

		// OnResponse is called after every generateResult() call.
		// Use it to inspect raw responses, especially when unmarshal fails.
		OnResponse: func(resp *http.Response, body []byte, err error) {
			if err != nil {
				// Log the error
				log.Printf("Unmarshal error: %v", err)

				// Log the raw response body for debugging
				log.Printf("Raw response body:\n%s", string(body))

				// You could also:
				// - Send to monitoring/alerting
				// - Write to a debug file
				// - Parse the JSON manually to extract specific fields
			}

			// You can also log successful responses if needed
			if resp != nil {
				log.Printf("Response status: %d %s", resp.StatusCode, resp.Status)
			}
		},
	})

	if err != nil {
		fmt.Println("unable to create medplum client: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("Successfully authenticated")

	// Perform a search - if any responses fail to unmarshal, OnResponse will log them
	result, err := m.Search(nil, codes_go_proto.ResourceTypeCode_PATIENT, "_count=10")
	if err != nil {
		fmt.Println("Search failed: " + err.Error())
		os.Exit(1)
	}

	bundle := result.ContainedResource.GetBundle()
	if bundle != nil {
		fmt.Printf("Found %d patients\n", len(bundle.Entry))
	}
}
