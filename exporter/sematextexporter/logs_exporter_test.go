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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/internal/data/testdata"
)

func TestLoggingLogExporterNoErrors(t *testing.T) {
	lle, err := NewLogsExporter(configmodels.ExporterSettings{}, "debug", zap.NewNop())
	require.NotNil(t, lle)
	assert.NoError(t, err)

	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogDataEmpty()))
	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogDataOneEmptyResourceLogs()))
	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogDataOneEmptyOneNilResourceLogs()))
	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogDataNoLogRecords()))
	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogDataOneEmptyLogs()))

	assert.NoError(t, lle.Shutdown(context.Background()))
}
