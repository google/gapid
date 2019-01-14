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
func GetTimestamps(ctx context.Context, capturePath *path.Capture, device *path.Device) (*service.GetTimestampsResponse, error) {
	c, err := capture.ResolveGraphicsFromPath(ctx, capturePath)
	if err != nil {
		return nil, err
	}

	ts := []Timestamp{}
	if device != nil {
		intent := Intent{
			Capture: capturePath,
			Device:  device,
		}

		mgr := GetManager(ctx)
		hints := &service.UsageHints{Background: true}
		for _, a := range c.APIs {
			if qi, ok := a.(QueryTimestamps); ok {
				ts, err = qi.QueryTimestamps(ctx, intent, mgr, hints)
				if err != nil {
					log.E(ctx, "Query timestamps failed.")
					continue
				}
			}
		}
	}

	if err != nil {
		return nil, err
	}

	var timestamps service.Timestamps
	for _, t := range ts {
		item := &service.TimestampsItem{
			Begin:             t.Begin,
			End:               t.End,
			TimeInNanoseconds: uint64(t.Time),
		}
		timestamps.Timestamps = append(timestamps.Timestamps, item)
	}

	res := service.GetTimestampsResponse{
		Res: &service.GetTimestampsResponse_Timestamps{
			Timestamps: &timestamps,
		},
	}

	return &res, nil
}
