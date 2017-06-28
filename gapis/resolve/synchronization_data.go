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

package resolve

import (
	"context"

	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/gfxapi/sync"
)

// Resolve builds a SynchronizationResolvable object for the given capture
func (r *SynchronizationResolvable) Resolve(ctx context.Context) (interface{}, error) {
	capture, err := capture.ResolveFromPath(ctx, r.Capture)
	if err != nil {
		return nil, err
	}
	s := sync.NewData()

	for _, api := range capture.APIs {
		if sync, ok := api.(sync.SynchronizedAPI); ok {
			if err = sync.ResolveSynchronization(ctx, s, r.Capture); err != nil {
				return nil, err
			}
		}
	}

	return s, nil
}
