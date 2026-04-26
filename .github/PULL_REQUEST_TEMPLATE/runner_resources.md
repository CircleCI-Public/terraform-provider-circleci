## Summary

<!-- Briefly describe what runner infrastructure this PR adds or changes. -->

## Changes

### New Components

<!-- List each new resource, data source, or ephemeral resource. For each one include:
     - The Terraform type name (e.g. `circleci_runner_token`)
     - CRUD operations implemented
     - Any notable behaviour (write-once fields, force-delete, etc.)
     - Import support and the import ID format -->

### Provider Integration

<!-- Confirm provider.go changes:
     - New SDK service field added to CircleCiClientWrapper
     - Service initialized in Configure()
     - Resource/data source registered in Resources() or DataSources() -->

### SDK Dependency

<!-- Does this require the local circleci-sdk-go replace directive in go.mod?
     If so, note which package(s) and when they are expected to be published remotely. -->

## Schema Reference

<!-- Add a table for each new resource/data source. -->

### `circleci_runner_<name>`

| Attribute | Type | Category | Description |
| :--- | :--- | :--- | :--- |
| `id` | String | Computed | UUID assigned by the API |
| | | | |

## Example Usage

```hcl
resource "circleci_runner_resource_class" "example" {
  resource_class = "myorg/myrunner"
  description    = "My self-hosted runner"
}

resource "circleci_runner_token" "example" {
  resource_class = circleci_runner_resource_class.example.resource_class
  nickname       = "my-agent-token"
}
```

## Testing

- [ ] `go build ./...` passes
- [ ] Acceptance tests require a CircleCI token with **self-hosted runner admin** permissions (separate scope from the standard `CIRCLE_TOKEN` used by other tests)
- [ ] `TF_ACC=1 CIRCLE_TOKEN=<runner-admin-token> go test ./internal/provider/ -run TestAccRunner -v -timeout 120s`
- [ ] Create and Read verified against live API
- [ ] Delete (and force-delete if applicable) verified
- [ ] ImportState verified

## Notes

<!-- Anything reviewers should be aware of — API quirks, missing endpoints, known limitations. -->
