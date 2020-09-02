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
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff"

	elasticsearch "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/internal/data"
	v1 "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"
)

// https://github.com/elastic/go-elasticsearch/blob/master/esutil/bulk_indexer.go
// https://github.com/elastic/go-elasticsearch/tree/master/_examples/bulk#indexergo

type endpointList struct {
	usa    []string
	europe []string
}

var endpoints endpointList = endpointList{
	usa: []string{
		"logsene-receiver.sematext.com:443",
	},
	europe: []string{
		"logsene-receiver.eu.sematext.com:443",
	},
}

// LogsConduit TODO doc comment
type LogsConduit struct {
	config        *Config
	elasticsearch struct {
		config elasticsearch.Config
		pool   *elasticsearch.Client
	}
	indexer struct {
		config esutil.BulkIndexerConfig
		bulk   esutil.BulkIndexer
	}
	backoff *backoff.ExponentialBackOff
}

// NewLogsExporter creates a new exporter, under config formed by the factory
func NewLogsExporter(config *Config) (component.LogExporter, error) {

	var err error
	var addresses []string

	config.LogsConduitSettings.Conduit = LogsConduit{}
	conduit := config.LogsConduitSettings.Conduit

	switch strings.ToLower(conduit.config.LogsConduitSettings.Endpoint) {
	case "uslogs":
		addresses = endpoints.usa
	case "eulogs":
		addresses = endpoints.europe
	default:
		addresses = endpoints.usa
	}

	conduit.config = config
	conduit.backoff = backoff.NewExponentialBackOff()

	conduit.elasticsearch.config = elasticsearch.Config{
		Addresses:     addresses,
		RetryOnStatus: []int{502, 503, 504, 429},
		RetryBackoff:  config.LogsConduitSettings.Conduit.retryStrategy,
		MaxRetries:    5,
	}

	conduit.elasticsearch.pool, err = elasticsearch.NewClient(conduit.elasticsearch.config)
	if err != nil {
		return nil, err
	}

	conduit.indexer.config = esutil.BulkIndexerConfig{
		Client:        conduit.elasticsearch.pool,
		NumWorkers:    config.LogsConduitSettings.NumWorkers,
		FlushBytes:    config.LogsConduitSettings.FlushBytes,
		FlushInterval: config.LogsConduitSettings.FlushInterval,
		OnError:       config.LogsConduitSettings.Conduit.onBulkError,
		OnFlushStart:  config.LogsConduitSettings.Conduit.onBulkFlushStart,
		OnFlushEnd:    config.LogsConduitSettings.Conduit.onBulkFlushEnd,
		Index:         config.LogsConduitSettings.Token,
		Human:         config.LogsConduitSettings.Human,
		Pipeline:      config.LogsConduitSettings.Pipeline,
		Pretty:        config.LogsConduitSettings.Pretty,
		Refresh:       config.LogsConduitSettings.Refresh,
		Timeout:       config.LogsConduitSettings.Timeout,
	}

	conduit.indexer.bulk, err = esutil.NewBulkIndexer(conduit.indexer.config)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewLogsExporter(
		config,
		conduit.PushLogsData,
		exporterhelper.WithShutdown(conduit.Shutdown),
	)

}

// PushLogsData TODO  doc comment
func (conduit *LogsConduit) PushLogsData(ctx context.Context, source data.Logs) (droppedTimeSeries int, err error) {

	var item *esutil.BulkIndexerItem

	resources := source.ResourceLogs()
	for i := 0; i < resources.Len(); i++ {
		resource := resources.At(i)
		if resource.IsNil() {
			continue // TODO Log this?
		}
		logs := resource.Logs()
		for j := 0; j < logs.Len(); j++ {
			log := logs.At(j)
			if log.IsNil() {
				continue
			}

			item, err = conduit.NewBulkIndexerItem(log)
			if err != nil {
				return 0, err
			}

			err = conduit.indexer.bulk.Add(context.Background(), *item)
			if err != nil {
				return 0, err
			}
		}
	}

	return 0, nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (conduit *LogsConduit) Shutdown(ctx context.Context) error {
	conduit.indexer.bulk.Close(ctx)
	return nil
}

var countSuccessful uint64

func (conduit *LogsConduit) itemOnSuccess(ctx context.Context, item esutil.BulkIndexerItem, response esutil.BulkIndexerResponseItem) {
	atomic.AddUint64(&countSuccessful, 1)
}

func (conduit *LogsConduit) itemOnFailure(ctx context.Context, item esutil.BulkIndexerItem, response esutil.BulkIndexerResponseItem, err error) {
	if err != nil {
		log.Printf("ERROR: %s", err)
	} else {
		log.Printf("ERROR: %s: %s", response.Error.Type, response.Error.Reason)
	}
}

func (conduit *LogsConduit) onBulkError(context.Context, error) {
	// TODO - Called for indexer errors.
}

func (conduit *LogsConduit) onBulkFlushStart(ctx context.Context) context.Context {
	// TODO Called when the flush starts.
	return ctx
}

func (conduit *LogsConduit) onBulkFlushEnd(context.Context) {
	// TODO Called when the flush ends.
}

func (conduit *LogsConduit) retryStrategy(i int) time.Duration {
	if i == 1 {
		conduit.backoff.Reset()
	}
	return conduit.backoff.NextBackOff()
}

// EsDoc TODO Doc comment
type EsDoc struct {
	Token           string            `json:"token"`
	Timestamp       string            `json:"@timestamp"` //A date field, on which log retention is based. If it's not present, it will be added automatically when the log event is received by Sematext. See Supported Date Formats.
	Facility        string            `json:"facility"`   //	A single-valued field used by syslog to indicate the facility level. Sematext stores the keyword values of these levels (such as user or auth).
	Severity        string            `json:"severity"`   //	A single-valued field and should contain the log level, such as error or info.
	Severitynumeric v1.SeverityNumber `json:"severity_numeric"`
	Spanid          []byte            `json:"span_id"`
	Traceid         []byte            `json:"trace_id"`
	Message         string            `json:"message"`

	//host string // A single-valued field and should contain the ID, typically a hostname, of the device or server sending logs.
	//source string //A single-valued field and should contain the ID or descriptor of where the data is coming from. For example, this could be a file name or even a full path to a filename, or the name of the application or process
	//syslogtag string   // A single-valued field used by syslog to indicate the name and the PID of the application generating the event (for example, httpd[215]:).
	//tags      []string // A multi-valued array field that can contain zero or more tags. Tags can contain multiple tokens separated by space.
	//message   string   // A string field that can contain any sort of text (usually the original log line or some other free text).
}

//NewBulkIndexerItem TODO Doc comment
func (conduit *LogsConduit) NewBulkIndexerItem(logRecord pdata.LogRecord) (*esutil.BulkIndexerItem, error) {
	var item *esutil.BulkIndexerItem
	var esdoc EsDoc

	esdoc = EsDoc{}
	esdoc.Token = conduit.config.LogsConduitSettings.Token
	esdoc.Timestamp = logRecord.Timestamp().String()
	esdoc.Facility = logRecord.SeverityText()
	esdoc.Severity = logRecord.SeverityText()
	esdoc.Severitynumeric = logRecord.SeverityNumber()
	esdoc.Spanid = logRecord.SpanID()
	esdoc.Traceid = logRecord.TraceID()
	// esdoc.? = logRecord.GetFlags() (syslog trace flags)
	// esdoc.? = logRecord.GetAttributes() array of key-value objects
	//esdoc.host = ?
	//esdoc.source = ?
	//esdoc.syslogtag = ?
	//esdoc.tags = ?
	//esdoc.geo.location  = ?
	//esdoc.geo.city_name = ?
	//esdoc.geo.region = ?
	//esdoc.geo.region_iso_code = ?
	//esdoc.geo.country_name = ?
	//esdoc.geo.country_iso_code = ?
	//esdoc.geo.continent_name = ?

	esdoc.Message = logRecord.Body()
	data, err := json.Marshal(esdoc)
	if err != nil {
		return nil, err
	}

	item = &esutil.BulkIndexerItem{
		Action:    "index",
		Index:     conduit.config.MetricsConduitSettings.Token,
		Body:      bytes.NewReader(data),
		OnSuccess: conduit.itemOnSuccess,
		OnFailure: conduit.itemOnFailure,
	}

	return item, nil
}
