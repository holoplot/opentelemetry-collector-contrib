// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hardwarescraper // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/hardwarescraper"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/scraper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/hardwarescraper/internal/metadata"
)

// scraper for Hardware Metrics
type hardwareScraper struct {
	settings scraper.Settings
	config   *Config
	mb       *metadata.MetricsBuilder

	// for mocking
	temperature func(context.Context) (float64, error)
	humidity    func(context.Context) (float64, error)
}

// newHardwareScraper creates an Uptime related metric
func newHardwareScraper(_ context.Context, settings scraper.Settings, cfg *Config) *hardwareScraper {
	return &hardwareScraper{settings: settings, config: cfg}
}

func (s *hardwareScraper) start(_ context.Context, _ component.Host) error {
	s.mb = metadata.NewMetricsBuilder(s.config.MetricsBuilderConfig, s.settings)
	return nil
}

func (s *hardwareScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	now := pcommon.NewTimestampFromTime(time.Now())

	h, err := s.humidity(ctx)
	if err != nil {
		return pmetric.NewMetrics(), err
	}

	t, err := s.temperature(ctx)
	if err != nil {
		return pmetric.NewMetrics(), err
	}

	s.mb.RecordHardwareHumidityDataPoint(now, h)
	s.mb.RecordHardwareTemperatureDataPoint(now, t)

	return s.mb.Emit(), nil
}
