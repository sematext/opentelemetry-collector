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

package pdata

import (
	"testing"

	gogoproto "github.com/gogo/protobuf/proto"
	goproto "github.com/golang/protobuf/proto"
	otlpmetrics_goproto "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
	"github.com/stretchr/testify/assert"

	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
)

func TestResourceMetricsWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate ResourceMetrics as pdata struct.
	pdataRM := generateTestResourceMetrics()

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := gogoproto.Marshal(*pdataRM.orig)
	assert.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoRM otlpmetrics_goproto.ResourceMetrics
	err = goproto.Unmarshal(wire1, &goprotoRM)
	assert.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoRM)
	assert.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var gogoprotoRM otlpmetrics.ResourceMetrics
	err = gogoproto.Unmarshal(wire1, &gogoprotoRM)
	assert.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.True(t, gogoproto.Equal(*pdataRM.orig, &gogoprotoRM))
}
