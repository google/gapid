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
	"strings"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
)

var _ = (capture.AtomsImportHandler)(api{})

func (api) TransformAtomStream(ctx log.Context, atoms []atom.Atom) ([]atom.Atom, error) {
	out := make([]atom.Atom, 0, len(atoms))
	for i, a := range atoms {
		switch a := a.(type) {
		case *ContextInfo: // DEPRECATED
			// ContextInfo atoms have been deprecated for ContextState extras.
			// Convert each ContextInfo atom to a ContextState extra and add it
			// to the preceeding atom.
			if i > 0 {
				if eglMakeCurrent, ok := atoms[i-1].(*EglMakeCurrent); ok {
					scs, dcs := ContextInfoToContextState(ctx, a)
					eglMakeCurrent.Extras().Add(scs, dcs)
				}
			}
		default:
			out = append(out, a)
		}
	}
	return out, nil
}

// ContextInfoToContextState is a backwards-compatibility function that converts
// the deprecated ContextInfo atom into a StaticContextState and
// DynamicContextState extra.
func ContextInfoToContextState(ctx log.Context, info *ContextInfo) (scs *StaticContextState, dcs *DynamicContextState) {
	scs = &StaticContextState{
		Constants: Constants{
			// Constants which we need, but which are not fetched in GLES2 (they error)
			MajorVersion:             2,
			MinorVersion:             0,
			MaxDrawBuffers:           1,
			CompressedTextureFormats: U32ːGLenumᵐ{},
			ProgramBinaryFormats:     U32ːGLenumᵐ{},
			ShaderBinaryFormats:      U32ːGLenumᵐ{},
			Extensions:               map[uint32]string{},
		},
	}

	dcs = &DynamicContextState{
		BackbufferWidth:       info.BackbufferWidth,
		BackbufferHeight:      info.BackbufferHeight,
		BackbufferColorFmt:    info.BackbufferColorFmt,
		BackbufferDepthFmt:    info.BackbufferDepthFmt,
		BackbufferStencilFmt:  info.BackbufferStencilFmt,
		ResetViewportScissor:  info.ResetViewportScissor,
		PreserveBuffersOnSwap: info.PreserveBuffersOnSwap,
	}

	// This function is called before we have a capture ID, so state.New() would fail.
	// nil allocator so we can fail and reconsider if we actually do need it here.
	s := gfxapi.NewStateWithNilAllocator()
	c := uint64(info.ConstantCount)
	info.Extras().Observations().ApplyReads(s.Memory[memory.ApplicationPool])
	names := info.ConstantNames.Slice(0, c, s).Read(ctx, info, s, nil)
	offsets := info.ConstantOffsets.Slice(0, c, s).Read(ctx, info, s, nil)
	sizes := info.ConstantSizes.Slice(0, c, s).Read(ctx, info, s, nil)

	for i := uint64(0); i < c; i++ {
		name, offset, size := names[i], uint64(offsets[i]), uint64(sizes[i])
		data := info.ConstantData.Slice(offset, offset+size, s)
		switch name {
		case GLenum_GL_EXTENSIONS:
			// Performance optimization since the API version is slow.
			list := string(gfxapi.CharToBytes(AsCharˢ(data, s).Read(ctx, info, s, nil)))
			for _, e := range strings.Split(list, " ") {
				if len(e) > 0 {
					scs.Constants.Extensions[uint32(len(scs.Constants.Extensions))] = e
				}
			}
		default:
			subSetConstant(ctx, info, info.Extras().Observations(), s, GetState(s), nil, scs, name, data)
		}
	}

	return
}
