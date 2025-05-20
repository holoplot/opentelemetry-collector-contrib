// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hardwarescraper

import (
	"math/rand/v2"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/scraper/scrapertest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/hardwarescraper/internal/metadata"
)

type mockDevice struct {
	device string
	name   string
	values []int64
}

func addMockDevice(t *testing.T, rootDir, device string, names ...string) mockDevice {
	m := mockDevice{
		device: device,
		name:   "fake",
		values: make([]int64, len(names)),
	}

	err := os.Mkdir(path.Join(rootDir, device), 0o700)
	require.NoErrorf(t, err, "creating fake device: %v", err)

	err = os.WriteFile(path.Join(rootDir, device, "name"), []byte(m.name), 0o600)
	require.NoErrorf(t, err, "creating fake device: %v", err)

	for i, name := range names {
		m.values[i] = rand.Int64N(100000)

		err := os.WriteFile(path.Join(rootDir, device, name+"_input"), []byte(strconv.FormatInt(m.values[i], 10)), 0o600)
		require.NoErrorf(t, err, "creating fake device: %v", err)
	}

	return m
}

func TestScrape(t *testing.T) {
	ctx := t.Context()

	c := Config{MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig()}
	s := newHardwareScraper(ctx, scrapertest.NewNopSettings(metadata.Type), &c)

	s.rootDir = t.TempDir()

	hwmon0 := addMockDevice(t, s.rootDir, "hwmon0", "temp1", "temp2")
	hwmon1 := addMockDevice(t, s.rootDir, "hwmon1", "fan1")

	require.NoError(t, s.start(ctx, componenttest.NewNopHost()))

	assert.Len(t, s.chips, 2)

	metrics, err := s.scrape(ctx)
	require.NoErrorf(t, err, "scrape error %v", err)

	assert.Equal(t, 2, metrics.MetricCount())
	assert.Equal(t, 2, metrics.ResourceMetrics().Len())
	assert.Equal(t, 3, metrics.DataPointCount())

	metric := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	assert.Equalf(t, pmetric.MetricTypeGauge, metric.Type(), "invalid metric type: %v", metric.Type())

	for i, v := range hwmon0.values {
		dataPoint := metric.Gauge().DataPoints().At(i)
		assert.Equal(t, 0.001*float64(v), dataPoint.DoubleValue())
	}

	metric = metrics.ResourceMetrics().At(1).ScopeMetrics().At(0).Metrics().At(0)
	assert.Equalf(t, pmetric.MetricTypeGauge, metric.Type(), "invalid metric type: %v", metric.Type())

	for i, v := range hwmon1.values {
		dataPoint := metric.Gauge().DataPoints().At(i)
		assert.Equal(t, v, dataPoint.IntValue())
	}
}
