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
		Api:                            "Vulkan",
		CanDisablePcs:                  false,
		MidExecutionCaptureSupport:     service.FeatureStatus_Supported,
		CanEnableUnsupportedExtensions: true,
		RequiresApplication:            true,
	}
}

// GLESTraceOptions returns the default trace options for GLES.
func GLESTraceOptions() *service.TraceTypeCapabilities {
	return &service.TraceTypeCapabilities{
		Type:                           service.TraceType_Graphics,
		Api:                            "OpenGLES",
		CanDisablePcs:                  true,
		MidExecutionCaptureSupport:     service.FeatureStatus_Experimental,
		CanEnableUnsupportedExtensions: false,
		RequiresApplication:            true,
	}
}

// PerfettoTraceOptions returns the default trace options for Perfetto.
func PerfettoTraceOptions() *service.TraceTypeCapabilities {
	return &service.TraceTypeCapabilities{
		Type:                           service.TraceType_Perfetto,
		CanDisablePcs:                  false,
		MidExecutionCaptureSupport:     service.FeatureStatus_Supported,
		CanEnableUnsupportedExtensions: false,
		RequiresApplication:            false,
	}
}
