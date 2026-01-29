# Changelog

## v1.0.7 - 2025-01-28

### New Features

- **`OnResponse` callback**: Added optional `OnResponse` callback to `Options` that fires after every call to `generateResult()`
  - Receives the HTTP response, body bytes, and any error that occurred
  - Useful for debugging protobuf unmarshal failures, monitoring, or custom error handling
  - See "OnResponse Callback" section in README.md for usage examples

### Other Changes

- `Options.Log` field type changed from `clog.ICustomLog` to `medplum.Logger`
  - The `Logger` interface is identical to `clog.ICustomLog` and is now defined locally
  - This removes the dependency on the private `go-common-lib` repo
  - Existing code using `clog.ICustomLog` will continue to work (interface is compatible)
- Removed unused `ErrBundleCannotBeNil` and `ErrBundleEntryCannotBeEmpty` error variables

## v1.0.6 - 2025-01-24

### Dependencies

- Bumped `github.com/superpowerdotcom/fhir/go` from v0.0.10 to v0.2.1

## v1.0.5 - 2025-01-24

### New Features

- **Optional `clog.ICustomLogger`**: Allow passing in an optional custom logger
  - If not present, will default to `log.Println()` (like before)

## v1.0.3 - 2025-01-06

### Breaking Changes

- `Result.RawHTTPResponse` renamed to `Result.RawHTTPResponses` (now a slice)
  - For single-request methods, use `result.RawHTTPResponses[0]` instead of `result.RawHTTPResponse`

### New Features

- **Auto-pagination for Search()**: The `Search()` method now automatically follows all pagination links and returns a combined Bundle with all entries
  - No code changes required for callers - just get all results instead of first page
  - All HTTP responses collected in `RawHTTPResponses` slice

### Migration

If you access `RawHTTPResponse` directly, update to:

```go
// Before
result.RawHTTPResponse.StatusCode

// After
result.RawHTTPResponses[0].StatusCode
```
