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

package device

// #cgo LDFLAGS: -ldeviceinfo -lprotobuf -lprotobuf_lite -lstdc++ -lpthread -lm -lcityhash
// #include "core/os/device/deviceinfo/cc/instance.h"
import "C"
import "unsafe"

func get_device() []byte {
	s := C.get_device_instance(nil)
	defer C.free_device_instance(s)
	buf := C.GoBytes(unsafe.Pointer(s.data), C.int(s.size))
	return append([]byte{}, buf...)
}
