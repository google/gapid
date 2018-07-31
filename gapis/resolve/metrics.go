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

package resolve

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

func Metrics(ctx context.Context, p *path.Metrics, r *path.ResolveConfig) (*api.Metrics, error) {
	res := api.Metrics{}
	if p.MemoryBreakdown {
		breakdown, err := memoryBreakdown(ctx, p.Command, r)
		if err != nil {
			return nil, log.Errf(ctx, err, "Failed to get memory breakdown")
		}
		res.MemoryBreakdown = breakdown
	}
	return &res, nil
}

func memoryBreakdown(ctx context.Context, c *path.Command, r *path.ResolveConfig) (*api.MemoryBreakdown, error) {
	cmd, err := Cmd(ctx, c, r)
	if err != nil {
		return nil, err
	}
	a := cmd.API()
	if a == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrStateUnavailable()}
	}

	state, err := GlobalState(ctx, c.GlobalStateAfter(), r)
	if err != nil {
		return nil, err
	}
	if ml, ok := a.(api.MemoryBreakdownProvider); ok {
		return ml.MemoryBreakdown(state)
	}
	return nil, fmt.Errorf("Memory breakdown not supported for API %v", a.Name())

}
