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

package compiler

import (
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapil/compiler/mangling"
)

// Settings describe the options used to compile an API file.
type Settings struct {
	// The name of the module to produce.
	// This will be a global of the gapil_module type.
	Module string

	// TargetABI is the ABI used by the device running the compiled code.
	TargetABI *device.ABI

	// CaptureABI is the ABI of the device used to generate the capture data.
	CaptureABI *device.ABI

	// The mangler used for global functions and types.
	Mangler mangling.Mangler

	// Prefix for mangler
	Namespaces []string

	// EmitDebug is true if the compiler should emit DWARF debug info.
	EmitDebug bool

	// WriteToApplicationPool is true if writes to the application pool should
	// be performed.
	WriteToApplicationPool bool

	// Plugins are the list of extensions to include in the compilation.
	Plugins []Plugin
}
