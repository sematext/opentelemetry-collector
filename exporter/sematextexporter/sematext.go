// Copyright 2019, OpenTelemetry Authors
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

	influxdb2 "github.com/influxdata/influxdb-client-go"
	jaegerproto "github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	jaegertranslator "go.opentelemetry.io/collector/translator/trace/jaeger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

/**
-- TODO
- logs : pavel or rafal : omit
- traces : nedim traces : jaeger
- metrics : boyan metrics ingesttion infra. influx line protocol influxdb
https://github.com/influxdata/influxdb-client-go

*/

// protoGRPCSender forwards spans encoded in the jaeger proto
// format, to a grpc server.
type protoGRPCSender struct {
	client       jaegerproto.CollectorServiceClient
	metadata     metadata.MD
	waitForReady bool
}

//NewTraceExporter todo doc comment
// The exporter name is the name to be used in the observability of the exporter.
// The collectorEndpoint should be of the form "hostname:14250" (a gRPC target).
func NewTraceExporter(config *Config) (component.TraceExporter, error) {

	opts, err := configgrpc.GrpcSettingsToDialOptions(config.GRPCSettings)
	if err != nil {
		return nil, err
	}

	client, err := grpc.Dial(config.GRPCSettings.Endpoint, opts...)
	if err != nil {
		return nil, err
	}

	collectorServiceClient := jaegerproto.NewCollectorServiceClient(client)

	s := &protoGRPCSender{
		client:       collectorServiceClient,
		metadata:     metadata.New(config.GRPCSettings.Headers),
		waitForReady: config.WaitForReady,
	}

	exp, err := exporterhelper.NewTraceExporter(config, s.pushTraceData)

	return exp, err

}

func (s *protoGRPCSender) pushTraceData(ctx context.Context, td pdata.Traces) (droppedSpans int, err error) {

	batches, err := jaegertranslator.InternalTracesToJaegerProto(td)
	if err != nil {
		return td.SpanCount(), consumererror.Permanent(err)
	}

	if s.metadata.Len() > 0 {
		ctx = metadata.NewOutgoingContext(ctx, s.metadata)
	}

	var sentSpans int
	for _, batch := range batches {
		_, err = s.client.PostSpans(ctx, &jaegerproto.PostSpansRequest{Batch: *batch}, grpc.WaitForReady(s.waitForReady))
		if err != nil {
			return td.SpanCount() - sentSpans, err
		}
		sentSpans += len(batch.Spans)
	}

	return 0, nil
}


type sematextExporter struct {
	exporter *influxdb2.Client // TODO explore making this multiple exporters under a channel
}

// NewMetricsExporter todo doc comment
func NewMetricsExporter(config *Config) (component.MetricsExporter, error) {

	opts := influxdb2.DefaultOptions()
	opts.SetBatchSize(config.batchsize)
	opts.SetUseGZip(true)
	opts.SetTlsConfig(&tls.Config{
		InsecureSkipVerify: true,
	})

	client := influxdb2.NewClientWithOptions(config.endpoint, config.token, opts)
	writeApi := client.WriteApi(config.org, config.bucket)

	se := &sematextExporter{
		exporter: writeApi,
	}

	exp, err := exporterhelper.NewMetricsExporter(config, se.pushMetricsData, exporterhelper.WithShutdown(se.Shutdown))

	return exp, err
}

// PushMetricsData sends Metrics to Influx DB - TODO Doc Comment
func (se *sematextExporter) PushMetricsData(ctx context.Context, md consumerdata.MetricsData) (int, error) {


	/*
	TODO
	md :
		Node     *commonpb.Node
		Resource *resourcepb.Resource
		Metrics  []*metricspb.Metric
	*/


	for i, metric := range md.Metrics {

		p := influxdb2.NewPoint( // TODO
            "system",
            map[string]string{ // TODO integrate tags as per sematext influxdb endpoint.
                "id":       fmt.Sprintf("rack_%v", i%10),
                "vendor":   "AWS",
                "hostname": fmt.Sprintf("host_%v", i%100),
            },
            map[string]interface{}{ // TODO integrate fields as per sematext influxdb endpoint.
                "temperature": rand.Float64() * 80.0,
                "disk_free":   rand.Float64() * 1000.0,
                "disk_total":  (i/10 + 1) * 1000000,
                "mem_total":   (i/100 + 1) * 10000000,
                "mem_free":    rand.Uint64(),
            },
			time.Now() // TODO integrate fields as per sematext influxdb endpoint.
		)
        writeApi.WritePoint(p)
	}
	writeApi.Flush()
	return 0, nil
}


// Shutdown stops the exporter and is invoked during shutdown.
func (se *sematextExporter) Shutdown(context.Context) error {
	se.exporter.Close()
	return nil
}
