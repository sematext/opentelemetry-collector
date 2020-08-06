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
	"net/http"
	"runtime"
	"time"

	"github.com/alecthomas/units"
	"go.opentelemetry.io/collector/config/configmodels"
)

// LogsConfig defines configuration for Sematext log exporter.
type LogsConfig struct {
	configmodels.ExporterSettings `mapstructure:",squash"`
	Conduit                       LogsConduit
	Token                         string
	Endpoint                      string
	NumWorkers                    int
	FlushBytes                    int
	FlushInterval                 time.Duration
	Index                         string
	Header                        http.Header
	Human                         bool
	Pipeline                      string
	Pretty                        bool
	Refresh                       string
	Source                        []string
	SourceExcludes                []string
	SourceIncludes                []string
	Timeout                       time.Duration
}

// LogsConfigSetDefaults contains default settings used in the factory.
func (config *LogsConfig) LogsConfigSetDefaults() {
	config.Token = ""    // TODO pull out of environment?
	config.Endpoint = "" // TODO make this more friendly
	config.NumWorkers = int(runtime.NumCPU())
	config.FlushBytes = int(5 * units.Mebibyte)
	config.FlushInterval = 500 * time.Millisecond
	config.Index = config.Token
	config.Header = http.Header{}
	config.Human = false // TODO consider this
	config.Pretty = false
	config.Refresh = "false"
	config.Timeout = 20 * time.Second
}
