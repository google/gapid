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

package vulkan

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
)

// drawCallPipeline returns the bound pipeline for dc at p.
func drawCallPipeline(ctx context.Context, dc *VkQueueSubmit, p *path.Pipelines, r *path.ResolveConfig) (api.BoundPipeline, error) {
	bound := api.BoundPipeline{}
	cmdPath := path.FindCommand(p)
	if cmdPath == nil {
		log.W(ctx, "Couldn't find command at path '%v'", p)
		return bound, api.ErrPipelineNotAvailable
	}

	cmd, err := resolve.Cmd(ctx, cmdPath, r)
	if err != nil {
		return bound, err
	}

	if !cmd.CmdFlags().IsExecutedDraw() && !cmd.CmdFlags().IsExecutedDispatch() {
		return bound, api.ErrPipelineNotAvailable
	}

	s, err := resolve.GlobalState(ctx, cmdPath.GlobalStateAfter(), r)
	if err != nil {
		return bound, err
	}

	c := getStateObject(s)

	lastQueue := c.LastBoundQueue()
	if lastQueue.IsNil() {
		return bound, fmt.Errorf("No previous queue submission")
	}

	if cmd.CmdFlags().IsExecutedDraw() {
		lastDrawInfo, ok := c.LastDrawInfos().Lookup(lastQueue.VulkanHandle())
		if !ok {
			return bound, fmt.Errorf("There have been no previous draws")
		}
		bound.Pipeline = lastDrawInfo.GraphicsPipeline()
	} else {
		lastComputeInfo, ok := c.LastComputeInfos().Lookup(lastQueue.VulkanHandle())
		if !ok {
			return bound, fmt.Errorf("There have been no previous dispatches")
		}
		bound.Pipeline = lastComputeInfo.ComputePipeline()
	}

	if bound.Pipeline == nil {
		return bound, api.ErrPipelineNotAvailable
	}

	bound.Data, err = bound.Pipeline.ResourceData(ctx, s, cmdPath, r)
	return bound, err
}
