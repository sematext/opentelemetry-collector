// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/**
-- TODO

- traces : nedim traces : jaeger

- metrics : boyan metrics ingesttion infra. influx line protocol influxdb

- logs : pavel or rafal : https://sematext.com/docs/logs/index-events-via-elasticsearch-api/

https://github.com/influxdata/influxdb-client-go

//----------------------------------------------------------------------------------------------------------------------

*/

package sematextexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
)

const (
	// The value of "type" key in configuration.
	typeStr = "sematext"
)

// Factory is the factory for Sematext exporter.
type Factory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for exporter.
func (f *Factory) CreateDefaultConfig() configmodels.Exporter {

	config := &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		MetricsConduitSettings: MetricsConfig{}, //TODO only if present in yaml
		LogsConduitSettings:    LogsConfig{},    //TODO only if present in yaml
	}

	config.MetricsConduitSettings.MetricsConfigSetDefaults()
	config.LogsConduitSettings.LogsConfigSetDefaults()

	return config
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *Factory) CreateMetricsExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	config configmodels.Exporter,
) (component.MetricsExporter, error) {

	cfg := config.(*Config)

	// TODO cfg checks

	exp, err := NewMetricsExporter(cfg)
	if err != nil {
		return nil, err
	}

	return exp, nil
}

// CreateLogsExporter creates a metrics exporter based on this config.
func (f *Factory) CreateLogsExporter(
	ctx context.Context,
	params component.ExporterCreateParams,
	config configmodels.Exporter,
) (component.LogExporter, error) {

	cfg := config.(*Config)

	// TODO cfg checks

	exp, err := NewLogsExporter(cfg)
	if err != nil {
		return nil, err
	}

	return exp, nil
}
