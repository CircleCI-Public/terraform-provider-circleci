//go:build generate

package main

// Generate copyright headers
//nolint:misspell // this is not a mis-spelling
//go:generate go tool copywrite headers --config .copywrite.hcl

// Format Terraform code for use in documentation.
// If you do not have Terraform installed, you can remove the formatting command, but it is suggested
// to ensure the documentation is formatted properly.
//go:generate terraform fmt -recursive ./examples/

// Generate documentation.
//go:generate go tool tfplugindocs generate --provider-name circleci
