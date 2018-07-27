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

// Package all is used to import all known api APIs for their side effects.
package all

import (
	"unsafe"

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapil/executor"
)

// #include "gapil/runtime/cc/runtime.h"
// extern gapil_module prebuilt_host;
// extern gapil_module prebuilt_armv7a;
import "C"

func init() {
	executor.RegisterPrebuilt(executor.Config{
		Execute: true, Optimize: true,
	}, (unsafe.Pointer)(&C.prebuilt_host))
	executor.RegisterPrebuilt(executor.Config{
		CaptureABI: device.AndroidARM, Execute: true, Optimize: true,
	}, (unsafe.Pointer)(&C.prebuilt_armv7a))
	executor.RegisterPrebuilt(executor.Config{
		CaptureABI: device.AndroidARMv7a, Execute: true, Optimize: true,
	}, (unsafe.Pointer)(&C.prebuilt_armv7a))
}
