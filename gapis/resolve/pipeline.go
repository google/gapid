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

package resolve

import (
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Pipelines resolves the data of the currently bound pipelines at the specified
// point in the capture.
func Pipelines(ctx context.Context, p *path.Pipelines, r *path.ResolveConfig) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent(), r)
	if err != nil {
		return nil, err
	}

	if pipelines, err := pipelinesFor(ctx, obj, p, r); err != api.ErrPipelineNotAvailable {
		return pipelines, err
	}
	return nil, &service.ErrDataUnavailable{Reason: messages.ErrNotADrawCall()}
}

func pipelinesFor(ctx context.Context, o interface{}, p *path.Pipelines, r *path.ResolveConfig) (interface{}, error) {
	switch o := o.(type) {
	case api.APIObject:
		if a := o.API(); a != nil {
			if pp, ok := a.(api.PipelineProvider); ok {
				bound, err := pp.BoundPipeline(ctx, o, p, r)
				if err != nil {
					return nil, err
				}
				return &service.MultiResourceData{
					Resources: map[string]*service.MultiResourceData_ResourceOrError{
						bound.Pipeline.ResourceHandle(): &service.MultiResourceData_ResourceOrError{
							Val: &service.MultiResourceData_ResourceOrError_Resource{
								Resource: bound.Data,
							},
						},
					},
				}, nil
			}
		}

	case *service.CommandTreeNode:
		cmds, err := Cmds(ctx, o.Commands.Capture)
		if err != nil {
			return nil, err
		}

		representation := o.Representation.Indices
		p := o.Commands.Capture.Command(representation[0], representation[1:]...).Pipelines()
		if pl, err := pipelinesFor(ctx, cmds[representation[0]], p, r); err != api.ErrPipelineNotAvailable {
			return pl, err
		}

		if len(o.Commands.From) != len(o.Commands.To) {
			return nil, log.Errf(ctx, nil, "Subcommand indices must be the same length")
		}

		lastSubcommand := len(o.Commands.From) - 1
		for i := 0; i < lastSubcommand; i++ {
			if o.Commands.From[i] != o.Commands.To[i] {
				return nil, log.Errf(ctx, nil, "Subcommand ranges must be identical everywhere but the last element")
			}
		}

		cmd := append([]uint64{}, o.Commands.From...) // make a copy of o.Commands.From
		for i := int64(o.Commands.To[lastSubcommand]); i >= int64(o.Commands.From[lastSubcommand]); i-- {
			cmd[lastSubcommand] = uint64(i)
			p := o.Commands.Capture.Command(cmd[0], cmd[1:]...).Pipelines()
			if pl, err := pipelinesFor(ctx, cmds[o.Commands.From[0]], p, r); err != api.ErrPipelineNotAvailable {
				return pl, err
			}
		}

		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNotADrawCall()}
	}
	return nil, nil
}
