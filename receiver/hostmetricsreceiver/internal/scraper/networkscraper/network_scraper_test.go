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

package networkscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/net"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

func TestScrapeMetrics(t *testing.T) {
	type testCase struct {
		name              string
		bootTimeFunc      func() (uint64, error)
		ioCountersFunc    func(bool) ([]net.IOCountersStat, error)
		connectionsFunc   func(string) ([]net.ConnectionStat, error)
		expectedStartTime pdata.TimestampUnixNano
		initializationErr string
		expectedErr       string
	}

	testCases := []testCase{
		{
			name: "Standard",
		},
		{
			name:              "Validate Start Time",
			bootTimeFunc:      func() (uint64, error) { return 100, nil },
			expectedStartTime: 100 * 1e9,
		},
		{
			name:              "Boot Time Error",
			bootTimeFunc:      func() (uint64, error) { return 0, errors.New("err1") },
			initializationErr: "err1",
		},
		{
			name:           "IOCounters Error",
			ioCountersFunc: func(bool) ([]net.IOCountersStat, error) { return nil, errors.New("err2") },
			expectedErr:    "err2",
		},
		{
			name:            "Connections Error",
			connectionsFunc: func(string) ([]net.ConnectionStat, error) { return nil, errors.New("err3") },
			expectedErr:     "err3",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper := newNetworkScraper(context.Background(), &Config{})
			if test.bootTimeFunc != nil {
				scraper.bootTime = test.bootTimeFunc
			}
			if test.ioCountersFunc != nil {
				scraper.ioCounters = test.ioCountersFunc
			}
			if test.connectionsFunc != nil {
				scraper.connections = test.connectionsFunc
			}

			err := scraper.Initialize(context.Background())
			if test.initializationErr != "" {
				assert.EqualError(t, err, test.initializationErr)
				return
			}
			require.NoError(t, err, "Failed to initialize network scraper: %v", err)
			defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

			metrics, err := scraper.ScrapeMetrics(context.Background())
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err, "Failed to scrape metrics: %v", err)

			assert.Equal(t, 5, metrics.Len())

			assertNetworkIOMetricValid(t, metrics.At(0), networkPacketsDescriptor, test.expectedStartTime)
			assertNetworkIOMetricValid(t, metrics.At(1), networkDroppedPacketsDescriptor, test.expectedStartTime)
			assertNetworkIOMetricValid(t, metrics.At(2), networkErrorsDescriptor, test.expectedStartTime)
			assertNetworkIOMetricValid(t, metrics.At(3), networkIODescriptor, test.expectedStartTime)

			assertNetworkTCPConnectionsMetricValid(t, metrics.At(4))
		})
	}
}

func assertNetworkIOMetricValid(t *testing.T, metric pdata.Metric, descriptor pdata.MetricDescriptor, startTime pdata.TimestampUnixNano) {
	internal.AssertDescriptorEqual(t, descriptor, metric.MetricDescriptor())
	if startTime != 0 {
		internal.AssertInt64MetricStartTimeEquals(t, metric, startTime)
	}
	assert.Equal(t, 2, metric.Int64DataPoints().Len())
	internal.AssertInt64MetricLabelHasValue(t, metric, 0, directionLabelName, transmitDirectionLabelValue)
	internal.AssertInt64MetricLabelHasValue(t, metric, 1, directionLabelName, receiveDirectionLabelValue)
}

func assertNetworkTCPConnectionsMetricValid(t *testing.T, metric pdata.Metric) {
	internal.AssertDescriptorEqual(t, networkTCPConnectionsDescriptor, metric.MetricDescriptor())
	internal.AssertInt64MetricLabelExists(t, metric, 0, stateLabelName)
	assert.Equal(t, 12, metric.Int64DataPoints().Len())
}
