// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hardwarescraper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/scraper/scrapertest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/hardwarescraper/internal/metadata"
)

func TestScrape(t *testing.T) {
	ctx := context.Background()

	s := newHardwareScraper(ctx, scrapertest.NewNopSettings(metadata.Type), &Config{
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	})
	require.NoError(t, s.start(ctx, componenttest.NewNopHost()))

	_, err := s.scrape(ctx)
	require.NoErrorf(t, err, "scrape error %v", err)
}
