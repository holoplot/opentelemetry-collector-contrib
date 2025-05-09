// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hardwarescraper // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/hardwarescraper"

import (
	"context"
	"errors"
	"runtime"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/scraper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/hardwarescraper/internal/metadata"
)

// NewFactory for System scraper.
func NewFactory() scraper.Factory {
	return scraper.NewFactory(metadata.Type, createDefaultConfig, scraper.WithMetrics(createMetricsScraper, metadata.MetricsStability))
}

// createDefaultConfig creates the default configuration for the Scraper.
func createDefaultConfig() component.Config {
	return &Config{
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}
}

// createMetricsScraper creates a resource scraper based on provided config.
func createMetricsScraper(
	ctx context.Context,
	settings scraper.Settings,
	cfg component.Config,
) (scraper.Metrics, error) {
	if runtime.GOOS != "linux" {
		return nil, errors.New("hwmon scraper only available on Linux")
	}

	hardwareScraper := newHardwareScraper(ctx, settings, cfg.(*Config))

	return scraper.NewMetrics(
		hardwareScraper.scrape,
		scraper.WithStart(hardwareScraper.start),
	)
}
