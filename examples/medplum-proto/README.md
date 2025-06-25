# Medplum protobuf testing suite

This folder tests both our custom-implemented protos for Medplum's `User` resource from our `superpower/fhir` lib using `go-medplum-lib`. 
More resources will be added later as we add proto support for them.

## Notes on the User resource

In order to work with the User resource, you need to initialize a Medplum Client made inside a Super Admin project or you will get a `403 Forbidden`. This is already done for you inside each `main.go` file.

## How to run?

cd into `medplum-proto` from root and run:

```
go run .
```
