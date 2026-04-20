// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.
package main

import (
	"fmt"
	"os"

	"github.com/microsoft/retina/cli/cmd"
	"github.com/microsoft/retina/cli/cmd/capture"
	"github.com/microsoft/retina/pkg/log"
)

func main() {
	_, err := log.SetupZapLogger(log.GetDefaultLogOpts())
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	kubeClient, err := capture.GetClientset()
	if err != nil {
		fmt.Printf("Failed to get Kubernetes client: %v\n", err)
		os.Exit(1)
	}
	cmd.Retina.AddCommand(capture.NewCommand(kubeClient, log.Logger().Named("retina-cli")))
	if err := cmd.Retina.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
