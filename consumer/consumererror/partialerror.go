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

package consumererror

import "go.opentelemetry.io/collector/consumer/pdata"

// PartialError can be used to signalize that a subset of received data failed to be processed or send.
// The preceding components in the pipeline can use this information for partial retries.
type PartialError struct {
	error
	failed pdata.Traces
}

// PartialTracesError creates PartialError for failed traces.
// Use this error type only when a subset of received data set failed to be processed or sent.
func PartialTracesError(err error, failed pdata.Traces) error {
	return PartialError{
		error:  err,
		failed: failed,
	}
}

// GetTraces returns failed traces.
func (err PartialError) GetTraces() pdata.Traces {
	return err.failed
}
