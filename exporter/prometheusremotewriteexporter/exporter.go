// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Note: implementation for this class is in a separate PR
package prometheusremotewriteexporter

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
)

// PrwExporter converts OTLP metrics to Prometheus remote write TimeSeries and sends them to a remote endpoint
type PrwExporter struct {
	namespace   string
	endpointURL *url.URL
	client      *http.Client
	wg          *sync.WaitGroup
	closeChan   chan struct{}
}

// NewPrwExporter initializes a new PrwExporter instance and sets fields accordingly.
// client parameter cannot be nil.
func NewPrwExporter(namespace string, endpoint string, client *http.Client) (*PrwExporter, error) {

	if client == nil {
		return nil, errors.New("http client cannot be nil")
	}

	endpointURL, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return nil, errors.New("invalid endpoint")
	}

	return &PrwExporter{
		namespace:   namespace,
		endpointURL: endpointURL,
		client:      client,
		wg:          new(sync.WaitGroup),
		closeChan:   make(chan struct{}),
	}, nil
}

// Shutdown stops the exporter from accepting incoming calls(and return error), and wait for current export operations
// to finish before returning
func (prwe *PrwExporter) Shutdown(context.Context) error {
	close(prwe.closeChan)
	prwe.wg.Wait()
	return nil
}

// PushMetrics converts metrics to Prometheus remote write TimeSeries and send to remote endpoint. It maintain a map of
// TimeSeries, validates and handles each individual metric, adding the converted TimeSeries to the map, and finally
// exports the map.
func (prwe *PrwExporter) PushMetrics(ctx context.Context, md pdata.Metrics) (int, error) {
	prwe.wg.Add(1)
	defer prwe.wg.Done()
	select {
	case <-prwe.closeChan:
		return pdatautil.MetricCount(md), errors.New("shutdown has been called")
	default:
		tsMap := map[string]*prompb.TimeSeries{}
		dropped := 0
		errs := []error{}

		resourceMetrics := data.MetricDataToOtlp(pdatautil.MetricsToInternalMetrics(md))
		for _, resourceMetric := range resourceMetrics {
			if resourceMetric == nil {
				continue
			}
			// TODO: add resource attributes as labels, probably in next PR
			for _, instrumentationMetrics := range resourceMetric.InstrumentationLibraryMetrics {
				if instrumentationMetrics == nil {
					continue
				}
				// TODO: decide if instrumentation library information should be exported as labels
				for _, metric := range instrumentationMetrics.Metrics {
					if metric == nil {
						dropped++
						continue
					}
					// check for valid type and temporality combination and for matching data field and type
					if ok := validateMetrics(metric); !ok {
						dropped++
						errs = append(errs, errors.New("invalid temporality and type combination"))
						continue
					}
					// handle individual metric based on type
					switch metric.Data.(type) {
					case *otlp.Metric_DoubleSum, *otlp.Metric_IntSum, *otlp.Metric_DoubleGauge, *otlp.Metric_IntGauge:
						if err := prwe.handleScalarMetric(tsMap, metric); err != nil {
							dropped++
							errs = append(errs, err)
						}
					case *otlp.Metric_DoubleHistogram, *otlp.Metric_IntHistogram:
						if err := prwe.handleHistogramMetric(tsMap, metric); err != nil {
							dropped++
							errs = append(errs, err)
						}
					default:
						dropped++
						errs = append(errs, errors.New("unsupported metric type"))
					}
				}
			}
		}

		if err := prwe.export(ctx, tsMap); err != nil {
			dropped = pdatautil.MetricCount(md)
			errs = append(errs, err)
		}

		if dropped != 0 {
			return dropped, componenterror.CombineErrors(errs)
		}

		return 0, nil
	}
}

// handleScalarMetric processes data points in a single OTLP scalar metric by adding the each point as a Sample into
// its corresponding TimeSeries in tsMap.
// tsMap and metric cannot be nil, and metric must have a non-nil descriptor
func (prwe *PrwExporter) handleScalarMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {

	switch metric.Data.(type) {
	// int points
	case *otlp.Metric_DoubleGauge:
		if metric.GetDoubleGauge().GetDataPoints() == nil {
			return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
		}
		for _, pt := range metric.GetDoubleGauge().GetDataPoints() {
			addSingleDoubleDataPoint(pt, metric, prwe.namespace, tsMap)
		}
	case *otlp.Metric_IntGauge:
		if metric.GetIntGauge().GetDataPoints() == nil {
			return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
		}
		for _, pt := range metric.GetIntGauge().GetDataPoints() {
			addSingleIntDataPoint(pt, metric, prwe.namespace, tsMap)
		}
	case *otlp.Metric_DoubleSum:
		if metric.GetDoubleSum().GetDataPoints() == nil {
			return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
		}
		for _, pt := range metric.GetDoubleSum().GetDataPoints() {
			addSingleDoubleDataPoint(pt, metric, prwe.namespace, tsMap)
		}
	case *otlp.Metric_IntSum:
		if metric.GetIntSum().GetDataPoints() == nil {
			return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
		}
		for _, pt := range metric.GetIntSum().GetDataPoints() {
			addSingleIntDataPoint(pt, metric, prwe.namespace, tsMap)
		}
	}
	return nil
}

// handleHistogramMetric processes data points in a single OTLP histogram metric by mapping the sum, count and each
// bucket of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (prwe *PrwExporter) handleHistogramMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {

	switch metric.Data.(type) {
	case *otlp.Metric_IntHistogram:
		if metric.GetIntHistogram().GetDataPoints() == nil {
			return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
		}
		for _, pt := range metric.GetIntHistogram().GetDataPoints() {
			addSingleIntHistogramDataPoint(pt, metric, prwe.namespace, tsMap)
		}
	case *otlp.Metric_DoubleHistogram:
		if metric.GetDoubleHistogram().GetDataPoints() == nil {
			return fmt.Errorf("nil data point. %s is dropped", metric.GetName())
		}
		for _, pt := range metric.GetDoubleHistogram().GetDataPoints() {
			addSingleDoubleHistogramDataPoint(pt, metric, prwe.namespace, tsMap)
		}
	}
	return nil
}

// export sends a Snappy-compressed WriteRequest containing TimeSeries to a remote write endpoint in order
func (prwe *PrwExporter) export(ctx context.Context, tsMap map[string]*prompb.TimeSeries) error {
	//Calls the helper function to convert the TsMap to the desired format
	req, err := wrapTimeSeries(tsMap)
	if err != nil {
		return err
	}

	//Uses proto.Marshal to convert the WriteRequest into bytes array
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	buf := make([]byte, len(data), cap(data))
	compressedData := snappy.Encode(buf, data)

	//Create the HTTP POST request to send to the endpoint
	httpReq, err := http.NewRequest("POST", prwe.endpointURL.String(), bytes.NewReader(compressedData))
	if err != nil {
		return err
	}

	// Add necessary headers specified by:
	// https://cortexmetrics.io/docs/apis/#remote-api
	httpReq.Header.Add("Content-Encoding", "snappy")
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	httpReq = httpReq.WithContext(ctx)

	_, cancel := context.WithTimeout(context.Background(), prwe.client.Timeout)
	defer cancel()

	httpResp, err := prwe.client.Do(httpReq)
	if err != nil {
		return err
	}

	if httpResp.StatusCode/100 != 2 {
		scanner := bufio.NewScanner(io.LimitReader(httpResp.Body, 256))
		line := ""
		if scanner.Scan() {
			line = scanner.Text()
		}
		errMsg := "server returned HTTP status " + httpResp.Status + ": " + line
		return errors.New(errMsg)
	}
	return nil
}
