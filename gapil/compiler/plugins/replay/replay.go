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
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/semantic"
)

//#include "gapil/runtime/cc/replay/replay.h"
import "C"

const (
	// GetReplayData is the name of the function that retrieves the replay
	// data from the context.
	GetReplayData = "get_replay_data"

	// Anything very low in application address-space is extremely
	// unlikely to be a valid pointer.
	observableAddressStart = 0x1000

	initialStreamCap = 64 << 10

	data = "replay_data" // Additional context field.

	// Fields of gapil_replay_data:
	stream = "stream"
	call   = "call" // void (*call)(context*)

	// Fields of pointer_fixup:
	offset = "offset"
	addr   = "addr"
)

// replayer is the compiler plugin that adds replay generation functionality.
type replayer struct {
	*compiler.C
	replayLayout  *device.MemoryLayout
	getReplayData *codegen.Function
	T             types
	callbacks     callbacks
}

type types struct {
	*compiler.Types
	ReplayTypes   *compiler.StorageTypes
	callFPtr      codegen.Type
	replayData    *codegen.Struct
	replayDataPtr codegen.Pointer
	asm           asm
}

// Replay returns the codegen type used to represent ty when stored in a
// replay device buffer.
func (t *types) Replay(ty semantic.Type) codegen.Type {
	return t.ReplayTypes.Get(ty)
}

// Plugin is the replay plugin for the gapil compiler.
func Plugin(replayLayout *device.MemoryLayout) compiler.Plugin {
	return &replayer{replayLayout: replayLayout}
}

var (
	_ compiler.ContextDataPlugin      = (*replayer)(nil)
	_ compiler.FunctionExposerPlugin  = (*replayer)(nil)
	_ compiler.OnBeginCommandListener = (*replayer)(nil)
	_ compiler.OnFenceListener        = (*replayer)(nil)
	_ compiler.OnEndCommandListener   = (*replayer)(nil)
	_ compiler.OnReadListener         = (*replayer)(nil)
	_ compiler.OnWriteListener        = (*replayer)(nil)
)

func (r *replayer) OnPreBuildContext(c *compiler.C) {
	callFPtr := c.T.Pointer(c.T.Function(nil, c.T.CtxPtr))

	replayData := c.T.Struct("gapil_replay_data", append(
		c.T.FieldsOf(C.gapil_replay_data{}),
		codegen.Field{Name: call, Type: callFPtr},
	)...)
	replayDataPtr := c.T.Pointer(replayData)

	r.C = c
	r.T = types{
		Types:         &c.T,
		callFPtr:      callFPtr,
		replayData:    replayData,
		replayDataPtr: replayDataPtr,
	}

	if r.replayLayout == nil {
		r.replayLayout = c.Settings.CaptureABI.MemoryLayout
	}

	r.getReplayData = c.M.Function(replayDataPtr, GetReplayData, c.T.CtxPtr)

	r.parseCallbacks()
}

func (r *replayer) Build(c *compiler.C) {
	r.parseAsmTypes()
	r.T.ReplayTypes = r.StorageTypes(r.replayLayout, "R_")
	c.Build(r.getReplayData, func(s *compiler.S) { s.Return(s.Ctx.Index(0, data)) })
}

func (r *replayer) ContextData(c *compiler.C) []compiler.ContextField {
	return []compiler.ContextField{
		{
			Name: data,
			Type: r.T.replayData,
			Init: func(s *compiler.S, dataPtr *codegen.Value) {
				c.InitBuffer(s, dataPtr.Index(0, stream), s.Scalar(uint32(initialStreamCap)))
				s.Call(r.callbacks.initData, s.Ctx, dataPtr)

				// The pointer alignment field is to support identical output
				// with the legacy replay builder logic. This should be removed.
				// See Builder::layout_volatile_memory in builder.cpp.
				pointerAlignment := uint32(r.replayLayout.Pointer.Alignment)
				dataPtr.Index(0, "pointer_alignment").Store(s.Scalar(pointerAlignment))
			},
			Term: func(s *compiler.S, dataPtr *codegen.Value) {
				s.Call(r.callbacks.termData, s.Ctx, dataPtr)
				c.TermBuffer(s, dataPtr.Index(0, stream))
			},
		},
	}
}

func (r *replayer) Functions() map[string]*codegen.Function {
	return map[string]*codegen.Function{
		GetReplayData: r.getReplayData,
	}
}

func (r *replayer) OnBeginCommand(s *compiler.S, cmd *semantic.Function) {
	callFunc := r.buildCall(cmd)
	s.Ctx.Index(0, data, call).Store(s.FuncAddr(callFunc))
	r.emitLabel(s)
}

func (r *replayer) OnFence(s *compiler.S) {
	r.emitCall(s)
}

func (r *replayer) OnEndCommand(s *compiler.S, cmd *semantic.Function) {

}

func (r *replayer) OnRead(s *compiler.S, slice *codegen.Value, ty *semantic.Slice) {
	r.reserve(s, slice, ty)

	if !r.isInferredExpression() {
		// If the expression is inferred, then it is an output of a command.
		// We do not wish to overwrite the replayed command's output with the
		// observed data.

		s.If(isAppPool(slice), func(s *compiler.S) {
			slicePtr := s.LocalInit("sliceptr", slice)
			slice, ns := r.observed(s, slice, ty)
			dst := slice.Extract(compiler.SliceBase).Cast(r.T.Pointer(r.T.Replay(ty.To)))
			r.read(s, dst, slicePtr, ty, ns)
		})
	}
}

func (r *replayer) read(s *compiler.S, dst, src *codegen.Value, ty semantic.Type, ns asmNamespace) {
	switch ty := ty.(type) {
	case *semantic.Slice:
		slicePtr := src
		slice := slicePtr.Load()
		if isRemapped(ty.To) {
			count := slice.Extract(compiler.SliceCount)
			els := r.SliceDataForRead(s, slicePtr, r.T.Capture(ty.To))
			s.ForN(count, func(s *compiler.S, index *codegen.Value) *codegen.Value {
				dst := dst.Index(index)
				src := els.Index(index)
				r.read(s, dst, src, ty.To, ns)
				return nil
			})
		} else {
			resID := r.addResource(s, slice)
			r.asmResource(s, resID, r.asmObservedPtr(s, dst, ns))
		}

	case *semantic.Class:
		// Start by splatting the class in as a resource.
		// TODO: Consider the size of the class.
		//       If it's small, compose the class with stores.
		slice := s.Zero(r.T.Sli).
			Insert(compiler.SliceRoot, dst.Cast(r.T.Uint64)).
			Insert(compiler.SliceBase, dst.Cast(r.T.Uint64)).
			Insert(compiler.SliceSize, s.SizeOf(r.T.Replay(ty))).
			Insert(compiler.SliceCount, s.Scalar(uint64(1)))

		resID := r.addResource(s, slice)
		r.asmResource(s, resID, r.asmObservedPtr(s, dst, ns))

		// Then patch up the fields that need remapping.
		for _, f := range ty.Fields {
			if isRemapped(f.Type) {
				src := src.Index(0, f.Name())
				dst := dst.Index(0, f.Name())
				r.read(s, dst, src, f.Type, ns)
			}
		}

	default:
		val := src.Load()
		r.loadRemap(s, val, ty, func(s *compiler.S) {
			r.asmPush(s, r.value(s, val, ty))
		})
		r.asmStore(s, r.asmObservedPtr(s, dst, ns))
	}
}

func (r *replayer) OnWrite(s *compiler.S, slice *codegen.Value, ty *semantic.Slice) {
	r.reserve(s, slice, ty)

	s.If(isAppPool(slice), func(s *compiler.S) {
		if isRemapped(ty.To) {
			count := slice.Extract(compiler.SliceCount)
			slicePtr := s.LocalInit("slicePtr", slice)
			dst := s.LocalInit("dst", slice.Extract(compiler.SliceBase))
			dstStride := s.Scalar(uint64(r.T.ReplayTypes.StrideOf(ty.To)))
			els := r.SliceDataForRead(s, slicePtr, r.T.Capture(ty.To))
			s.ForN(count, func(s *compiler.S, index *codegen.Value) *codegen.Value {
				if _, isStruct := semantic.Underlying(ty.To).(*semantic.Class); isStruct {
					return nil // TODO
				}
				el, elTy := els.Index(index).Load(), ty.To
				r.storeRemap(s, el, elTy, func(s *compiler.S) {
					addr := r.observedPtr(s, dst.Load(), ty)
					r.asmLoad(s, addr, r.asmType(elTy))
				})
				dst.Store(s.Add(dst.Load(), dstStride))
				return nil
			})
		}
	})
}

func (r *replayer) buildCall(cmd *semantic.Function) *codegen.Function {
	callFunc := r.M.Function(r.T.Void, "call_"+cmd.Name(), r.T.CtxPtr)

	r.C.Build(callFunc, func(s *compiler.S) {
		// Load the command parameters
		r.LoadParameters(s, cmd)

		// Push the arguments
		for _, p := range cmd.CallParameters() {
			val := r.Parameter(s, p)
			r.push(s, val, p.Type)
		}

		api := cmd.Owner().(*semantic.API)

		apiIdx := int(api.Index)
		cmdIdx := api.CommandIndex(cmd)
		if cmdIdx < 0 {
			panic("Command not found in API?!")
		}
		called := false
		if cmd.Signature.Return != semantic.VoidType {
			r.storeRemap(s, r.Parameter(s, cmd.Return), cmd.Signature.Return, func(s *compiler.S) {
				// Call, and push return on stack for remapping
				r.asmCall(s, apiIdx, cmdIdx, true)
				called = true
			})
		}
		if !called {
			// Remapping wasn't needed. We still need to call without pushing.
			r.asmCall(s, apiIdx, cmdIdx, false)
		}
	})

	return callFunc
}

func (r *replayer) emitCall(s *compiler.S) {
	callFunc := s.Ctx.Index(0, data, call).Load()
	s.CallIndirect(callFunc, s.Ctx)
}

func (r *replayer) emitLabel(s *compiler.S) {
	cmdID := s.Ctx.Index(0, compiler.ContextCmdID).Load().Cast(r.T.Uint32)
	r.asmLabel(s, cmdID)
}

func (r *replayer) reserve(s *compiler.S, slice *codegen.Value, ty *semantic.Slice) {
	s.If(isAppPool(slice), func(s *compiler.S) {
		slice, ns := r.observed(s, slice, ty)
		slicePtr := s.LocalInit("sliceptr", slice)
		dataPtr := s.Ctx.Index(0, data)
		alignment := s.AlignOf(r.T.Replay(ty.To)).Cast(r.T.Uint32)
		s.Call(r.callbacks.reserveMemory, s.Ctx, dataPtr, slicePtr, s.Scalar(ns), alignment)
	})
}

func (r *replayer) addResource(s *compiler.S, slice *codegen.Value) *codegen.Value {
	slicePtr := s.LocalInit("sliceptr", slice)
	dataPtr := s.Ctx.Index(0, data)
	return s.Call(r.callbacks.addResource, s.Ctx, dataPtr, slicePtr)
}

func (r *replayer) addConstant(s *compiler.S, val *codegen.Value, ty semantic.Type) *codegen.Value {
	dataPtr := s.Ctx.Index(0, data)
	ptr := s.LocalInit("constant-ptr", val).Cast(r.T.VoidPtr)
	size := s.SizeOf(r.T.Replay(ty)).Cast(r.T.Uint32)
	alignment := s.AlignOf(r.T.Replay(ty)).Cast(r.T.Uint32)
	addr := s.Call(r.callbacks.addConstant, s.Ctx, dataPtr, ptr, size, alignment)
	return r.asmConstantPtr(s, addr)
}

func (r *replayer) push(s *compiler.S, val *codegen.Value, ty semantic.Type) {
	r.loadRemap(s, val, ty, func(s *compiler.S) {
		r.asmPush(s, r.value(s, val, ty))
	})
}

// value returns the gapil_replay_asm_value for the given value and type.
func (r *replayer) value(s *compiler.S, val *codegen.Value, ty semantic.Type) *codegen.Value {
	switch ty := semantic.Underlying(ty).(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.BoolType:
			return r.asmBool(s, val)

		case semantic.IntType, semantic.UintType, semantic.SizeType, semantic.CharType,
			semantic.Int8Type, semantic.Uint8Type,
			semantic.Int16Type, semantic.Uint16Type,
			semantic.Int32Type, semantic.Uint32Type,
			semantic.Int64Type, semantic.Uint64Type:
			return r.asmValue(s, val.Cast(r.T.Uint64), r.asmType(ty))

		case semantic.Float32Type:
			return r.asmValue(s, val.Bitcast(r.T.Uint32).Cast(r.T.Uint64), r.asmType(ty))

		case semantic.Float64Type:
			return r.asmValue(s, val.Bitcast(r.T.Uint64), r.asmType(ty))
		}
	case *semantic.Pointer:
		return s.Select(s.LessThan(val, s.Scalar(observableAddressStart).Cast(val.Type())),
			r.asmAbsolutePtr(s, val),
			r.observedPtr(s, val, ty))
	case *semantic.StaticArray:
		if isRemapped(ty.ValueType) {
			r.Fail("Static array parameters of remapped types currently not supported")
		}
		return r.addConstant(s, val, ty)
	}
	r.Fail("Unhandled type %v", ty.Name())
	return nil
}

// observed returns returns val adjusted to an approriate replay equivalent
// value when used as an observation value, along with the value's namespace.
// Namespaces are used to separate address ranges that might overlap due to ABI
// differences between the capture device and replay device.
// For example, a pointer may be a different size between the capture and replay
// devices. To prevent a wider pointer overlapping neighbouring data when being
// stored into memory, we place these types in a unique namespace.
func (r *replayer) observed(s *compiler.S, val *codegen.Value, ty semantic.Type) (*codegen.Value, asmNamespace) {
	remapAddr := func(addr *codegen.Value, ty semantic.Type) *codegen.Value {
		captureStride := r.T.CaptureTypes.StrideOf(ty)
		replayStride := r.T.ReplayTypes.StrideOf(ty)
		if captureStride != replayStride {
			addrTy := addr.Type()
			addr = addr.Cast(r.T.Uint64)
			idx := s.Div(addr, s.Scalar(captureStride))
			addr = s.Mul(idx, s.Scalar(replayStride))
			addr = addr.Cast(addrTy)
		}
		return addr
	}

	switch ty := ty.(type) {
	case *semantic.Pointer:
		if _, isPtrToPtr := ty.To.(*semantic.Pointer); isPtrToPtr {
			return remapAddr(val, ty.To), 1
		}
	case *semantic.Slice:
		if _, isPtr := ty.To.(*semantic.Pointer); isPtr {
			root := remapAddr(val.Extract(compiler.SliceRoot), ty.To)
			base := remapAddr(val.Extract(compiler.SliceBase), ty.To)
			size := s.Mul(val.Extract(compiler.SliceCount), s.Scalar(r.T.ReplayTypes.StrideOf(ty.To)))
			val = val.
				Insert(compiler.SliceRoot, root).
				Insert(compiler.SliceBase, base).
				Insert(compiler.SliceSize, size)
			return val, 1
		}
	}
	return val, 0
}

func (r *replayer) observedPtr(s *compiler.S, addr *codegen.Value, ty semantic.Type) *codegen.Value {
	addr, ns := r.observed(s, addr, ty)
	return r.asmObservedPtr(s, addr, ns)
}

func isAppPool(slice *codegen.Value) *codegen.Value {
	return slice.Extract(compiler.SlicePool).IsNull()
}

func (r *replayer) isInferredExpression() bool {
	for _, e := range r.ExpressionStack() {
		if _, ok := e.(*semantic.Unknown); ok {
			return true
		}
	}
	return false
}
