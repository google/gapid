// Copyright (C) 2018 Google Inc.
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

package validate

import (
	"context"
	"fmt"
	"math"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/perfetto"
	perfetto_service "github.com/google/gapid/gapis/perfetto/service"
)

const (
	counterIDQuery     = "select id from gpu_counter_track where name = '%v'"
	counterValuesQuery = "" +
		"select value from counter " +
		"where track_id = %v order by ts " +
		"limit %v offset 10"
	renderStageTrackIDQuery = "select id from gpu_track where scope = 'gpu_render_stage'"
	sampleCounter           = 100
)

// Checker is a function that checks the validity of the values of the given result set column.
type Checker func(column *perfetto_service.QueryResult_ColumnValues, columnType perfetto_service.QueryResult_ColumnDesc_Type) bool

// GpuCounter represents a GPU counter for which the profiling data is validated.
type GpuCounter struct {
	Id    uint32
	Name  string
	Check Checker
}

// Validator is an interface implemented by the various hardware specific validators.
type Validator interface {
	Validate(ctx context.Context, processor *perfetto.Processor) error
	GetCounters() []GpuCounter
}

// And returns a checker that is only valid if both of its arguments are.
func And(c1, c2 Checker) Checker {
	return func(column *perfetto_service.QueryResult_ColumnValues, columnType perfetto_service.QueryResult_ColumnDesc_Type) bool {
		return c1(column, columnType) && c2(column, columnType)
	}
}

// IsNumber is a checker that checks that the column is a number type.
func IsNumber(column *perfetto_service.QueryResult_ColumnValues, columnType perfetto_service.QueryResult_ColumnDesc_Type) bool {
	if columnType != perfetto_service.QueryResult_ColumnDesc_LONG && columnType != perfetto_service.QueryResult_ColumnDesc_DOUBLE {
		return false
	}
	return true
}

// CheckLargerThanZero is a checker that checks that the values are all greater than zero.
func CheckLargerThanZero(column *perfetto_service.QueryResult_ColumnValues, columnType perfetto_service.QueryResult_ColumnDesc_Type) bool {
	longValues := column.GetLongValues()
	doubleValues := column.GetDoubleValues()
	for i := 0; i < sampleCounter; i++ {
		if columnType == perfetto_service.QueryResult_ColumnDesc_LONG {
			if longValues[i] <= 0 {
				return false
			}
		} else if columnType == perfetto_service.QueryResult_ColumnDesc_DOUBLE {
			if doubleValues[i] <= 0.0 {
				return false
			}
		}
	}
	return true
}

// CheckEqualTo returns a checker that checks that all returned value equal the given value.
func CheckEqualTo(num float64) Checker {
	return func(column *perfetto_service.QueryResult_ColumnValues, columnType perfetto_service.QueryResult_ColumnDesc_Type) bool {
		longValues := column.GetLongValues()
		doubleValues := column.GetDoubleValues()
		for i := 0; i < sampleCounter; i++ {
			if columnType == perfetto_service.QueryResult_ColumnDesc_LONG {
				if longValues[i] != int64(num) {
					return false
				}
			} else if columnType == perfetto_service.QueryResult_ColumnDesc_DOUBLE {
				if doubleValues[i] != num {
					return false
				}
			}
		}
		return true
	}
}

// CheckApproximateTo returns a checker that checks that values are within a margin of the given value.
func CheckApproximateTo(num, err float64) Checker {
	return func(column *perfetto_service.QueryResult_ColumnValues, columnType perfetto_service.QueryResult_ColumnDesc_Type) bool {
		longValues := column.GetLongValues()
		doubleValues := column.GetDoubleValues()
		for i := 0; i < sampleCounter; i++ {
			if columnType == perfetto_service.QueryResult_ColumnDesc_LONG {
				if math.Abs(num-float64(longValues[i])) > err {
					return false
				}
			} else if columnType == perfetto_service.QueryResult_ColumnDesc_DOUBLE {
				if math.Abs(num-doubleValues[i]) > err {
					return false
				}
			}
		}
		return true
	}
}

// ValidateGpuCounters validates the GPU counters.
// GPU counters validation will fail in the below cases:
// 1. Fail to query
// 2. Missing GPU counter samples
// 3. Fail to check
func ValidateGpuCounters(ctx context.Context, processor *perfetto.Processor, counters []GpuCounter) error {
	for _, counter := range counters {
		queryResult, err := processor.Query(fmt.Sprintf(counterIDQuery, counter.Name))
		if err != nil {
			return log.Errf(ctx, err, "Failed to query with %v", fmt.Sprintf(counterIDQuery, counter.Name))
		}
		if len(queryResult.GetColumns()) != 1 {
			return log.Errf(ctx, err, "Expect one result with query: %v", fmt.Sprintf(counterIDQuery, counter.Name))
		}
		var counterID int64
		for _, column := range queryResult.GetColumns() {
			longValues := column.GetLongValues()
			if len(longValues) != 1 {
				// This should never happen, but sill have a check.
				return log.Err(ctx, nil, "Query result is not 1.")
			}
			counterID = longValues[0]
			break
		}
		queryResult, err = processor.Query(fmt.Sprintf(counterValuesQuery, counterID, sampleCounter))
		if err != nil {
			return log.Errf(ctx, err, "Failed to query with %v for counter %v", fmt.Sprintf(counterValuesQuery, counterID), counter)
		}

		// Query exactly #sampleCounter samples, fail if not enough samples
		if queryResult.GetNumRecords() != sampleCounter {
			return log.Errf(ctx, nil, "Number of samples is incorrect for counter: %v %v", counter, queryResult.GetNumRecords())
		}

		if !counter.Check(queryResult.GetColumns()[0], queryResult.GetColumnDescriptors()[0].GetType()) {
			return log.Errf(ctx, nil, "Check failed for counter: %v", counter)
		}
	}
	return nil
}

// GetRenderStageTrackIDs returns all track ids from gpu_track where the scope is gpu_render_stage
func GetRenderStageTrackIDs(ctx context.Context, processor *perfetto.Processor) ([]int64, error) {
	queryResult, err := processor.Query(renderStageTrackIDQuery)
	if err != nil || queryResult.GetNumRecords() <= 0 {
		return []int64{}, log.Err(ctx, err, "Failed to query GPU render stage track ids")
	}
	result := make([]int64, queryResult.GetNumRecords())
	for i, v := range queryResult.GetColumns()[0].GetLongValues() {
		result[i] = v
	}
	return result, nil
}
