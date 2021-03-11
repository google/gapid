// Copyright (C) 2020 Google Inc.
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

package utils

import (
	"context"
	"fmt"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Helper function in the process of grouping GPU slices.
// For a renderPass leafy group, find its parent group and return it.
// If the parent groups are not created yet, create them and store them in map.
func FindParentGroup(ctx context.Context, subOrder, cb uint64, parentLookup map[api.CmdSubmissionKey]*service.ProfilingData_GpuSlices_Group, groups *[]*service.ProfilingData_GpuSlices_Group, links map[api.CmdSubmissionKey][]api.SubCmdIdx, capture *path.Capture) *service.ProfilingData_GpuSlices_Group {
	commandBufferKey := api.CmdSubmissionKey{subOrder, cb, 0, 0}
	if group, ok := parentLookup[commandBufferKey]; ok {
		return group
	} else {
		submissionKey := api.CmdSubmissionKey{subOrder, 0, 0, 0}
		var submissionGroup *service.ProfilingData_GpuSlices_Group
		if g, ok := parentLookup[submissionKey]; ok {
			submissionGroup = g
		} else {
			submissionGroup = &service.ProfilingData_GpuSlices_Group{
				Id:     int32(len(*groups)),
				Name:   fmt.Sprintf("Submission: %v", subOrder),
				Parent: nil,
				Link:   &path.Command{Capture: capture, Indices: links[submissionKey][0]},
			}
			parentLookup[submissionKey] = submissionGroup
			*groups = append(*groups, submissionGroup)
		}

		commandBufferGroup := &service.ProfilingData_GpuSlices_Group{
			Id:     int32(len(*groups)),
			Name:   fmt.Sprintf("Command Buffer: %v", cb),
			Parent: submissionGroup,
			Link:   &path.Command{Capture: capture, Indices: links[commandBufferKey][0]},
		}
		parentLookup[commandBufferKey] = commandBufferGroup
		*groups = append(*groups, commandBufferGroup)
		return commandBufferGroup
	}
}
