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

package sync

import (
	"context"
	"math"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
)

// RenderPassKey is a combination of handles used to lookup a submitted render pass commands.
type RenderPassKey struct {
	Submission    int
	CommandBuffer uint64
	RenderPass    uint64
	Framebuffer   uint64
}

// RenderPassLookup maintains a mapping of RenderPassKey to api.SubCmdIdx. It allows for fuzzy
// lookup of command indecies for submitted render passes and command buffers.
type RenderPassLookup struct {
	commandBuffers map[uint64]*commandBufferLookup
	renderPasses   map[uint64]*renderPassLookup
}

// NewRenderPassLookup creates and initilizes a new RenderPassLookup.
func NewRenderPassLookup() *RenderPassLookup {
	return &RenderPassLookup{
		commandBuffers: map[uint64]*commandBufferLookup{},
		renderPasses:   map[uint64]*renderPassLookup{},
	}
}

// AddCommandBuffer adds a submitted command buffer start index to the mapping.
func (l *RenderPassLookup) AddCommandBuffer(ctx context.Context, submission int, commandBuffer uint64, idx api.SubCmdIdx) {
	log.D(ctx, "Adding mapping for command buffer %d %d -> %v", submission, commandBuffer, idx)
	cbl, ok := l.commandBuffers[commandBuffer]
	if !ok {
		cbl = newCommandBufferLookup()
		l.commandBuffers[commandBuffer] = cbl
	}
	cbl.add(submission, idx)
}

// AddRenderPass adds a submitted render pass start index to the mapping.
func (l *RenderPassLookup) AddRenderPass(ctx context.Context, key RenderPassKey, idx api.SubCmdIdx) {
	log.D(ctx, "Adding mapping for render pass %v -> %v", key, idx)
	rpl, ok := l.renderPasses[key.RenderPass]
	if !ok {
		rpl = newRenderPassLookup()
		l.renderPasses[key.RenderPass] = rpl
	}
	rpl.add(key, idx)
}

// Lookup finds the best matching command index for the given key. Specifying zero for any of the
// handles is treated as "unknown" and will cause the lookup to match up with the best known
// index, if it exists. Returned indecies either point to a submitted command buffer or a render
// pass within a submitted command buffer.
func (l *RenderPassLookup) Lookup(ctx context.Context, key RenderPassKey) api.SubCmdIdx {
	if key.RenderPass != 0 {
		if rpl, ok := l.renderPasses[key.RenderPass]; ok {
			return rpl.lookup(key)
		}
	}
	if cbl, ok := l.commandBuffers[key.CommandBuffer]; ok {
		return cbl.lookup(key)
	}
	return nil
}

type commandBufferLookup struct {
	submissions     map[int]api.SubCmdIdx
	firstSubmission int
}

func newCommandBufferLookup() *commandBufferLookup {
	return &commandBufferLookup{
		submissions:     map[int]api.SubCmdIdx{},
		firstSubmission: math.MaxInt,
	}
}

func (l *commandBufferLookup) add(submission int, idx api.SubCmdIdx) {
	if current, ok := l.submissions[submission]; !ok || idx.LessThan(current) {
		l.submissions[submission] = idx
	}
	if submission < l.firstSubmission {
		l.firstSubmission = submission
	}
}

func (l *commandBufferLookup) lookup(key RenderPassKey) api.SubCmdIdx {
	if idx, ok := l.submissions[key.Submission]; ok {
		return idx
	}

	if key.Submission == 0 || len(l.submissions) == 1 {
		return l.submissions[l.firstSubmission]
	}

	// Find the command buffer that matches the submission index the closest.
	var idx api.SubCmdIdx
	distance := math.MaxInt
	for submission, cmd := range l.submissions {
		d := abs(key.Submission - submission)
		if d < distance {
			idx = cmd
			distance = d
		}
	}
	return idx
}

type renderPassLookup struct {
	mappings        map[RenderPassKey]api.SubCmdIdx
	byCommandBuffer map[uint64][]RenderPassKey
	byFramebuffer   map[uint64][]RenderPassKey
}

func newRenderPassLookup() *renderPassLookup {
	return &renderPassLookup{
		mappings:        map[RenderPassKey]api.SubCmdIdx{},
		byCommandBuffer: map[uint64][]RenderPassKey{},
		byFramebuffer:   map[uint64][]RenderPassKey{},
	}
}

func (l *renderPassLookup) add(key RenderPassKey, idx api.SubCmdIdx) {
	key.RenderPass = 0 // clear it out, since we don't care about it anymore.

	if current, ok := l.mappings[key]; !ok || idx.LessThan(current) {
		l.mappings[key] = idx
		// Only add this key to the by* lookups, if we haven't seen it before.
		if !ok {
			l.byCommandBuffer[key.CommandBuffer] = append(l.byCommandBuffer[key.CommandBuffer], key)
			l.byFramebuffer[key.Framebuffer] = append(l.byFramebuffer[key.Framebuffer], key)
		}
	}
}

func (l *renderPassLookup) lookup(key RenderPassKey) api.SubCmdIdx {
	key.RenderPass = 0

	if idx, ok := l.mappings[key]; ok {
		return idx
	}

	if key.CommandBuffer != 0 {
		if list, ok := l.byCommandBuffer[key.CommandBuffer]; ok {
			if len(list) == 1 { // most common case.
				return l.mappings[list[0]]
			}

			// Find entry where the framebuffer matches, or the closest submission.
			distance := math.MaxInt
			var found RenderPassKey
			for i := range list {
				if list[i].Framebuffer == key.Framebuffer {
					return l.mappings[list[i]]
				}
				d := abs(list[i].Submission - key.Submission)
				if d < distance {
					found = list[i]
					distance = d
				}
			}
			return l.mappings[found]
		}
	}

	if key.Framebuffer != 0 { // and key.CommandBuffer == 0 or unknown
		if list, ok := l.byFramebuffer[key.Framebuffer]; ok {
			if len(list) == 1 { // most common case.
				return l.mappings[list[0]]
			}

			// Find the entry with the closest submission index.
			distance := math.MaxInt
			var found RenderPassKey
			for i := range list {
				d := abs(list[i].Submission - key.Submission)
				if d < distance {
					found = list[i]
					distance = d
				}
			}
			return l.mappings[found]
		}
	}

	// Find the closest submission.
	distance := math.MaxInt
	var idx api.SubCmdIdx
	for k, cmd := range l.mappings {
		d := abs(k.Submission - key.Submission)
		if d < distance {
			idx = cmd
			distance = d
		}
	}
	return idx
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
