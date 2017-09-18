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

import (
	"testing"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
)

func TestCallSub(t *testing.T) {
	ctx := log.Testing(t)
	e := externs{}
	s := api.NewStateWithEmptyAllocator(device.Little32)
	e.callSub(ctx, &VkCreateBuffer{}, 10, s, nil, subDovkCmdDispatch, &VkCmdDispatchArgs{})
}
