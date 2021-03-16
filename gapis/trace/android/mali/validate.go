// Copyright (C) 2020 Google Inc.
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

package mali

import (
	"context"

	"github.com/google/gapid/gapis/perfetto"
	"github.com/google/gapid/gapis/trace/android/validate"
)

var (
	// All counters must be inside this array.
	counters = []validate.GpuCounter{
		{6, "GPU active cycles", counterChecker()},
		{8, "Fragment jobs", counterChecker()},
		{196, "Fragment active cycles", counterChecker()},
		{65536, "GPU utilization", counterChecker()},
		{65538, "Fragment queue utilization", counterChecker()},
		{65579, "Execution core utilization", counterChecker()},
	}
)

func counterChecker() validate.Checker {
	return validate.And(validate.IsNumber, validate.CheckNonNegative(), validate.Not(validate.CheckAllEqualTo(0)))
}

type MaliValidator struct {
}

func (v *MaliValidator) Validate(ctx context.Context, processor *perfetto.Processor) error {
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

func (v *MaliValidator) GetCounters() []validate.GpuCounter {
	return counters
}
