// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name openmetadata

package main

import (
	"context"
	"log"

	"github.com/Codility/terraform-provider-openmetadata/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version is set by goreleaser via ldflags.
var version string = "dev"

func main() {
	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/codility/openmetadata",
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
