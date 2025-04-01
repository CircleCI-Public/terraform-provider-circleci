# CircleCI Terraform Provider

_This template repository is built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework). 

This provider will be published to [the Terraform Registry](https://developer.hashicorp.com/terraform/registry/providers/publishing) so that others can use it.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.5.7
- [Go](https://golang.org/doc/install) >= 1.24.1

## Building The Provider

1. Create a `.terraformrc` file in your home directory. Please replace the `<GOPATH>` with your actual directory path:
```hcl
provider_installation {

  dev_overrides {
      "registry.terraform.io/circleci/circleci" = "<GOPATH>/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```
2. Clone the repository
3. Enter the repository directory
4. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

The current [CircleCI go client](https://github.com/CircleCI-Public/circleci-sdk-go) implements the following resources:
- Context
- Env
- Pipeline
- Project
- Trigger

This provider implements both data sources and resources for the previously listed objects.

### Initialize the provider

### Context

### Env

### Pipeline

### Project

### Trigger

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
