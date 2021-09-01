// Copyright (C) 2021 Google Inc.
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

package api

// Handle is an interface implemented by every handle type in the API. Handles
// can be represented by a uint64 as per the Vulkan spec
// (https://www.khronos.org/registry/vulkan/specs/1.2-extensions/html/chap3.html#fundamentals-objectmodel-overview).
type Handle interface {
	Labeled

	// Handle returns this handle's value as a uint64 for display.
	Handle() uint64
}
