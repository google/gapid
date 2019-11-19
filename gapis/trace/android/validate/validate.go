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
	counterIdQuery = "" +
		"select counter_id from counter_definitions " +
		"where name = '%v'"
	counterValuesQuery = "" +
		"select value from counter_values " +
		"where counter_id = %v order by ts " +
		"limit %v offset 10"
	sampleCounter = 100
)

type Checker func(column *perfetto_service.QueryResult_ColumnValues, columnType perfetto_service.QueryResult_ColumnDesc_Type) bool

type GpuCounter struct {
	Id    uint32
	Name  string
	Check Checker
}

type Validator interface {
	Validate(ctx context.Context, processor *perfetto.Processor) error
	GetCounters() []GpuCounter
}

func And(c1, c2 Checker) Checker {
	return func(column *perfetto_service.QueryResult_ColumnValues, columnType perfetto_service.QueryResult_ColumnDesc_Type) bool {
		return c1(column, columnType) && c2(column, columnType)
	}
}

func IsNumber(column *perfetto_service.QueryResult_ColumnValues, columnType perfetto_service.QueryResult_ColumnDesc_Type) bool {
	if columnType != perfetto_service.QueryResult_ColumnDesc_LONG && columnType != perfetto_service.QueryResult_ColumnDesc_DOUBLE {
		return false
	}
	return true
}

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

// GPU counters validation will fail in the below cases:
// 1. Fail to query
// 2. Missing GPU counter samples
// 3. Fail to check
func ValidateGpuCounters(ctx context.Context, processor *perfetto.Processor, counters []GpuCounter) error {
	for _, counter := range counters {
		queryResult, err := processor.Query(fmt.Sprintf(counterIdQuery, counter.Name))
		if err != nil {
			return log.Errf(ctx, err, "Failed to query with %v", fmt.Sprintf(counterIdQuery, counter.Name))
		}
		if len(queryResult.GetColumns()) != 1 {
			return log.Errf(ctx, err, "Expect one result with query: %v", fmt.Sprintf(counterIdQuery, counter.Name))
		}
		var counterId int64
		for _, column := range queryResult.GetColumns() {
			longValues := column.GetLongValues()
			if len(longValues) != 1 {
				// This should never happen, but sill have a check.
				return log.Err(ctx, nil, "Query result is not 1.")
			}
			counterId = longValues[0]
			break
		}
		queryResult, err = processor.Query(fmt.Sprintf(counterValuesQuery, counterId, sampleCounter))
		if err != nil {
			return log.Errf(ctx, err, "Failed to query with %v for counter %v", fmt.Sprintf(counterValuesQuery, counterId), counter)
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
