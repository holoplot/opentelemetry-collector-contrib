// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hardwarescraper // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/hardwarescraper"

import (
	"context"
	"fmt"
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

var hwmonFilenameFormat = regexp.MustCompile(`^(?P<type>\D+)(?P<id>\d*)?(_(?P<property>.+))?$`)

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
		if match == "id" && matches[i] != "" {
			num, err := strconv.Atoi(matches[i])
			if err != nil {
				return false, sensorType, sensorNum, sensorProperty
			}

			sensorNum = num
		}
	}
	return true, sensorType, sensorNum, sensorProperty
}

func ReadFile(filename string) (string, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return strings.Trim(string(bytes), "\n"), nil
}

type chip struct {
	device   string
	name     string
	resource pcommon.Resource
	sensors  []*sensor
}

func newChip(rb *metadata.ResourceBuilder, rootDir, device string) (*chip, error) {
	name, err := ReadFile(path.Join(rootDir, device, "name"))
	if err != nil {
		return nil, err
	}

	rb.SetName(name)

	c := &chip{device: device, name: name, resource: rb.Emit()}

	entries, _ := os.ReadDir(path.Join(rootDir, device))
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			if s := newSensor(c, rootDir, entry.Name()); s != nil {
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
	path  string
	kind  string
	id    string
	label string
}

func newSensor(chip *chip, rootDir, name string) *sensor {
	var s *sensor

	if ok, kind, num, prop := explodeSensorFilename(name); ok {
		switch kind {
		case "fan", "humidity", "temp":
			if prop == "input" {
				s = &sensor{
					path: path.Join(rootDir, chip.device, name),
					kind: kind,
					id:   fmt.Sprintf("%s_%s%d", chip.device, kind, num),
				}
			}
		}

		if s != nil {
			filename := path.Join(rootDir, chip.device, kind+"_label")
			if label, err := ReadFile(filename); err != nil {
				s.label = label
			} else {
				s.label = chip.name
			}
		}
	}

	return s
}

func (s *sensor) scrape(ctx context.Context, mb *metadata.MetricsBuilder, ts pcommon.Timestamp) error {
	fromMilli := func(v int64) float64 {
		return 0.001 * float64(v)
	}

	val, err := s.readValue(ctx)
	if err == nil {
		switch s.kind {
		case "fan":
			mb.RecordHardwareFanSpeedDataPoint(ts, val, s.id, s.label)
		case "humidity":
			mb.RecordHardwareHumidityDataPoint(ts, fromMilli(val), s.id, s.label)
		case "temp":
			mb.RecordHardwareTemperatureDataPoint(ts, fromMilli(val), s.id, s.label)
		}
	}

	return err
}

func (s *sensor) readValue(_ context.Context) (int64, error) {
	str, err := ReadFile(s.path)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return 0, err
	}

	return val, nil
}

// scraper for hardware metrics
type hardwareScraper struct {
	settings scraper.Settings
	config   *Config
	mb       *metadata.MetricsBuilder
	rootDir  string
	chips    []*chip
}

// newHardwareScraper creates a scraper for hardware metrics
func newHardwareScraper(_ context.Context, settings scraper.Settings, cfg *Config) *hardwareScraper {
	return &hardwareScraper{
		settings: settings,
		config:   cfg,
		rootDir:  "/sys/class/hwmon",
	}
}

func (s *hardwareScraper) start(_ context.Context, _ component.Host) error {
	s.mb = metadata.NewMetricsBuilder(s.config.MetricsBuilderConfig, s.settings)

	rb := s.mb.NewResourceBuilder()

	entries, _ := os.ReadDir(s.rootDir)

	for _, entry := range entries {
		path := path.Join(s.rootDir, entry.Name())

		fileInfo, err := os.Lstat(path)
		if err != nil {
			continue
		}

		if fileInfo.Mode()&os.ModeSymlink > 0 {
			fileInfo, err = os.Stat(path)
			if err != nil {
				continue
			}
		}

		if !fileInfo.IsDir() {
			continue
		}

		if c, err := newChip(rb, s.rootDir, entry.Name()); err == nil {
			s.chips = append(s.chips, c)
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
