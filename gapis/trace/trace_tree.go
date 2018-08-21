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

package trace

import (
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/trace/tracer"
)

// TraceTargetTreeNode returns a trace target tree node for the given request
func TraceTargetTreeNode(ctx context.Context, device path.Device, uri string, density float32) (*tracer.TraceTargetTreeNode, error) {
	mgr := GetManager(ctx)
	tracer, ok := mgr.tracers[device.ID.ID()]
	if !ok {
		return nil, log.Errf(ctx, nil, "Could not find tracer for device %d", device.ID.ID())
	}
	return tracer.GetTraceTargetNode(ctx, uri, density)
}

// FindTraceTargets returns trace targets matching the given search parameters.
func FindTraceTargets(ctx context.Context, device path.Device, uri string) ([]*tracer.TraceTargetTreeNode, error) {
	mgr := GetManager(ctx)
	tracer, ok := mgr.tracers[device.ID.ID()]
	if !ok {
		return nil, log.Errf(ctx, nil, "Could not find tracer for device %d", device.ID.ID())
	}
	return tracer.FindTraceTargets(ctx, uri)
}
