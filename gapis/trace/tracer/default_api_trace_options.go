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

package tracer

import (
	"github.com/google/gapid/gapis/service"
)

// VulkanTraceOptions returns the default trace options for Vulkan.
func VulkanTraceOptions() *service.TraceTypeCapabilities {
	return &service.TraceTypeCapabilities{
		Type:                           service.TraceType_Graphics,
		CanEnableUnsupportedExtensions: true,
		RequiresApplication:            true,
		CanSelectProcessName:           true,
	}
}

// AngleTraceOptions returns the default trace options for Angle.
func AngleTraceOptions() *service.TraceTypeCapabilities {
	return &service.TraceTypeCapabilities{
		Type:                           service.TraceType_ANGLE,
		CanEnableUnsupportedExtensions: true,
		RequiresApplication:            true,
		CanSelectProcessName:           true,
	}
}

// PerfettoTraceOptions returns the default trace options for Perfetto.
func PerfettoTraceOptions() *service.TraceTypeCapabilities {
	return &service.TraceTypeCapabilities{
		Type:                           service.TraceType_Perfetto,
		CanEnableUnsupportedExtensions: false,
		RequiresApplication:            false,
		CanSelectProcessName:           false,
	}
}
