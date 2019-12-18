// Copyright (C) 2019 Google Inc.
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

package adreno

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/perfetto"
	"github.com/google/gapid/gapis/trace/android/validate"
)

const (
	renderStageSlicesQuery = "" +
		"select name, depth, parent_stack_id " +
		"from gpu_slice " +
		"where track_id = %v " +
		"order by slice_id"
)

var (
	// All counters must be inside this array.
	counters = []validate.GpuCounter{
		{1, "Clocks / Second", validate.And(validate.IsNumber, validate.CheckLargerThanZero)},
		{3, "GPU % Utilization", validate.And(validate.IsNumber, validate.CheckLargerThanZero)},
		{21, "% Shaders Busy", validate.And(validate.IsNumber, validate.CheckLargerThanZero)},
		{26, "Fragment ALU Instructions / Sec (Full)", validate.And(validate.IsNumber, validate.CheckLargerThanZero)},
		{30, "Textures / Vertex", validate.And(validate.IsNumber, validate.CheckEqualTo(0.0))},
		{31, "Textures / Fragment", validate.And(validate.IsNumber, validate.CheckApproximateTo(1.0, 0.01))},
		{37, "% Time Shading Fragments", validate.And(validate.IsNumber, validate.CheckLargerThanZero)},
		{38, "% Time Shading Vertices", validate.And(validate.IsNumber, validate.CheckLargerThanZero)},
		{39, "% Time Compute", validate.And(validate.IsNumber, validate.CheckEqualTo(0.0))},
	}
)

type AdrenoValidator struct {
}

func (v *AdrenoValidator) validateRenderStage(ctx context.Context, processor *perfetto.Processor) error {
	tIds, err := validate.GetRenderStageTrackIDs(ctx, processor)
	if err != nil {
		return err
	}
	for _, tId := range tIds {
		queryResult, err := processor.Query(fmt.Sprintf(renderStageSlicesQuery, tId))
		if err != nil || queryResult.GetNumRecords() <= 0 {
			return log.Errf(ctx, err, "Failed to query with %v", fmt.Sprintf(renderStageSlicesQuery, tId))
		}
		columns := queryResult.GetColumns()
		names := columns[0].GetStringValues()

		// Skip slices until we hit the first 'Surface' slice.
		skipNum := -1
		hasSurfaceSlice := false
		hasRenderSlice := false
		for i, name := range names {
			if name == "Surface" {
				hasSurfaceSlice = true
				if skipNum == -1 {
					skipNum = i
				}
			}
			if name == "Render" {
				hasRenderSlice = true
			}
		}
		if !hasSurfaceSlice {
			return log.Errf(ctx, err, "Render stage verification failed: No Surface slice found")
		}
		if !hasRenderSlice {
			return log.Errf(ctx, err, "Render stage verification failed: No Render slice found")
		}
		depths := columns[1].GetLongValues()
		parentStackId := columns[2].GetLongValues()

		for i := skipNum; i < len(names); i++ {
			// Surface slice must be the top level slice, hence its depth is 0 and
			// it has no parent stack id.
			// Render slice must be a non-top-level slice, hence its depth must not be 0
			// and it must have a parent stack id.
			if names[i] == "Surface" {
				if depths[i] != 0 || parentStackId[i] != 0 {
					return log.Errf(ctx, err, "Render stage verification failed on Surface slice")
				}
			} else if names[i] == "Render" {
				if depths[i] <= 0 || parentStackId[i] <= 0 {
					return log.Errf(ctx, err, "Render stage verification failed on Render slice")
				}
			}
		}
	}
	return nil
}

func (v *AdrenoValidator) Validate(ctx context.Context, processor *perfetto.Processor) error {
	if err := validate.ValidateGpuCounters(ctx, processor, v.GetCounters()); err != nil {
		return err
	}
	if err := v.validateRenderStage(ctx, processor); err != nil {
		return err
	}

	return nil
}

func (v *AdrenoValidator) GetCounters() []validate.GpuCounter {
	return counters
}
