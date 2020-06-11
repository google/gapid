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

package flags

import (
	"flag"
)

// Experimental features are hidden behind the flags. All experimental feature flags must:
//     1) be named --experimental-enable-<feature-name>
//     2) be by default false/off
//     3) be removed once the feature is no longer in experiment
var (
	EnableFrameLifecycle = flag.Bool("experimental-enable-frame-lifecycle", false, "Enable the experimental feature Frame Lifecycle.")
	EnableVulkanTracing  = flag.Bool("experimental-enable-vulkan-tracing", false, "Enable the experimental feature Vulkan tracing.")
	EnableAngleTracing   = flag.Bool("experimental-enable-angle-tracing", false, "Enable the experimental feature ANGLE tracing.")
)
