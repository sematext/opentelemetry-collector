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

package jaegerexporter

import (
	"context"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtls"
)

func TestLoadConfig(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.NoError(t, err)

	factory := &Factory{}
	factories.Exporters[typeStr] = factory
	cfg, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	e0 := cfg.Exporters["jaeger"]

	// Endpoint doesn't have a default value so set it directly.
	defaultCfg := factory.CreateDefaultConfig().(*Config)
	defaultCfg.Endpoint = "some.target:55678"
	defaultCfg.GRPCClientSettings.Endpoint = defaultCfg.Endpoint
	defaultCfg.GRPCClientSettings.TLSSetting = configtls.TLSClientSetting{
		Insecure: true,
	}
	assert.Equal(t, defaultCfg, e0)

	e1 := cfg.Exporters["jaeger/2"]
	assert.Equal(t, "jaeger/2", e1.(*Config).Name())
	assert.Equal(t, "a.new.target:1234", e1.(*Config).Endpoint)
	assert.Equal(t, "round_robin", e1.(*Config).GRPCClientSettings.BalancerName)
	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	te, err := factory.CreateTraceExporter(context.Background(), params, e1)
	require.NoError(t, err)
	require.NotNil(t, te)
}
