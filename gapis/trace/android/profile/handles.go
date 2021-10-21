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

package profile

import (
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service"
)

// ExtractTraceHandles translates the handles in the replayHandles array based on the mappings.
func ExtractTraceHandles(ctx context.Context, replayHandles *[]int64, replayHandleType string, handleMapping map[uint64][]service.VulkanHandleMappingItem) {
	for i, v := range *replayHandles {
		handles, ok := handleMapping[uint64(v)]
		if !ok {
			log.E(ctx, "%v not found in replay: %v", replayHandleType, v)
			continue
		}

		found := false
		for _, handle := range handles {
			if handle.HandleType == replayHandleType {
				(*replayHandles)[i] = int64(handle.TraceValue)
				found = true
				break
			}
		}

		if !found {
			log.E(ctx, "Incorrect Handle type for %v: %v", replayHandleType, v)
		}
	}
}
