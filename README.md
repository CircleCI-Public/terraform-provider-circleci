# CircleCI Terraform Provider
The CircleCI Terraform Provider enable customers to manage CircleCI projects with IaC patterns, matching the same patterns used to manage GitHub repos. For large-scale organizations, this enables automated project creation for new teams or projects.

## Upcoming Features in the Next Release

- Import existing resources

## Important Notice: Breaking Change
The [breaking change presented on May 27, 2025](https://discuss.circleci.com/t/upcoming-changes-1-breaking-to-crud-trigger-v2-apis-may-27-2025/53314) in the public-api-service that affects the Get Trigger operation is a side effect to this provider. We are working to fix this on the next release.

Impact: name and description will be removed from the API response for the getTrigger endpoint

Timeline: We are actively working on compatibility updates

ETA: Fix expected in next release

## Acknowledgments
This repository was created following the Terraform plugin framework defined by Hashicorp [here](https://developer.hashicorp.com/terraform/plugin/framework).

# Development

This repository makes use of [Task](https://taskfile.dev/#/). It may be installed (on MacOS) with:
```
$ brew install go-task/tap/go-task
```

See the full list of available tasks by running `task -l`, or, see the [Taskfile.yml](./Taskfile.yml) script.

```sh
task lint
task fmt
task generate

# Run all the tests
task test
# Run the tests for one package
task test -- ./client/...
# Run all the quick tests
task test -- -short ./...
