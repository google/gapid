// Copyright (C) 2017 Google Inc.
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

package vulkan

import "github.com/google/gapid/gapis/api"

// A placeholder interface for draw calls to conform the DrawCall annotation in
// API files. In Vulkan, all the drawing actually happens in the execution of
// vkQueueSubmit and all the drawing information should be found in the state
// when vkQueueSubmit is called.
type drawCall interface {
	api.Cmd
}
