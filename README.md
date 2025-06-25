<p align="center">
  <img src="https://raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png" style="width: 10%; height: 10%; display: inline-block;" alt="Gopher Logo">
  +
  <img src="https://www.hl7.org/fhir/assets/images/fhir-logo-www.png" style="width: 50%; height: 50%; display: inline-block;" alt="FHIR Logo">
</p>

go-medplum-lib
==============

A Go library for interfacing with Medplum using [Google's FHIR protos](https://github.com/superpowerdotcom/fhir).

## Why?

The FHIR specification is vast and intricate, comprising numerous complex resources, profiles, and interactions. Crafting accurate JSON requests by hand is not only time-consuming but also prone to errors, making it easy to misstep and create non-compliant or invalid FHIR resources.

This library was created to simplify this process.

By leveraging Google’s FHIR Protobufs, developers can manage FHIR resources programmatically, ensuring that requests are both syntactically correct and compliant with the FHIR standard. This approach eliminates the need for manual JSON crafting, reducing the potential for errors and speeding up development. Whether you’re building healthcare applications, integrating with FHIR-compliant systems, or just need to interact with Medplum’s API, this library provides a robust, type-safe way to work with FHIR resources in Go.

## Features

* **Type-Safe FHIR Resource Management**: Create, read, update, and delete FHIR resources using Google's Protobufs.
* **Simplified Interaction with Medplum**: High-level abstractions over the Medplum API make it easy to integrate into Go applications.
* **Error Reduction**: Eliminates common mistakes in hand-crafting FHIR JSON, ensuring compliance with the FHIR standard.
* **Search and Query Support**: Perform complex FHIR-compliant searches and queries with ease.
* **Support for Binary Resources**: Upload and manage binary data like PDFs and images associated with FHIR resources.
* **Support for Transactions**: Atomically create resources and their refs.

## Installation

Install the library using go get:

```bash
$ go get github.com/superpowerdotcom/go-medplum-lib
```

Then, import it into your Go code:

```go
import (
"github.com/superpowerdotcom/go-medplum-lib"
)
```

## Usage
Refer to the [./examples](./examples) directory for example code demonstrating how to create, read, search, update, and delete resources using this library.

### Basic Example
Here's a basic example to get you started:

```go
package main

import (
    "fmt"
    "os"

    dt "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
    cr "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
    "github.com/superpowerdotcom/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"

    "github.com/superpowerdotcom/go-medplum-lib"
)

func main() {
    // Authenticate
    m, err := medplum.New(&medplum.Options{
        MedplumURL:   "http://localhost:8103",
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
    })

    if err != nil {
        fmt.Println("unable to create medplum client: " + err.Error())
        os.Exit(1)
    }

    // Create a patient
    patient := &patient_go_proto.Patient{
        Id: &dt.Id{Value: "12345"}, // Will be ignored by server and a new ID will be generated
        Name: []*dt.HumanName{
            {
                Text: &dt.String{Value: "Example Patient"},
            },
        },
    }

    // Put the patient inside of a ContainedResource
    patientCR := &cr.ContainedResource{
        OneofResource: &cr.ContainedResource_Patient{
            Patient: patient,
        },
    }

    // Create it via medplum client
    result, err := m.CreateResource(patientCR)
    if err != nil {
        fmt.Println("Unable to create patient resource: " + err.Error())
        os.Exit(1)
    }

    // Inspect patient details
    fmt.Printf("[safe] Patient ID: %s\n", result.ContainedResource.GetPatient().Id.Value)
}
```

## Advanced Usage

For more advanced usage, such as handling binary resources and linking them to 
`DocumentReference`, how to perform transactions and more, refer to the detailed
examples in the [./examples](./examples) directory.

## Contributing

Contributions are welcome! If you have suggestions for improvements or have found a bug, please open an issue or submit a pull request.

## Development Setup

1. Fork the repository and clone it locally.
1. Install dependencies using `go mod tidy`.
1. Run tests using `go test ./...` to ensure everything is working as expected.

## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Acknowledgments

* **Google FHIR Protos**: This library heavily relies on Google's FHIR Protos for resource definitions and handling.
* **Medplum**: Special thanks to the Medplum team for providing a robust and flexible FHIR API.
