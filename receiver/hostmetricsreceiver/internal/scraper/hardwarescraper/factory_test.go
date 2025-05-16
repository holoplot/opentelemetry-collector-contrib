// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hardwarescraper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/scraper/scrapertest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/hardwarescraper/internal/metadata"
)

func TestCreateHardwareScraper(t *testing.T) {
	factory := NewFactory()
	cfg := &Config{}

	scraper, err := factory.CreateMetrics(context.Background(), scrapertest.NewNopSettings(metadata.Type), cfg)

	assert.NoError(t, err)
	assert.NotNil(t, scraper)
}
