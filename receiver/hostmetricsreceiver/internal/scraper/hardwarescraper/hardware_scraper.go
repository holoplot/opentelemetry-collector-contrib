// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hardwarescraper // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/hardwarescraper"

import (
	"context"
	"os"
	"path"
	"regexp"
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

var hwmonFilenameFormat = regexp.MustCompile(`^(?P<type>[^0-9]+)(?P<id>[0-9]*)?(_(?P<property>.+))?$`)

// explodeSensorFilename splits a sensor name into <type><num>_<property>.
func explodeSensorFilename(filename string) (ok bool, sensorType string, sensorNum int, sensorProperty string) {
	matches := hwmonFilenameFormat.FindStringSubmatch(filename)
	if len(matches) == 0 {
		return false, sensorType, sensorNum, sensorProperty
	}
	for i, match := range hwmonFilenameFormat.SubexpNames() {
		if i >= len(matches) {
			return true, sensorType, sensorNum, sensorProperty
		}
		if match == "type" {
			sensorType = matches[i]
		}
		if match == "property" {
			sensorProperty = matches[i]
		}
		if match == "id" && len(matches[i]) > 0 {
			num, err := strconv.Atoi(matches[i])
			if err != nil {
				return false, sensorType, sensorNum, sensorProperty
			}

			sensorNum = num
		}
	}
	return true, sensorType, sensorNum, sensorProperty
}

type chip struct {
	device   string
	resource pcommon.Resource
	sensors  []*sensor
}

func newChip(rb *metadata.ResourceBuilder, device string) (*chip, error) {
	raw, err := os.ReadFile(path.Join(hwmonRoot, device, "name"))
	if err != nil {
		return nil, err
	}

	rb.SetHardwareChipName(strings.Trim(string(raw), "\n"))

	c := &chip{device: device, resource: rb.Emit()}

	entries, _ := os.ReadDir(path.Join(hwmonRoot, device))
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			if s := newSensor(c, entry.Name()); s != nil {
				c.sensors = append(c.sensors, s)
			}
		}
	}

	return c, nil
}

func (c *chip) scrape(ctx context.Context, mb *metadata.MetricsBuilder, ts pcommon.Timestamp) error {
	for _, s := range c.sensors {
		if err := s.scrape(ctx, mb, ts); err != nil {
			return err
		}
	}

	mb.EmitForResource(metadata.WithResource(c.resource))

	return nil
}

type sensor struct {
	path string
	kind string
}

func newSensor(chip *chip, name string) *sensor {
	// TODO: deal with multiple sensors of the same kind
	if ok, kind, _, prop := explodeSensorFilename(name); ok {
		switch kind {
		case "temp", "humidity":
			if prop == "input" {
				return &sensor{
					path: path.Join(hwmonRoot, chip.device, name),
					kind: kind,
				}
			}
		}
	}

	return nil
}

func (s *sensor) scrape(ctx context.Context, mb *metadata.MetricsBuilder, ts pcommon.Timestamp) error {
	val, err := s.read(ctx)
	if err == nil {
		switch s.kind {
		case "temp":
			mb.RecordHardwareTemperatureDataPoint(ts, val)
		case "humidity":
			mb.RecordHardwareHumidityDataPoint(ts, val)
		}
	}

	return err
}

func (s *sensor) read(_ context.Context) (float64, error) {
	raw, err := os.ReadFile(s.path)
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
	chips    []*chip
}

// newHardwareScraper creates a scraper for hardware metrics
func newHardwareScraper(_ context.Context, settings scraper.Settings, cfg *Config) *hardwareScraper {
	return &hardwareScraper{
		settings: settings,
		config:   cfg,
	}
}

func (s *hardwareScraper) start(_ context.Context, _ component.Host) error {
	s.mb = metadata.NewMetricsBuilder(s.config.MetricsBuilderConfig, s.settings)

	rb := s.mb.NewResourceBuilder()

	entries, _ := os.ReadDir(hwmonRoot)
	for _, entry := range entries {
		if entry.IsDir() {
			if c, err := newChip(rb, entry.Name()); err == nil {
				s.chips = append(s.chips, c)
			}
		}
	}

	return nil
}

func (s *hardwareScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	now := pcommon.NewTimestampFromTime(time.Now())

	for _, c := range s.chips {
		if err := c.scrape(ctx, s.mb, now); err != nil {
			return pmetric.NewMetrics(), err
		}
	}

	return s.mb.Emit(), nil
}
