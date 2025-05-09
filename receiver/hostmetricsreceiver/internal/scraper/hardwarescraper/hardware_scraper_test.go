// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hardwarescraper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/scraper/scrapertest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/hardwarescraper/internal/metadata"
)

func TestScrape(t *testing.T) {
	ctx := context.Background()

	s := newHardwareScraper(ctx, scrapertest.NewNopSettings(metadata.Type), &Config{
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	})

	s.sensors[0].mock(func(_ context.Context, _ string) (float64, error) {
		return float64(23), nil
	})
	s.sensors[1].mock(func(_ context.Context, _ string) (float64, error) {
		return float64(42), nil
	})

	require.NoError(t, s.start(ctx, componenttest.NewNopHost()))

	metrics, err := s.scrape(ctx)
	require.NoErrorf(t, err, "scrape error %v", err)

	assert.Equal(t, 2, metrics.MetricCount())
	assert.Equal(t, 2, metrics.DataPointCount())

	for _, m := range metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().All() {
		assert.Equalf(t, pmetric.MetricTypeGauge, m.Type(), "invalid metric type: %v", m.Type())
		switch m.Name() {
		case "hardware.temperature":
			assert.Equalf(t, "Cel", m.Unit(), "invalid metric unit: %s", m.Unit())

			dataPoint := m.Gauge().DataPoints().At(0)
			assert.Equal(t, float64(23), dataPoint.DoubleValue())

		case "hardware.humidity":
			assert.Equalf(t, "%", m.Unit(), "invalid metric unit: %s", m.Unit())

			dataPoint := m.Gauge().DataPoints().At(0)
			assert.Equal(t, float64(42), dataPoint.DoubleValue())

		default:
			assert.Fail(t, "should not be reached", "unexpected metric %s", m.Name())
		}
	}
}
