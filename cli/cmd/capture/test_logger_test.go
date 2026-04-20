// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package capture

import (
	"testing"

	"github.com/microsoft/retina/pkg/log"
)

func testLogger(t *testing.T) *log.ZapLogger {
	t.Helper()

	logger, err := log.SetupZapLogger(log.GetDefaultLogOpts())
	if err != nil {
		t.Fatalf("failed to set up logger: %v", err)
	}

	return logger.Named("capture-test")
}
