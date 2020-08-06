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

package sematextexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configerror"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()

	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	_, err := factory.CreateMetricsExporter(context.Background(), params, cfg) // TODO pass logger in params
	assert.Error(t, err, configerror.ErrDataTypeIsNotSupported)
}

func TestCreateLogsExporter(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()

	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	_, err := factory.CreateMetricsExporter(context.Background(), params, cfg)
	assert.Error(t, err, configerror.ErrDataTypeIsNotSupported)
}

func TestCreateMetricsInstanceViaFactory(t *testing.T) {
	factory := Factory{}

	cfg := factory.CreateDefaultConfig()

	// Default config doesn't have default endpoint so creating from it should fail.
	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateMetricsExporter(context.Background(), params, cfg)

	assert.NotNil(t, err)
	assert.Equal(t, "\"sematext\" config requires a non-empty \"endpoint\"", err.Error()) // TODO adjust test
	assert.Nil(t, exp)

	// Endpoint doesn't have a default value so set it directly.
	expCfg := cfg.(*Config)
	expCfg.MetricsConduitSettings.Endpoint = "some.target.org:12345"
	exp, err = factory.CreateMetricsExporter(context.Background(), params, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, exp)

	// TODO - Other tests

	assert.NoError(t, exp.Shutdown(context.Background()))
}

func TestCreateLogsInstanceViaFactory(t *testing.T) {
	factory := Factory{}

	cfg := factory.CreateDefaultConfig()

	// Default config doesn't have default endpoint so creating from it should fail.
	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	exp, err := factory.CreateLogsExporter(context.Background(), params, cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "\"sematext\" config requires a non-empty \"endpoint\"", err.Error()) // TODO adjust test
	assert.Nil(t, exp)

	// Endpoint doesn't have a default value so set it directly.
	expCfg := cfg.(*Config)
	expCfg.LogsConduitSettings.Endpoint = "some.target.org:12345"
	exp, err = factory.CreateLogsExporter(context.Background(), params, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, exp)

	// TODO - Other tests

	assert.NoError(t, exp.Shutdown(context.Background()))
}
