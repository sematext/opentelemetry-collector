// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package groupbytraceprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestMemoryCreateAndGetTrace(t *testing.T) {
	// prepare
	st := newMemoryStorage()

	traceIDs := [][]byte{
		{1, 2, 3, 4},
		{2, 3, 4, 5},
	}

	span := pdata.NewSpan()
	span.InitEmpty()

	ils := pdata.NewInstrumentationLibrarySpans()
	ils.InitEmpty()
	ils.Spans().Append(&span)

	baseTrace := pdata.NewResourceSpans()
	baseTrace.InitEmpty()
	baseTrace.InstrumentationLibrarySpans().Append(&ils)

	// test
	for _, traceID := range traceIDs {
		span.SetTraceID(pdata.NewTraceID(traceID))
		st.createOrAppend(traceID, baseTrace)
	}

	// verify
	assert.Equal(t, 2, st.count())
	for _, traceID := range traceIDs {
		expected := []pdata.ResourceSpans{baseTrace}
		expected[0].InstrumentationLibrarySpans().At(0).Spans().At(0).SetTraceID(traceID)

		retrieved, err := st.get(traceID)
		st.createOrAppend(traceID, expected[0])

		require.NoError(t, err)
		assert.Equal(t, expected, retrieved)
	}
}

func TestMemoryDeleteTrace(t *testing.T) {
	// prepare
	st := newMemoryStorage()

	traceID := pdata.NewTraceID([]byte{1, 2, 3, 4})
	span := pdata.NewSpan()
	span.InitEmpty()
	span.SetTraceID(traceID)

	ils := pdata.NewInstrumentationLibrarySpans()
	ils.InitEmpty()
	ils.Spans().Append(&span)

	trace := pdata.NewResourceSpans()
	trace.InitEmpty()
	trace.InstrumentationLibrarySpans().Append(&ils)

	st.createOrAppend(traceID, trace)

	// test
	deleted, err := st.delete(traceID)

	// verify
	require.NoError(t, err)
	assert.Equal(t, []pdata.ResourceSpans{trace}, deleted)

	retrieved, err := st.get(traceID)
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestMemoryAppendSpans(t *testing.T) {
	// prepare
	st := newMemoryStorage()

	traceID := pdata.NewTraceID([]byte{1, 2, 3, 4})
	span := pdata.NewSpan()
	span.InitEmpty()
	span.SetTraceID(traceID)
	span.SetSpanID(pdata.NewSpanID([]byte{1, 2, 3, 4}))

	ils := pdata.NewInstrumentationLibrarySpans()
	ils.InitEmpty()
	ils.Spans().Append(&span)

	batch := pdata.NewResourceSpans()
	batch.InitEmpty()
	batch.InstrumentationLibrarySpans().Append(&ils)

	st.createOrAppend(traceID, batch)

	secondSpan := pdata.NewSpan()
	secondSpan.InitEmpty()
	secondSpan.SetName("second-name")
	secondSpan.SetTraceID(traceID)
	secondSpan.SetSpanID(pdata.NewSpanID([]byte{5, 6, 7, 8}))

	secondIls := pdata.NewInstrumentationLibrarySpans()
	secondIls.InitEmpty()
	secondIls.Spans().Append(&secondSpan)

	secondBatch := pdata.NewResourceSpans()
	secondBatch.InitEmpty()
	secondBatch.InstrumentationLibrarySpans().Append(&secondIls)

	expectedIls := pdata.NewInstrumentationLibrarySpans()
	expectedIls.InitEmpty()
	expectedIls.Spans().Append(&span)

	expected := []pdata.ResourceSpans{
		pdata.NewResourceSpans(),
		pdata.NewResourceSpans(),
	}
	expected[0].InitEmpty()
	expected[0].InstrumentationLibrarySpans().Append(&expectedIls)

	expected[1].InitEmpty()
	expected[1].InstrumentationLibrarySpans().Append(&secondIls)

	// test
	err := st.createOrAppend(traceID, secondBatch)
	require.NoError(t, err)

	// override something in the second span, to make sure we are storing a copy
	secondSpan.SetName("changed-second-name")

	// verify
	retrieved, err := st.get(traceID)
	require.NoError(t, err)
	assert.Equal(t, "second-name", retrieved[1].InstrumentationLibrarySpans().At(0).Spans().At(0).Name())

	// now that we checked that the secondSpan change here didn't have an effect, revert
	// so that we can compare the that everything else has the same value
	secondSpan.SetName("second-name")
	assert.Equal(t, expected, retrieved)
}

func TestMemoryTraceIsBeingCloned(t *testing.T) {
	// prepare
	st := newMemoryStorage()
	traceID := pdata.NewTraceID([]byte{1, 2, 3, 4})
	span := pdata.NewSpan()
	span.InitEmpty()
	span.SetTraceID(traceID)
	span.SetSpanID(pdata.NewSpanID([]byte{1, 2, 3, 4}))
	span.SetName("should-not-be-changed")

	ils := pdata.NewInstrumentationLibrarySpans()
	ils.InitEmpty()
	ils.Spans().Append(&span)

	batch := pdata.NewResourceSpans()
	batch.InitEmpty()
	batch.InstrumentationLibrarySpans().Append(&ils)

	// test
	err := st.createOrAppend(traceID, batch)
	require.NoError(t, err)
	span.SetName("changed-trace")

	// verify
	retrieved, err := st.get(traceID)
	require.NoError(t, err)
	assert.Equal(t, "should-not-be-changed", retrieved[0].InstrumentationLibrarySpans().At(0).Spans().At(0).Name())
}

func TestCreateWithNilParameter(t *testing.T) {
	// prepare
	st := newMemoryStorage()
	traceID := []byte{1, 2, 3, 4}

	// test
	err := st.createOrAppend(traceID, pdata.NewResourceSpans())

	// verify
	require.Equal(t, errStorageNilResourceSpans, err)
}
