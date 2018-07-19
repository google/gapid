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

// Package replay is a plugin for the gapil compiler to generate replay streams
// for commands.
package replay

import (
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/semantic"
)

// hasReplayRemapAnnotation returns true if ty is annotated with the
// @replay_remap annotation.
func hasReplayRemapAnnotation(ty semantic.Type) bool {
	a, ok := ty.(semantic.Annotated)
	return ok && a.GetAnnotation("replay_remap") != nil
}

// isRemapped returns true if data of type ty is dynamic (unknown at replay
// build time).
func isRemapped(ty semantic.Type) bool {
	if hasReplayRemapAnnotation(ty) {
		// This type is annotated with @replay_remap, and so has a
		// replay-dynamic value.
		return true
	}
	for ty != nil {
		switch ty := ty.(type) {
		case *semantic.Pointer:
			return true // Pointers are remapped.
		case *semantic.Class:
			for _, f := range ty.Fields {
				if isRemapped(f.Type) {
					return true
				}
			}
		case *semantic.StaticArray:
			return isRemapped(ty.ValueType)
		}
		u := semantic.Underlying(ty)
		if u == ty {
			break
		}
		ty = u
	}
	return false
}

func (r *replayer) getRemapKeyFunc(s *compiler.S, ty semantic.Type) *codegen.Value {
	if !hasReplayRemapAnnotation(ty) {
		return nil
	}
	// TODO: Pre-cache these calls!
	return s.Call(r.callbacks.getRemapFunc, s.Scalar(r.CurrentAPI().Name()), s.Scalar(ty.Name()))
}

func (r *replayer) loadRemap(s *compiler.S, val *codegen.Value, ty semantic.Type,
	pushValue func(s *compiler.S)) {

	if f := r.getRemapKeyFunc(s, ty); f != nil {
		// This type requires remapping.
		// Call out to the user-defined function that produces a unique remap
		// key for the value.
		valPtr := s.LocalInit("remap_val_ptr", val)
		key := s.CallIndirect(f, s.Ctx, valPtr.Cast(r.T.VoidPtr))
		// See if we have already allocated a remapping address for this key.
		addr := s.Call(r.callbacks.lookupRemapping, s.Ctx, s.Ctx.Index(0, data), key)
		s.IfElse(s.Equal(addr, s.Scalar(^uint64(0))), func(s *compiler.S) {
			// First time we've seen this remap key.
			// Allocate memory to hold the replay remapped value.
			addr := s.Call(r.callbacks.allocateMemory, s.Ctx, s.Ctx.Index(0, data),
				s.SizeOf(r.T.Replay(ty)), s.AlignOf(r.T.Replay(ty)))
			// Bind the remapping address to the key so it can be looked up.
			s.Call(r.callbacks.addRemapping, s.Ctx, s.Ctx.Index(0, data), addr, key)
			// Push the value.
			pushValue(s)
			// Clone the value, as it will be used once for the store, but
			// we still want it on the stack when we return.
			r.asmClone(s, s.Scalar(uint32(0)))
			// Store the value into the allocated slot. This is slot and value
			// is now immutable.
			r.asmStore(s, r.asmVolatilePtr(s, addr))
		}, /* else */ func(s *compiler.S) {
			// We've encountered this key before.
			// Load initial remap value.
			r.asmLoad(s, r.asmVolatilePtr(s, addr), r.asmType(ty))
		})
	} else {
		// This type does not require remapping
		pushValue(s)
	}
}

func (r *replayer) storeRemap(s *compiler.S, val *codegen.Value, ty semantic.Type,
	pushValue func(s *compiler.S)) {

	if f := r.getRemapKeyFunc(s, ty); f != nil {
		// This type requires remapping.
		// Call out to the user-defined function that produces a unique remap
		// key for the value.
		valPtr := s.LocalInit("remap_val_ptr", val)
		key := s.CallIndirect(f, s.Ctx, valPtr.Cast(r.T.VoidPtr))
		// See if we have already allocated a remapping address for this key.
		addr := s.Call(r.callbacks.lookupRemapping, s.Ctx, s.Ctx.Index(0, data), key)
		s.IfElse(s.Equal(addr, s.Scalar(^uint64(0))), func(s *compiler.S) {
			// First time we've seen this remap key.
			// Allocate memory to hold the replay remapped value.
			addr := s.Call(r.callbacks.allocateMemory, s.Ctx, s.Ctx.Index(0, data),
				s.SizeOf(r.T.Replay(ty)), s.AlignOf(r.T.Replay(ty)))
			// Bind the remapping address to the key so it can be looked up.
			s.Call(r.callbacks.addRemapping, s.Ctx, s.Ctx.Index(0, data), addr, key)
			// Push the value.
			pushValue(s)
			// Store the value into the allocated slot.
			r.asmStore(s, r.asmVolatilePtr(s, addr))
		}, /* else */ func(s *compiler.S) {
			// We've encountered this key before.
			// Push the value.
			pushValue(s)
			// Store the value into the allocated slot.
			r.asmStore(s, r.asmVolatilePtr(s, addr))
		})
	}
}
