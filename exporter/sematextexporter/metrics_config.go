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

// MetricsConfig defines configuration for Sematext metrics exporter.
type MetricsConfig struct {
	Conduit            MetricsConduit
	Token              string
	Org                string
	Bucket             string
	Endpoint           string
	BatchSize          uint
	FlushInterval      uint
	MaxRetries         uint
	HttpRequestTimeout uint
	RetryBufferLimit   uint
}

// MetricsConfigSetDefaults contains default settings used in the factory.
func (mc *MetricsConfig) MetricsConfigSetDefaults() {
	mc.Org = "Example Org"
	mc.Bucket = "Example Bucket"
	mc.Endpoint = "USMetrics"
	mc.BatchSize = 5000
	mc.FlushInterval = 5 * 1000
	mc.MaxRetries = 5
	mc.HttpRequestTimeout = 30 * 1000
	mc.RetryBufferLimit = mc.BatchSize * mc.MaxRetries
}

// TODO role/source identification
// TODO metric filtering
