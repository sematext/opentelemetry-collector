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
	otlptrace_goproto "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/stretchr/testify/assert"

	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
)

func TestSpanCount(t *testing.T) {
	md := NewTraces()
	assert.EqualValues(t, 0, md.SpanCount())

	md.ResourceSpans().Resize(1)
	assert.EqualValues(t, 0, md.SpanCount())

	md.ResourceSpans().At(0).InstrumentationLibrarySpans().Resize(1)
	assert.EqualValues(t, 0, md.SpanCount())

	md.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	assert.EqualValues(t, 1, md.SpanCount())

	rms := md.ResourceSpans()
	rms.Resize(3)
	rms.At(0).InstrumentationLibrarySpans().Resize(1)
	rms.At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	rms.At(1).InstrumentationLibrarySpans().Resize(1)
	rms.At(2).InstrumentationLibrarySpans().Resize(1)
	rms.At(2).InstrumentationLibrarySpans().At(0).Spans().Resize(5)
	assert.EqualValues(t, 6, md.SpanCount())
}

func TestSpanCountWithNils(t *testing.T) {
	assert.EqualValues(t, 0, TracesFromOtlp([]*otlptrace.ResourceSpans{nil, {}}).SpanCount())
	assert.EqualValues(t, 0, TracesFromOtlp([]*otlptrace.ResourceSpans{
		{
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{nil, {}},
		},
	}).SpanCount())
	assert.EqualValues(t, 2, TracesFromOtlp([]*otlptrace.ResourceSpans{
		{
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{
					Spans: []*otlptrace.Span{nil, {}},
				},
			},
		},
	}).SpanCount())
}

func TestTraceID(t *testing.T) {
	tid := NewTraceID(nil)
	assert.EqualValues(t, []byte(nil), tid.Bytes())

	tid = NewTraceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	assert.EqualValues(t, []byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}, tid.Bytes())
}

func TestToFromOtlp(t *testing.T) {
	otlp := []*otlptrace.ResourceSpans(nil)
	td := TracesFromOtlp(otlp)
	assert.EqualValues(t, NewTraces(), td)
	assert.EqualValues(t, otlp, TracesToOtlp(td))
	// More tests in ./tracedata/trace_test.go. Cannot have them here because of
	// circular dependency.
}

func TestResourceSpansWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate ResourceSpans as pdata struct.
	pdataRS := generateTestResourceSpans()

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := gogoproto.Marshal(*pdataRS.orig)
	assert.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoRS otlptrace_goproto.ResourceSpans
	err = goproto.Unmarshal(wire1, &goprotoRS)
	assert.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoRS)
	assert.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var gogoprotoRS2 otlptrace.ResourceSpans
	err = gogoproto.Unmarshal(wire1, &gogoprotoRS2)
	assert.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.True(t, gogoproto.Equal(*pdataRS.orig, &gogoprotoRS2))
}
