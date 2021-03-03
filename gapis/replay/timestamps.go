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

package replay

import (
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// GetTimestamps replays the trace and return the start and end timestamps for each commandbuffers
func GetTimestamps(ctx context.Context, capturePath *path.Capture, device *path.Device, handler service.TimeStampsHandler) error {
	c, err := capture.ResolveGraphicsFromPath(ctx, capturePath)
	if err != nil {
		return err
	}

	if device != nil {
		intent := Intent{
			Capture: capturePath,
			Device:  device,
		}

		mgr := GetManager(ctx)
		hints := &path.UsageHints{Background: true}
		for _, a := range c.APIs {
			if qi, ok := a.(QueryTimestamps); ok {
				err = qi.QueryTimestamps(ctx, intent, mgr, handler, hints)
				if err != nil {
					log.E(ctx, "Query timestamps failed.")
					continue
				}
			}
		}
	}

	return err
}
