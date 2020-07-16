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

package exportertest

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
)

// SinkTraceExporterOld acts as a trace receiver for use in tests.
type SinkTraceExporterOld struct {
	consumeTraceError error // to be returned by ConsumeTraceData, if set
	mu                sync.Mutex
	traces            []consumerdata.TraceData
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (ste *SinkTraceExporterOld) Start(context.Context, component.Host) error {
	return nil
}

// ConsumeTraceData stores traces for tests.
func (ste *SinkTraceExporterOld) ConsumeTraceData(_ context.Context, td consumerdata.TraceData) error {
	if ste.consumeTraceError != nil {
		return ste.consumeTraceError
	}

	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.traces = append(ste.traces, td)

	return nil
}

// AllTraces returns the traces sent to the test sink.
func (ste *SinkTraceExporterOld) AllTraces() []consumerdata.TraceData {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	return ste.traces
}

// SetConsumeTraceError sets an error that will be returned by ConsumeTraceData
func (ste *SinkTraceExporterOld) SetConsumeTraceError(err error) {
	ste.consumeTraceError = err
}

// Shutdown stops the exporter and is invoked during shutdown.
func (ste *SinkTraceExporterOld) Shutdown(context.Context) error {
	return nil
}

// SinkTraceExporter acts as a trace receiver for use in tests.
type SinkTraceExporter struct {
	mu                sync.Mutex
	consumeTraceError error // to be returned by ConsumeTraces, if set
	traces            []pdata.Traces
	spansCount        int
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (ste *SinkTraceExporter) Start(context.Context, component.Host) error {
	return nil
}

// ConsumeTraceData stores traces for tests.
func (ste *SinkTraceExporter) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	if ste.consumeTraceError != nil {
		return ste.consumeTraceError
	}

	ste.traces = append(ste.traces, td)
	ste.spansCount += td.SpanCount()

	return nil
}

// AllTraces returns the traces sent to the test sink.
func (ste *SinkTraceExporter) AllTraces() []pdata.Traces {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	copyTraces := make([]pdata.Traces, len(ste.traces))
	copy(copyTraces, ste.traces)
	return copyTraces
}

// SpansCount return the number of spans sent to the test sing.
func (ste *SinkTraceExporter) SpansCount() int {
	ste.mu.Lock()
	defer ste.mu.Unlock()
	return ste.spansCount
}

// SetConsumeTraceError sets an error that will be returned by ConsumeTraces
func (ste *SinkTraceExporter) SetConsumeTraceError(err error) {
	ste.mu.Lock()
	defer ste.mu.Unlock()
	ste.consumeTraceError = err
}

// Shutdown stops the exporter and is invoked during shutdown.
func (ste *SinkTraceExporter) Shutdown(context.Context) error {
	return nil
}

// SinkMetricsExporterOld acts as a metrics receiver for use in tests.
type SinkMetricsExporterOld struct {
	mu                  sync.Mutex
	consumeMetricsError error // to be returned by ConsumeMetricsData, if set
	metrics             []consumerdata.MetricsData
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (sme *SinkMetricsExporterOld) Start(context.Context, component.Host) error {
	return nil
}

// ConsumeMetricsData stores traces for tests.
func (sme *SinkMetricsExporterOld) ConsumeMetricsData(_ context.Context, md consumerdata.MetricsData) error {
	if sme.consumeMetricsError != nil {
		return sme.consumeMetricsError
	}

	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = append(sme.metrics, md)

	return nil
}

// AllMetrics returns the metrics sent to the test sink.
func (sme *SinkMetricsExporterOld) AllMetrics() []consumerdata.MetricsData {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	return sme.metrics
}

// SetConsumeMetricsError sets an error that will be returned by ConsumeMetricsData
func (sme *SinkMetricsExporterOld) SetConsumeMetricsError(err error) {
	sme.consumeMetricsError = err
}

// Shutdown stops the exporter and is invoked during shutdown.
func (sme *SinkMetricsExporterOld) Shutdown(context.Context) error {
	return nil
}

// SinkMetricsExporter acts as a metrics receiver for use in tests.
type SinkMetricsExporter struct {
	mu                  sync.Mutex
	consumeMetricsError error // to be returned by ConsumeMetrics, if set
	metrics             []pdata.Metrics
	metricsCount        int
}

// SetConsumeMetricsError sets an error that will be returned by ConsumeMetrics
func (sme *SinkMetricsExporter) SetConsumeMetricsError(err error) {
	sme.mu.Lock()
	defer sme.mu.Unlock()
	sme.consumeMetricsError = err
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (sme *SinkMetricsExporter) Start(context.Context, component.Host) error {
	return nil
}

// ConsumeMetricsData stores traces for tests.
func (sme *SinkMetricsExporter) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	sme.mu.Lock()
	defer sme.mu.Unlock()
	if sme.consumeMetricsError != nil {
		return sme.consumeMetricsError
	}

	sme.metrics = append(sme.metrics, md)
	sme.metricsCount += pdatautil.MetricsToInternalMetrics(md).MetricCount()

	return nil
}

// AllMetrics returns the metrics sent to the test sink.
func (sme *SinkMetricsExporter) AllMetrics() []pdata.Metrics {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	copyMetrics := make([]pdata.Metrics, len(sme.metrics))
	copy(copyMetrics, sme.metrics)
	return copyMetrics
}

// MetricsCount return the number of metrics sent to the test sing.
func (sme *SinkMetricsExporter) MetricsCount() int {
	sme.mu.Lock()
	defer sme.mu.Unlock()
	return sme.metricsCount
}

// Shutdown stops the exporter and is invoked during shutdown.
func (sme *SinkMetricsExporter) Shutdown(context.Context) error {
	return nil
}

// SinkLogExporter acts as a metrics receiver for use in tests.
type SinkLogExporter struct {
	consumeLogError error // to be returned by ConsumeLog, if set
	mu              sync.Mutex
	logs            []data.Logs
	logRecordsCount int
}

// SetConsumeLogError sets an error that will be returned by ConsumeLog
func (sle *SinkLogExporter) SetConsumeLogError(err error) {
	sle.mu.Lock()
	defer sle.mu.Unlock()
	sle.consumeLogError = err
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (sle *SinkLogExporter) Start(context.Context, component.Host) error {
	return nil
}

// ConsumeLogData stores traces for tests.
func (sle *SinkLogExporter) ConsumeLogs(_ context.Context, ld data.Logs) error {
	sle.mu.Lock()
	defer sle.mu.Unlock()
	if sle.consumeLogError != nil {
		return sle.consumeLogError
	}

	sle.logs = append(sle.logs, ld)
	sle.logRecordsCount += ld.LogRecordCount()

	return nil
}

// AllLog returns the metrics sent to the test sink.
func (sle *SinkLogExporter) AllLogs() []data.Logs {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	copyLogs := make([]data.Logs, len(sle.logs))
	copy(copyLogs, sle.logs)
	return copyLogs
}

// LogRecordsCount return the number of log records sent to the test sing.
func (sle *SinkLogExporter) LogRecordsCount() int {
	sle.mu.Lock()
	defer sle.mu.Unlock()
	return sle.logRecordsCount
}

// Shutdown stops the exporter and is invoked during shutdown.
func (sle *SinkLogExporter) Shutdown(context.Context) error {
	return nil
}
