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

package compiler

import "github.com/google/gapid/core/codegen"

func (c *C) buildBufferFuncs() {
	c.buf.init = c.M.Function(c.T.Void, "gapil_init_buffer", c.T.CtxPtr, c.T.BufPtr, c.T.Uint32).
		Inline().
		LinkOnceODR()
	c.Build(c.buf.init, func(s *S) {
		bufPtr, capacity := s.Parameter(1), s.Parameter(2)

		data := c.Alloc(s, capacity, c.T.Uint8)
		bufPtr.Index(0, BufData).Store(data)
		bufPtr.Index(0, BufCap).Store(capacity)
		bufPtr.Index(0, BufSize).Store(s.Scalar(uint32(0)))
	})

	c.buf.term = c.M.Function(c.T.Void, "gapil_term_buffer", c.T.CtxPtr, c.T.BufPtr).
		Inline().
		LinkOnceODR()
	c.Build(c.buf.term, func(s *S) {
		bufPtr := s.Parameter(1)

		bufData := bufPtr.Index(0, BufData).Load()
		c.Free(s, bufData)
	})

	c.buf.append = c.M.Function(c.T.Void, "gapil_append_buffer", c.T.CtxPtr, c.T.BufPtr, c.T.Uint32, c.T.VoidPtr).
		Inline().
		LinkOnceODR()
	c.Build(c.buf.append, func(s *S) {
		bufPtr, size, data := s.Parameter(1), s.Parameter(2), s.Parameter(3)

		bufData := bufPtr.Index(0, BufData).Load()
		bufCap := bufPtr.Index(0, BufCap).Load()
		bufSize := bufPtr.Index(0, BufSize).Load()
		newSize := s.Add(size, bufSize)
		bufPtr.Index(0, BufSize).Store(newSize)
		s.IfElse(s.GreaterThan(newSize, bufCap), func(s *S) {
			newCap := s.Mul(newSize, s.Scalar(uint32(2)))
			newData := c.Realloc(s, bufData, newCap.Cast(c.T.Uint64))
			bufPtr.Index(0, BufCap).Store(newCap)
			bufPtr.Index(0, BufData).Store(newData)
			s.Memcpy(newData.Index(bufSize), data, size)
		}, func(s *S) {
			s.Memcpy(bufData.Index(bufSize), data, size)
		})
	})
}

// InitBuffer initializes a buffer with the initial capacity.
// initialCapacity must be of type uint32.
func (c *C) InitBuffer(s *S, bufPtr, initialCapacity *codegen.Value) {
	s.Call(c.buf.init, s.Ctx, bufPtr, initialCapacity)
}

// TermBuffer frees a buffer's data without freeing the buffer itself.
func (c *C) TermBuffer(s *S, bufPtr *codegen.Value) {
	s.Call(c.buf.term, s.Ctx, bufPtr)
}

// AppendBuffer appends bytes to the buffer.
// size must be of type uint32.
// bytes must be of type void* (alias of uint8*).
func (c *C) AppendBuffer(s *S, bufPtr, size, bytes *codegen.Value) {
	s.Call(c.buf.append, s.Ctx, bufPtr, size, bytes)
}

// AppendBufferData appends a data to the buffer from the given pointer.
func (c *C) AppendBufferData(s *S, bufPtr, ptr *codegen.Value) {
	ptrTy := ptr.Type().(codegen.Pointer)
	size := s.SizeOf(ptrTy.Element)
	c.AppendBuffer(s, bufPtr, size.Cast(c.T.Uint32), ptr.Cast(c.T.VoidPtr))
}
