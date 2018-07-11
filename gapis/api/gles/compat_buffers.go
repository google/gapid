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

package gles

import (
	"context"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
)

// bufferCompat provides compatibility transformations for GL buffers.
type bufferCompat struct {
	// uniformBufferAlignment is the target's minimum alignment for uniform
	// buffers.
	uniformBufferAlignment int
	// unaligned is a map of compat generated aligned buffer IDs to their
	// original buffer.
	unaligned map[BufferId]Bufferʳ
	// scratch holds the temporary buffers created by the bufferCompat.
	scratch map[scratchBufferKey]scratchBuffer
	// nextBufferID is the buffer identifer to use for the next created scratch
	// buffer.
	nextBufferID BufferId
}

func newBufferCompat(uniformBufferAlignment int) *bufferCompat {
	return &bufferCompat{
		uniformBufferAlignment: uniformBufferAlignment,
		unaligned:              map[BufferId]Bufferʳ{},
		scratch:                map[scratchBufferKey]scratchBuffer{},
		nextBufferID:           BufferId(0xffff0000),
	}
}

// scratchBufferKey is the key to the bufferCompat's scratch-buffer map.
type scratchBufferKey struct {
	c      Contextʳ // The current GL context.
	Target GLenum   // The buffer target.
	Index  GLuint   // The buffer binding index.
}

// scratchBufferKey is the value to the bufferCompat's scratch-buffer map.
type scratchBuffer struct {
	size GLsizeiptr // Size of the buffer.
	id   BufferId   // Identifier of the buffer.
}

// modifyBufferData deals with the complexities of copying unaligned buffer data
// to their aligned copies and should be called when ever a buffer is to be
// modified. modify is called to apply the buffer modification.
func (m *bufferCompat) modifyBufferData(ctx context.Context, out transform.Writer, cb CommandBuilder, c Contextʳ, id api.CmdID, target GLenum, modify func()) {
	id = id.Derived()
	s := out.State()

	// Get the target buffer.
	buf, err := subGetBoundBuffer(ctx, nil, api.CmdNoID, nil, s, GetState(s), cb.Thread, nil, nil, target)
	if buf.IsNil() || err != nil {
		// Unknown buffer
		modify()
		return
	}

	// Lookup the original (unaligned) buffer.
	unaligned, ok := m.unaligned[buf.ID()]
	if !ok {
		// Buffer was not unaligned
		modify()
		return
	}

	// Walk the current bindings looking for those referencing the aligned
	// buffer.
	type binding struct {
		index  GLuint
		offset GLintptr
		size   GLsizeiptr
	}

	rebind := []binding{}
	for i, b := range c.Bound().UniformBuffers().All() {
		if b.Binding().IsNil() || m.unaligned[b.Binding().ID()] != unaligned {
			continue
		}

		rebind = append(rebind, binding{
			index:  GLuint(i),
			offset: b.Start(),
			size:   b.Size(),
		})
	}

	if len(rebind) == 0 {
		// No unaligned buffers require copying.
		modify()
		return
	}

	// Bind the original unaligned buffer.
	out.MutateAndWrite(ctx, id, cb.GlBindBuffer(target, unaligned.ID()))

	// Apply the modification.
	modify()

	// Rebind all the unaligned bindings.
	for _, r := range rebind {
		cmd := cb.GlBindBufferRange(target, r.index, buf.ID(), r.offset, r.size)
		m.bindBufferRange(ctx, out, cb, c, id, cmd)
	}
}

// bindBufferRange provides compatibiliy for glBindBufferRange by handling
// buffers that do not meet their minimum alignment on the target device.
// If a buffer is unaligned, then a new buffer is created and the data range is
// copied to this new buffer, and the new buffer is bound.
func (m *bufferCompat) bindBufferRange(ctx context.Context, out transform.Writer, cb CommandBuilder, c Contextʳ, id api.CmdID, cmd *GlBindBufferRange) {
	misalignment := cmd.Offset() % GLintptr(m.uniformBufferAlignment)

	if cmd.Target() != GLenum_GL_UNIFORM_BUFFER || misalignment == 0 {
		out.MutateAndWrite(ctx, id, cmd)
		return
	}

	dID := id.Derived()

	// We have a glBindBufferRange() taking a uniform buffer with an illegal
	// offset alignment.

	orig := c.Objects().Buffers().Get(cmd.Buffer())
	if orig.IsNil() {
		return // Don't know what buffer this is referring to.
	}

	// We need a scratch buffer to copy the buffer data to a correct
	// alignment.
	scratchKey := scratchBufferKey{c, cmd.Target(), cmd.Index()}

	// Look for pre-existing buffer we can reuse.
	buffer, ok := m.scratch[scratchKey]
	if !ok {
		buffer.id = m.newBuffer(ctx, dID, cb, out)
		m.scratch[scratchKey] = buffer
	}

	// Bind the scratch buffer to GL_COPY_WRITE_BUFFER
	origCopyWriteBuffer := c.Bound().CopyWriteBuffer()
	out.MutateAndWrite(ctx, dID, cb.GlBindBuffer(GLenum_GL_COPY_WRITE_BUFFER, buffer.id))

	if buffer.size < cmd.Size() {
		// Resize the scratch buffer
		out.MutateAndWrite(ctx, dID, cb.GlBufferData(GLenum_GL_COPY_WRITE_BUFFER, cmd.Size(), memory.Nullptr, GLenum_GL_DYNAMIC_COPY))
		buffer.size = cmd.Size()
		m.scratch[scratchKey] = buffer
	}

	// Copy out the unaligned data to the scratch buffer in the
	// GL_COPY_WRITE_BUFFER binding.
	out.MutateAndWrite(ctx, dID, cb.GlBindBuffer(cmd.Target(), cmd.Buffer()))
	out.MutateAndWrite(ctx, dID, cb.GlCopyBufferSubData(cmd.Target(), GLenum_GL_COPY_WRITE_BUFFER, cmd.Offset(), 0, cmd.Size()))

	// We can now bind the range with correct alignment.
	out.MutateAndWrite(ctx, id, cb.GlBindBufferRange(cmd.Target(), cmd.Index(), buffer.id, 0, cmd.Size()))

	// Restore old GL_COPY_WRITE_BUFFER binding.
	out.MutateAndWrite(ctx, dID, cb.GlBindBuffer(GLenum_GL_COPY_WRITE_BUFFER, origCopyWriteBuffer.GetID()))

	m.unaligned[buffer.id] = orig
}

func (m *bufferCompat) newBuffer(ctx context.Context, id api.CmdID, cb CommandBuilder, out transform.Writer) BufferId {
	s := out.State()
	bufID := m.nextBufferID
	tmp := s.AllocDataOrPanic(ctx, bufID)
	out.MutateAndWrite(ctx, id, cb.GlGenBuffers(1, tmp.Ptr()).AddWrite(tmp.Data()))
	m.nextBufferID--
	return bufID
}
