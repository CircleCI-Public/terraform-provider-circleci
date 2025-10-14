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
  #key  = "*****" # here you set you CircleCI API Key
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

## Upcoming Features in the Next Release

- Import existing resources
- New Data sources and resources:
    - Organizations
    - Webhooks

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
