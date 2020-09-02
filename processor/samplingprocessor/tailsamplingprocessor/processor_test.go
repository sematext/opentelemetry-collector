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

package tailsamplingprocessor

import (
	"context"
	"sync"
	"testing"
	"time"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/processor/samplingprocessor/tailsamplingprocessor/idbatcher"
	"go.opentelemetry.io/collector/processor/samplingprocessor/tailsamplingprocessor/sampling"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

const (
	defaultTestDecisionWait = 30 * time.Second
)

var testPolicy = []PolicyCfg{{Name: "test-policy", Type: AlwaysSample}}

func TestSequentialTraceArrival(t *testing.T) {
	traceIds, batches := generateIdsAndBatches(128)
	cfg := Config{
		DecisionWait:            defaultTestDecisionWait,
		NumTraces:               uint64(2 * len(traceIds)),
		ExpectedNewTracesPerSec: 64,
		PolicyCfgs:              testPolicy,
	}
	sp, _ := newTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	tsp := sp.(*tailSamplingSpanProcessor)
	for _, batch := range batches {
		tsp.ConsumeTraces(context.Background(), batch)
	}

	for i := range traceIds {
		d, ok := tsp.idToTrace.Load(traceKey(traceIds[i]))
		require.True(t, ok, "Missing expected traceId")
		v := d.(*sampling.TraceData)
		require.Equal(t, int64(i+1), v.SpanCount, "Incorrect number of spans for entry %d", i)
	}
}

func TestConcurrentTraceArrival(t *testing.T) {
	traceIds, batches := generateIdsAndBatches(128)

	var wg sync.WaitGroup
	cfg := Config{
		DecisionWait:            defaultTestDecisionWait,
		NumTraces:               uint64(2 * len(traceIds)),
		ExpectedNewTracesPerSec: 64,
		PolicyCfgs:              testPolicy,
	}
	sp, _ := newTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	tsp := sp.(*tailSamplingSpanProcessor)
	for _, batch := range batches {
		// Add the same traceId twice.
		wg.Add(2)
		go func(td pdata.Traces) {
			tsp.ConsumeTraces(context.Background(), td)
			wg.Done()
		}(batch)
		go func(td pdata.Traces) {
			tsp.ConsumeTraces(context.Background(), td)
			wg.Done()
		}(batch)
	}

	wg.Wait()

	for i := range traceIds {
		d, ok := tsp.idToTrace.Load(traceKey(traceIds[i]))
		require.True(t, ok, "Missing expected traceId")
		v := d.(*sampling.TraceData)
		require.Equal(t, int64(i+1)*2, v.SpanCount, "Incorrect number of spans for entry %d", i)
	}
}

func TestSequentialTraceMapSize(t *testing.T) {
	traceIds, batches := generateIdsAndBatches(210)
	const maxSize = 100
	cfg := Config{
		DecisionWait:            defaultTestDecisionWait,
		NumTraces:               uint64(maxSize),
		ExpectedNewTracesPerSec: 64,
		PolicyCfgs:              testPolicy,
	}
	sp, _ := newTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	tsp := sp.(*tailSamplingSpanProcessor)
	for _, batch := range batches {
		tsp.ConsumeTraces(context.Background(), batch)
	}

	// On sequential insertion it is possible to know exactly which traces should be still on the map.
	for i := 0; i < len(traceIds)-maxSize; i++ {
		_, ok := tsp.idToTrace.Load(traceKey(traceIds[i]))
		require.False(t, ok, "Found unexpected traceId[%d] still on map (id: %v)", i, traceIds[i])
	}
}

func TestConcurrentTraceMapSize(t *testing.T) {
	_, batches := generateIdsAndBatches(210)
	const maxSize = 100
	var wg sync.WaitGroup
	cfg := Config{
		DecisionWait:            defaultTestDecisionWait,
		NumTraces:               uint64(maxSize),
		ExpectedNewTracesPerSec: 64,
		PolicyCfgs:              testPolicy,
	}
	sp, _ := newTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	tsp := sp.(*tailSamplingSpanProcessor)
	for _, batch := range batches {
		wg.Add(1)
		go func(td pdata.Traces) {
			tsp.ConsumeTraces(context.Background(), td)
			wg.Done()
		}(batch)
	}

	wg.Wait()

	// Since we can't guarantee the order of insertion the only thing that can be checked is
	// if the number of traces on the map matches the expected value.
	cnt := 0
	tsp.idToTrace.Range(func(_ interface{}, _ interface{}) bool {
		cnt++
		return true
	})
	require.Equal(t, maxSize, cnt, "Incorrect traces count on idToTrace")
}

func TestSamplingPolicyTypicalPath(t *testing.T) {
	const maxSize = 100
	const decisionWaitSeconds = 5
	// For this test explicitly control the timer calls and batcher, and set a mock
	// sampling policy evaluator.
	msp := new(exportertest.SinkTraceExporter)
	mpe := &mockPolicyEvaluator{}
	mtt := &manualTTicker{}
	tsp := &tailSamplingSpanProcessor{
		ctx:             context.Background(),
		nextConsumer:    msp,
		maxNumTraces:    maxSize,
		logger:          zap.NewNop(),
		decisionBatcher: newSyncIDBatcher(decisionWaitSeconds),
		policies:        []*Policy{{Name: "mock-policy", Evaluator: mpe, ctx: context.TODO()}},
		deleteChan:      make(chan traceKey, maxSize),
		policyTicker:    mtt,
	}

	_, batches := generateIdsAndBatches(210)
	currItem := 0
	numSpansPerBatchWindow := 10
	// First evaluations shouldn't have anything to evaluate, until decision wait time passed.
	for evalNum := 0; evalNum < decisionWaitSeconds; evalNum++ {
		for ; currItem < numSpansPerBatchWindow*(evalNum+1); currItem++ {
			tsp.ConsumeTraces(context.Background(), batches[currItem])
			require.True(t, mtt.Started, "Time ticker was expected to have started")
		}
		tsp.samplingPolicyOnTick()
		require.False(
			t,
			msp.SpansCount() != 0 || mpe.EvaluationCount != 0,
			"policy for initial items was evaluated before decision wait period",
		)
	}

	// Now the first batch that waited the decision period.
	mpe.NextDecision = sampling.Sampled
	tsp.samplingPolicyOnTick()
	require.False(
		t,
		msp.SpansCount() == 0 || mpe.EvaluationCount == 0,
		"policy should have been evaluated totalspans == %d and evaluationcount == %d",
		msp.SpansCount(),
		mpe.EvaluationCount,
	)

	require.Equal(t, numSpansPerBatchWindow, msp.SpansCount(), "not all spans of first window were accounted for")

	// Late span of a sampled trace should be sent directly down the pipeline exporter
	tsp.ConsumeTraces(context.Background(), batches[0])
	expectedNumWithLateSpan := numSpansPerBatchWindow + 1
	require.Equal(t, expectedNumWithLateSpan, msp.SpansCount(), "late span was not accounted for")
	require.Equal(t, 1, mpe.LateArrivingSpansCount, "policy was not notified of the late span")
}

func generateIdsAndBatches(numIds int) ([][]byte, []pdata.Traces) {
	traceIds := make([][]byte, numIds)
	var tds []pdata.Traces
	for i := 0; i < numIds; i++ {
		traceIds[i] = tracetranslator.UInt64ToByteTraceID(1, uint64(i+1))
		// Send each span in a separate batch
		for j := 0; j <= i; j++ {
			td := testdata.GenerateTraceDataOneSpan()
			span := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0)
			span.SetTraceID(traceIds[i])
			span.SetSpanID(tracetranslator.UInt64ToByteSpanID(uint64(i + 1)))
			tds = append(tds, td)
		}
	}

	return traceIds, tds
}

type mockPolicyEvaluator struct {
	NextDecision           sampling.Decision
	NextError              error
	EvaluationCount        int
	LateArrivingSpansCount int
	OnDroppedSpansCount    int
}

var _ sampling.PolicyEvaluator = (*mockPolicyEvaluator)(nil)

func (m *mockPolicyEvaluator) OnLateArrivingSpans(sampling.Decision, []*tracepb.Span) error {
	m.LateArrivingSpansCount++
	return m.NextError
}
func (m *mockPolicyEvaluator) Evaluate([]byte, *sampling.TraceData) (sampling.Decision, error) {
	m.EvaluationCount++
	return m.NextDecision, m.NextError
}
func (m *mockPolicyEvaluator) OnDroppedSpans([]byte, *sampling.TraceData) (sampling.Decision, error) {
	m.OnDroppedSpansCount++
	return m.NextDecision, m.NextError
}

type manualTTicker struct {
	Started bool
}

var _ tTicker = (*manualTTicker)(nil)

func (t *manualTTicker) Start(time.Duration) {
	t.Started = true
}

func (t *manualTTicker) OnTick() {
}

func (t *manualTTicker) Stop() {
}

type syncIDBatcher struct {
	sync.Mutex
	openBatch idbatcher.Batch
	batchPipe chan idbatcher.Batch
}

var _ idbatcher.Batcher = (*syncIDBatcher)(nil)

func newSyncIDBatcher(numBatches uint64) idbatcher.Batcher {
	batches := make(chan idbatcher.Batch, numBatches)
	for i := uint64(0); i < numBatches; i++ {
		batches <- nil
	}
	return &syncIDBatcher{
		batchPipe: batches,
	}
}

func (s *syncIDBatcher) AddToCurrentBatch(id idbatcher.ID) {
	s.Lock()
	s.openBatch = append(s.openBatch, id)
	s.Unlock()
}

func (s *syncIDBatcher) CloseCurrentAndTakeFirstBatch() (idbatcher.Batch, bool) {
	s.Lock()
	defer s.Unlock()
	firstBatch := <-s.batchPipe
	s.batchPipe <- s.openBatch
	s.openBatch = nil
	return firstBatch, true
}

func (s *syncIDBatcher) Stop() {
}
