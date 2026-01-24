# Changelog

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
