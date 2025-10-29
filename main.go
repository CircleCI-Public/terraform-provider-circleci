// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"terraform-provider-circleci/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	version string = "dev"

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			// Print the panic reason and the stack trace to standard error
			fmt.Fprintf(os.Stderr, "!!! PROVIDER PANIC CAUGHT: %v\n", r)
			os.Stderr.Write(debug.Stack())
			// Exit with a non-zero code to signal failure
			os.Exit(1)
		}
	}()

	opts := providerserver.ServeOpts{
		// TODO: Update this string with the published name of your provider.
		// Also update the tfplugindocs generate command to either remove the
		// -provider-name flag or set its value to the updated provider name.
		Address: "registry.terraform.io/circleci/circleci",
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
