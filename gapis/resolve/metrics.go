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

func Metrics(ctx context.Context, p *path.Metrics) (*api.Metrics, error) {
	switch p.Type {
	case path.Metrics_MEMORY_BREAKDOWN:
		return memoryBreakdown(ctx, p)
	default:
		return nil, fmt.Errorf("Metrics type %v not implemented", p.Type)
	}

}

func memoryBreakdown(ctx context.Context, p *path.Metrics) (*api.Metrics, error) {
	cmd, err := Cmd(ctx, p.Command)
	if err != nil {
		return nil, err
	}
	a := cmd.API()
	if a == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrStateUnavailable()}
	}

	state, err := GlobalState(ctx, p.Command.GlobalStateAfter())
	if err != nil {
		return nil, err
	}
	if ml, ok := a.(api.MemoryBreakdownProvider); ok {
		val, err := ml.MemoryBreakdown(state)
		if err != nil {
			return nil, log.Errf(ctx, err, "Failed to get memory layout")
		}
		return &api.Metrics{&api.Metrics_MemoryBreakdown{val}}, nil
	} else {
		return nil, fmt.Errorf("Memory breakdown not supported for API %v", a.Name())
	}
}
