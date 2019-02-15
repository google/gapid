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

package host

// #include "core/os/device/deviceinfo/cc/instance.h"
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/os/device"
)

func getHostDevice() device.Instance {
	s := C.get_device_instance()
	defer C.free_device_instance(s)
	if s.data == nil {
		panic(fmt.Errorf("Failed to get host machine information: %v",
			C.GoString(C.get_device_instance_error())))
	}
	buf := C.GoBytes(unsafe.Pointer(s.data), C.int(s.size))
	var device device.Instance
	if err := proto.NewBuffer(buf).Unmarshal(&device); err != nil {
		panic(err)
	}
	return device
}
