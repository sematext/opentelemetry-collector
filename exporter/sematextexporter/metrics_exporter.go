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
	"crypto/tls"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go"
	"github.com/influxdata/influxdb-client-go/api"
	"github.com/influxdata/influxdb-client-go/api/write"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type MetricsEndpoint string

const (
	USMetrics MetricsEndpoint = "https://spm-receiver.sematext.com:443/write?db=metrics&v=3.0.0&sct=APP"
	EUMetrics MetricsEndpoint = "https://spm-receiver.eu.sematext.com:443/write?db=metrics&v=3.0.0&sct=APP"
)

// MetricsConduit todo doc comment
type MetricsConduit struct {
	config   *MetricsConfig
	influxdb struct {
		config *influxdb2.Options
		pool   influxdb2.Client
		bulk   api.WriteApi // TODO Consider making this multiple exporters under a channel
	}
}

// NewMetricsExporter todo doc comment
func NewMetricsExporter(config *Config) (component.MetricsExporter, error) {

	var endpoint string
	config.MetricsConduitSettings.Conduit = MetricsConduit{}
	conduit := config.MetricsConduitSettings.Conduit

	switch strings.ToLower(conduit.config.Endpoint) {
	case "usmetrics":
		endpoint = string(USMetrics)
	case "eumetrics":
		endpoint = string(EUMetrics)
	default:
		endpoint = string(USMetrics)
	}

	conduit.influxdb.config = influxdb2.DefaultOptions()
	conduit.influxdb.config.SetBatchSize(conduit.config.BatchSize)
	conduit.influxdb.config.SetFlushInterval(conduit.config.FlushInterval)
	conduit.influxdb.config.SetMaxRetries(conduit.config.MaxRetries)
	conduit.influxdb.config.SetHttpRequestTimeout(conduit.config.HttpRequestTimeout)
	conduit.influxdb.config.SetRetryBufferLimit(conduit.config.RetryBufferLimit)
	conduit.influxdb.config.SetUseGZip(true)
	conduit.influxdb.config.SetTlsConfig(&tls.Config{InsecureSkipVerify: true})

	conduit.influxdb.pool = influxdb2.NewClientWithOptions(endpoint, conduit.config.Token, conduit.influxdb.config)
	conduit.influxdb.bulk = conduit.influxdb.pool.WriteApi(conduit.config.Org, conduit.config.Bucket)

	return exporterhelper.NewMetricsExporter(
		config,
		conduit.PushMetricsData,
		exporterhelper.WithShutdown(conduit.Shutdown),
	)

}

// PushMetricsData sends Metrics to Influx DB - TODO Doc Comment
func (conduit *MetricsConduit) PushMetricsData(ctx context.Context, source pdata.Metrics) (droppedTimeSeries int, err error) {

	internalMetricData := pdatautil.MetricsToInternalMetrics(source)
	resourceMetrics := internalMetricData.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {

		resourceMetric := resourceMetrics.At(i)
		if resourceMetric.IsNil() {
			continue // TODO Consider logging this
		}

		instrumentationLibraryMetrics := resourceMetric.InstrumentationLibraryMetrics()
		for j := 0; j < instrumentationLibraryMetrics.Len(); j++ {

			instrumentationLibraryMetric := instrumentationLibraryMetrics.At(j)
			if instrumentationLibraryMetric.IsNil() {
				continue // TODO Consider logging this
			}

			metrics := instrumentationLibraryMetric.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				if metric.IsNil() {
					continue // TODO Consider logging this
				}

				t := translator{}
				t.setToken(conduit.config.Token)
				points := t.translate(metric)

				for _, point := range points {
					conduit.influxdb.bulk.WritePoint(&point)
				}

			}
		}
	}

	conduit.influxdb.bulk.Flush()
	return 0, nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (conduit *MetricsConduit) Shutdown(ctx context.Context) error {
	defer conduit.influxdb.pool.Close()
	return nil
}

type translator struct {
	token       string
	name        string
	description string
	unit        string
	acc         []write.Point
}

func (t *translator) setToken(token string) {
	t.token = token
}

func (t *translator) translate(metric pdata.Metric) []write.Point {

	descriptor := metric.MetricDescriptor()
	t.name = descriptor.Name()

	// TODO - this is metadata - different endpoint on SC cloud/
	t.description = descriptor.Description()
	t.unit = descriptor.Unit() // http://unitsofmeasure.org/ucum.html.

	switch descriptor.Type() {
	case pdata.MetricTypeInvalid:
		return nil
	case pdata.MetricTypeInt64:
		t.absorbInt64DataPoints(metric.Int64DataPoints())
	case pdata.MetricTypeDouble:
		t.absorbDoubleDataPoints(metric.DoubleDataPoints())
	case pdata.MetricTypeMonotonicInt64:
		t.absorbInt64DataPoints(metric.Int64DataPoints())
	case pdata.MetricTypeMonotonicDouble:
		t.absorbDoubleDataPoints(metric.DoubleDataPoints())
	case pdata.MetricTypeHistogram:
		t.absorbHistogramDataPoint(metric.HistogramDataPoints())
	case pdata.MetricTypeSummary:
		t.absorbSummaryDataPoint(metric.SummaryDataPoints())
	}

	return t.acc

}

func (t *translator) absorbInt64DataPoints(rows pdata.Int64DataPointSlice) {
	for i := 0; i < rows.Len(); i++ {
		row := rows.At(i)
		p := write.NewPointWithMeasurement(t.name)
		p.AddTag("token", t.token)
		//p.AddTag("os.host", ???) // TODO

		row.LabelsMap().ForEach(func(k string, v pdata.StringValue) {
			p.AddField(k, v)
		})
		p.AddField(t.name, row.Value)
		p.SetTime(time.Unix(0, int64(row.Timestamp())))
		p.SortTags()
		p.SortFields()
		t.acc = append(t.acc, *p)
	}
}

func (t *translator) absorbDoubleDataPoints(rows pdata.DoubleDataPointSlice) {
	for i := 0; i < rows.Len(); i++ {
		row := rows.At(i)
		p := write.NewPointWithMeasurement(t.name)
		p.AddTag("token", t.token)
		//p.AddTag("os.host", ???) // TODO

		row.LabelsMap().ForEach(func(k string, v pdata.StringValue) {
			p.AddField(k, v)
		})
		p.AddField(t.name, row.Value)
		p.SetTime(time.Unix(0, int64(row.Timestamp())))
		p.SortTags()
		p.SortFields()
		t.acc = append(t.acc, *p)
	}
}

func (t *translator) absorbHistogramDataPoint(rows pdata.HistogramDataPointSlice) {

	// TODO - Can SC handle this ?

	//Labels []*v11.StringKeyValue `protobuf:"bytes,1,rep,name=labels,proto3" json:"labels,omitempty"`
	//StartTimeUnixNano uint64 `protobuf:"fixed64,2,opt,name=start_time_unix_nano,json=startTimeUnixNano,proto3" json:"start_time_unix_nano,omitempty"`
	//TimeUnixNano uint64 `protobuf:"fixed64,3,opt,name=time_unix_nano,json=timeUnixNano,proto3" json:"time_unix_nano,omitempty"`
	//Count uint64 `protobuf:"varint,4,opt,name=count,proto3" json:"count,omitempty"`
	//Sum float64 `protobuf:"fixed64,5,opt,name=sum,proto3" json:"sum,omitempty"`
	//Buckets []*HistogramDataPoint_Bucket `protobuf:"bytes,6,rep,name=buckets,proto3" json:"buckets,omitempty"`
	//ExplicitBounds []float64 `protobuf:"fixed64,7,rep,packed,name=explicit_bounds,json=explicitBounds,proto3" json:"explicit_bounds,omitempty"`

	for i := 0; i < rows.Len(); i++ {
		row := rows.At(i)
		p := write.NewPointWithMeasurement(t.name)
		p.AddTag("token", t.token)
		//p.AddTag("os.host", ???) // TODO

		row.LabelsMap().ForEach(func(k string, v pdata.StringValue) {
			p.AddField(k, v)
		})
		p.AddField(t.name, row.Count)
		p.SetTime(time.Unix(0, int64(row.Timestamp())))
		p.SortTags()
		p.SortFields()
		t.acc = append(t.acc, *p)
	}
}

func (t *translator) absorbSummaryDataPoint(rows pdata.SummaryDataPointSlice) {

	// TODO - Can SC handle this ?

	//Labels []*v11.StringKeyValue `protobuf:"bytes,1,rep,name=labels,proto3" json:"labels,omitempty"`
	//StartTimeUnixNano uint64 `protobuf:"fixed64,2,opt,name=start_time_unix_nano,json=startTimeUnixNano,proto3" json:"start_time_unix_nano,omitempty"`
	//TimeUnixNano uint64 `protobuf:"fixed64,3,opt,name=time_unix_nano,json=timeUnixNano,proto3" json:"time_unix_nano,omitempty"`
	//Count uint64 `protobuf:"varint,4,opt,name=count,proto3" json:"count,omitempty"`
	//Sum float64 `protobuf:"fixed64,5,opt,name=sum,proto3" json:"sum,omitempty"`
	//PercentileValues []*SummaryDataPoint_ValueAtPercentile `protobuf:"bytes,6,rep,na

	for i := 0; i < rows.Len(); i++ {
		row := rows.At(i)
		p := write.NewPointWithMeasurement(t.name)
		p.AddTag("token", t.token)
		//p.AddTag("os.host", ???) // TODO

		row.LabelsMap().ForEach(func(k string, v pdata.StringValue) {
			p.AddField(k, v)
		})
		p.AddField(t.name, row.Sum) // TODO check
		p.SetTime(time.Unix(0, int64(row.Timestamp())))
		p.SortTags()
		p.SortFields()
		t.acc = append(t.acc, *p)
	}
}
