# CircleCI Terraform Provider
The CircleCI Terraform Provider enable customers to manage CircleCI projects with IaC patterns, matching the same patterns used to manage GitHub repos. For large-scale organizations, this enables automated project creation for new teams or projects.

## Usage
The current documentation is found [here](https://registry.terraform.io/providers/CircleCI-Public/circleci/latest/docs).
Define the provider:
```hcl
terraform {
  required_providers {
    circleci = {
      source = "CircleCI-Public/circleci"
      version = "0.1.1"
    }
  }
}

provider "circleci" {
  host = "https://circleci.com/api/v2"
  #key         = "*****" # here you set you CircleCI API Key
  #runner_host = "https://runner.circleci.com" # required for self-hosted runner resources on CircleCI server
}
```
Start defining Data Sources and Resources. For example:
```hcl
resource "circleci_project" "test_project" {
  name 				= "Project_Name"
  organization_id 	= "********-****-****-****-****************"
}
```

Use the Official [CircleCI API documentation](https://circleci.com/docs/api/v2/index.html) to check which valid values might be needed for some resources.

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

## Forgejo acceptance tests

`TestAccForgejoLive` runs against a real Forgejo instance, set up as the [Forgejo guide](docs/guides/forgejo.md) describes, and skips unless `CIRCLE_FORGEJO_URL` is set. It creates and deletes the test organization's Forgejo connection, deleting one a crashed run left behind first, so never run it concurrently against the same organization, and keep no hand-made Forgejo connection there. It reads these variables:

| Variable | Value |
|---|---|
| `CIRCLE_FORGEJO_URL` | The Forgejo instance's base URL |
| `CIRCLE_FORGEJO_OWNER` | The Forgejo organization that owns the test repository |
| `CIRCLE_FORGEJO_CONNECTION_AUDIENCE` | The audience of the connection Authorized Integration |
| `CIRCLE_FORGEJO_CHECKOUT_AUDIENCE` | The audience of the checkout Authorized Integration |
| `CIRCLE_FORGEJO_REPO` | The test repository's full name, `owner/name` |
| `CIRCLE_FORGEJO_OTHER_REPO` | Optional. A second repository's full name, to test moving the pipeline to it |
| `CIRCLE_FORGEJO_ORG_ID` | Optional. The CircleCI organization; defaults to the shared test organization |
| `CIRCLE_FORGEJO_REQUEST_HEADERS` | Optional. A JSON object of headers to send to the instance |

```sh
TF_ACC=1 task test -- -run '^TestAccForgejoLive$' ./internal/provider/
```
