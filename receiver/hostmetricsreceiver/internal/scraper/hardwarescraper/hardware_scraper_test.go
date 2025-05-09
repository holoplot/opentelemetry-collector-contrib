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

	// mock
	s.humidity = func(_ context.Context) (float64, error) {
		return float64(23), nil
	}
	s.temperature = func(_ context.Context) (float64, error) {
		return float64(42), nil
	}

	require.NoError(t, s.start(ctx, componenttest.NewNopHost()))

	metrics, err := s.scrape(ctx)
	require.NoErrorf(t, err, "scrape error %+v", err)

	assert.Equal(t, 2, metrics.MetricCount())
	assert.Equal(t, 2, metrics.DataPointCount())

	metric := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	assert.Equalf(t, pmetric.MetricTypeGauge, metric.Type(), "invalid metric type: %v", metric.Type())

	dataPoint := metric.Gauge().DataPoints().At(0)
	assert.Equal(t, float64(23), dataPoint.DoubleValue())

	metric = metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(1)
	assert.Equalf(t, pmetric.MetricTypeGauge, metric.Type(), "invalid metric type: %v", metric.Type())

	dataPoint = metric.Gauge().DataPoints().At(0)
	assert.Equal(t, float64(42), dataPoint.DoubleValue())
}
