# Precognition Support Proposal

## Goal

Add Laravel/Inertia Precognition validation support to `inertiaframe` without adding a second validation path or exposing request-level Precognition APIs.

## Public API

The only new public API is route-level opt-in through `Meta`:

```go
func (Endpoint) Meta() inertiaframe.Meta {
	return inertiaframe.Meta{
		Method:       http.MethodPost,
		Path:         "/users",
		Precognition: true,
	}
}
```

Validation remains a single entrypoint through `MountConfig.Validator`.

## Internal Behavior

Protocol-level helpers live in `internal/inertiaprecognition` rather than `inertiaframe`. This keeps Precognition request detection, response writing, `Vary` handling, and `Precognition-Validate-Only` filtering reusable by the root package without exposing a public root API.

When `Meta.Precognition` is true and the request includes `Precognition: true`, `inertiaframe` will:

- Decode the request body normally.
- Run the configured `Validator` normally.
- Return `204 No Content` with `Precognition: true` and `Precognition-Success: true` when validation passes.
- Return `422 Unprocessable Entity` with a JSON `errors` object when validation fails with an `inertia.ValidationErrorer`.
- Skip `Endpoint.Execute` for handled Precognition requests to avoid side effects.
- Add `Vary: Precognition` to Precognition responses.
- Filter returned validation errors using `Precognition-Validate-Only` when the client asks for specific fields.

When `Meta.Precognition` is false, a request with `Precognition: true` is treated like a normal request.

## Error Payload

Precognition validation errors are returned as arrays of messages per field:

```json
{
  "errors": {
    "email": ["Email is invalid", "Email must be unique"]
  }
}
```

This preserves all backend validation messages and lets the Inertia client decide whether to simplify to the first message.

## Non-Goals

- No `RequestValidator` or request-aware validator interface.
- No exported `IsPrecognitive`, `PrecognitionValidateOnly`, or response-writing helpers.
- No endpoint execution during supported Precognition validation requests.
- No custom rule system for field-level validation; field filtering is applied to returned validation errors only.

## Files

- `internal/inertiaheader/header.go`: adds Precognition protocol header constants.
- `internal/inertiaprecognition/precognition.go`: contains shared internal Precognition request detection, response writing, and field filtering.
- `inertiaframe/inertiaframe.go`: wires route opt-in into the handler lifecycle.
- `inertiaframe/inertiaframe_test.go`: covers success, validation failure, field filtering, and unsupported-route behavior.
