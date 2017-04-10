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

import (
	"context"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/os/device"
)

var (
	host     device.Instance
	hostOnce sync.Once
)

// Instance returns the device information for the host computer running the
// code.
func Instance(ctx context.Context) *device.Instance {
	hostOnce.Do(func() {
		buf, err := getHostDevice()
		if err != nil {
			panic(fmt.Errorf("Failed to get host machine information: %v", err))
		}
		if err := proto.NewBuffer(buf).Unmarshal(&host); err != nil {
			panic(err)
		}
	})
	return &host
}
