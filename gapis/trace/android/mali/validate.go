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
	"strings"

	"github.com/google/gapid/gapis/perfetto"
	"github.com/google/gapid/gapis/trace/android/validate"
)

var (
	// jmCounters are the hardware counters found on JM based GPUs.
	jmCounters = []validate.GpuCounter{
		{6, "GPU active cycles", counterChecker()},
		{8, "Fragment jobs", counterChecker()},
		{196, "Fragment active cycles", counterChecker()},
		{65536, "GPU utilization", counterChecker()},
		{65538, "Fragment queue utilization", counterChecker()},
		{65579, "Execution core utilization", counterChecker()},
	}

	// csfCounters are the hardware counters found on CSF based GPUs.
	csfCounters = []validate.GpuCounter{
		{4, "GPU active cycles", counterChecker()},
		{6, "Any iterator active cycles", counterChecker()},
		{33, "Fragment jobs", counterChecker()},
		{196, "Fragment active cycles", counterChecker()},
		{65536, "GPU utilization", counterChecker()},
		{65581, "Execution core utilization", counterChecker()},
	}
)

func counterChecker() validate.Checker {
	return validate.And(validate.IsNumber, validate.CheckNonNegative(), validate.Not(validate.CheckAllEqualTo(0)))
}

type MaliValidator struct {
	gpuName string
}

func NewMaliValidator(gpuName string) *MaliValidator {
	return &MaliValidator{gpuName}
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
	gpuName := v.gpuName
	if strings.HasSuffix(gpuName, "G31") ||
		strings.HasSuffix(gpuName, "G51") ||
		strings.HasSuffix(gpuName, "G52") ||
		strings.HasSuffix(gpuName, "G71") ||
		strings.HasSuffix(gpuName, "G72") ||
		strings.HasSuffix(gpuName, "G76") ||
		strings.HasSuffix(gpuName, "G57") ||
		strings.HasSuffix(gpuName, "G68") ||
		strings.HasSuffix(gpuName, "G77") ||
		strings.HasSuffix(gpuName, "G78") ||
		strings.HasSuffix(gpuName, "G78AE") {
		return jmCounters
	} else {
		return csfCounters
	}
}
