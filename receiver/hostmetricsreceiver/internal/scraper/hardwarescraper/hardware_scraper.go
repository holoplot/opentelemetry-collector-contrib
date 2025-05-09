// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hardwarescraper // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/hardwarescraper"

import (
	"context"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/scraper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/hardwarescraper/internal/metadata"
)

const (
	hwmonRoot = "/sys/class/hwmon"
)

type sensor interface {
	scrape(ctx context.Context, mb *metadata.MetricsBuilder) error
	mock(func(context.Context, string) (float64, error))
}

// reads a temperature sensor from hwmon interface
type temperatureSensor struct {
	path string
	read func(ctx context.Context, path string) (float64, error)
}

func newTemperatureSensor(chip, name string) *temperatureSensor {
	return &temperatureSensor{
		path: path.Join(hwmonRoot, chip, name),
		read: readsensor,
	}
}

func (v *temperatureSensor) scrape(ctx context.Context, mb *metadata.MetricsBuilder) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	val, err := v.read(ctx, v.path)
	if err == nil {
		mb.RecordHardwareTemperatureDataPoint(now, val)
	}

	return err
}

func (v *temperatureSensor) mock(m func(context.Context, string) (float64, error)) {
	v.read = m
}

// reads a humidity sensor from hwmon interface
type humiditySensor struct {
	path string
	read func(ctx context.Context, path string) (float64, error)
}

func newHumiditySensor(chip, name string) *humiditySensor {
	return &humiditySensor{
		path: path.Join(hwmonRoot, chip, name),
		read: readsensor,
	}
}

func (v *humiditySensor) scrape(ctx context.Context, mb *metadata.MetricsBuilder) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	val, err := v.read(ctx, v.path)
	if err == nil {
		mb.RecordHardwareHumidityDataPoint(now, val)
	}

	return err
}

func (v *humiditySensor) mock(m func(context.Context, string) (float64, error)) {
	v.read = m
}

func readsensor(_ context.Context, path string) (float64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseInt(strings.Trim(string(raw), "\n"), 10, 32)
	if err != nil {
		return 0, err
	}

	return float64(val) / 1000, nil
}

// scraper for hardware metrics
type hardwareScraper struct {
	settings scraper.Settings
	config   *Config
	mb       *metadata.MetricsBuilder

	sensors []sensor
}

// newHardwareScraper creates a scraper for hardware metrics
func newHardwareScraper(_ context.Context, settings scraper.Settings, cfg *Config) *hardwareScraper {
	return &hardwareScraper{
		settings: settings,
		config:   cfg,
		sensors: []sensor{
			newTemperatureSensor("hwmon0", "temp1_input"),
			newHumiditySensor("hwmon0", "humidity1_input"),
		},
	}
}

func (s *hardwareScraper) start(_ context.Context, _ component.Host) error {
	s.mb = metadata.NewMetricsBuilder(s.config.MetricsBuilderConfig, s.settings)
	return nil
}

func (s *hardwareScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	for _, sensor := range s.sensors {
		if err := sensor.scrape(ctx, s.mb); err != nil {
			return pmetric.NewMetrics(), err
		}
	}

	return s.mb.Emit(), nil
}
