// Copyright (C) 2019 Google Inc.
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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

// minimizeViewport returns a transform that sets viewport sizes to a 1x1 square
func minimizeViewport(ctx context.Context) transform.Transformer {
	ctx = log.Enter(ctx, "Minimize viewport")
	return transform.Transform("Minimize viewport", func(ctx context.Context,
		id api.CmdID, cmd api.Cmd, out transform.Writer) {

		const width = 1
		const height = 1

		s := out.State()
		l := s.MemoryLayout
		cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}
		switch cmd := cmd.(type) {
		case *VkCmdSetViewport:
			cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())

			viewportCount := uint64(cmd.viewportCount)
			oldViewports := cmd.PViewports().Slice(0, viewportCount, l)
			newViewports := make([]VkViewport, viewportCount)

			for i := uint64(0); i < viewportCount; i++ {
				viewport := oldViewports.Index(i).MustRead(ctx, cmd, s, nil)[0]
				viewport.SetWidth(width)
				viewport.SetHeight(height)
				newViewports[i] = viewport
			}

			newViewportDatas := s.AllocDataOrPanic(ctx, newViewports)
			defer newViewportDatas.Free()

			newCmd := cb.VkCmdSetViewport(cmd.commandBuffer,
				cmd.FirstViewport(),
				uint32(viewportCount),
				newViewportDatas.Ptr()).AddRead(newViewportDatas.Data())

			for _, w := range cmd.Extras().Observations().Writes {
				newCmd.AddWrite(w.Range, w.ID)
			}
			out.MutateAndWrite(ctx, id, newCmd)
		default:
			out.MutateAndWrite(ctx, id, cmd)
		}
	})
}
