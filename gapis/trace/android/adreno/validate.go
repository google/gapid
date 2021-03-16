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

	"github.com/google/gapid/gapis/perfetto"
	"github.com/google/gapid/gapis/trace/android/validate"
)

var (
	// All counters must be inside this array.
	counters = []validate.GpuCounter{
		{1, "Clocks / Second", validate.And(validate.IsNumber, validate.CheckLargerThanZero())},
		{3, "GPU % Utilization", validate.And(validate.IsNumber, validate.CheckLargerThanZero())},
		{21, "% Shaders Busy", validate.And(validate.IsNumber, validate.CheckLargerThanZero())},
		{26, "Fragment ALU Instructions / Sec (Full)", validate.And(validate.IsNumber, validate.CheckLargerThanZero())},
		{30, "Textures / Vertex", validate.And(validate.IsNumber, validate.CheckAllEqualTo(0.0))},
		{31, "Textures / Fragment", validate.And(validate.IsNumber, validate.CheckAverageApproximateTo(1.0, 0.1))},
		{37, "% Time Shading Fragments", validate.And(validate.IsNumber, validate.CheckLargerThanZero())},
		{38, "% Time Shading Vertices", validate.And(validate.IsNumber, validate.CheckLargerThanZero())},
		{39, "% Time Compute", validate.And(validate.IsNumber, validate.CheckAllEqualTo(0.0))},
	}
)

type AdrenoValidator struct {
}

func (v *AdrenoValidator) Validate(ctx context.Context, processor *perfetto.Processor) error {
	if err := validate.ValidateGpuCounters(ctx, processor, v.GetCounters()); err != nil {
		return err
	}
	if err := validate.ValidateGpuSlices(ctx, processor); err != nil {
		return err
	}
	if err := validate.ValidateVulkanEvents(ctx, processor); err != nil {
		return err
	}

	return nil
}

func (v *AdrenoValidator) GetCounters() []validate.GpuCounter {
	return counters
}
