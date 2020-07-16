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

package resourceprocessor

import (
	"context"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/processor/attraction"
)

type resourceProcessor struct {
	attrProc *attraction.AttrProc
}

// ProcessTraces implements the TProcessor interface
func (rp *resourceProcessor) ProcessTraces(_ context.Context, td pdata.Traces) (pdata.Traces, error) {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		resource := rss.At(i).Resource()
		if resource.IsNil() {
			resource.InitEmpty()
		}
		attrs := resource.Attributes()
		rp.attrProc.Process(attrs)
	}
	return td, nil
}

// ProcessMetrics implements the MProcessor interface
func (rp *resourceProcessor) ProcessMetrics(_ context.Context, md pdata.Metrics) (pdata.Metrics, error) {
	imd := pdatautil.MetricsToInternalMetrics(md)
	rms := imd.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		resource := rms.At(i).Resource()
		if resource.IsNil() {
			resource.InitEmpty()
		}
		if resource.Attributes().Len() == 0 {
			resource.Attributes().InitEmptyWithCapacity(1)
		}
		rp.attrProc.Process(resource.Attributes())
	}
	return md, nil
}
