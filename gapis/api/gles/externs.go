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
	"fmt"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	rb "github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
	"github.com/pkg/errors"
)

type externs struct {
	ctx   context.Context // Allowed because the externs struct is only a parameter proxy for a single call
	cmd   api.Cmd
	cmdID api.CmdID
	s     *api.GlobalState
	b     *rb.Builder
	w     api.StateWatcher
}

func (e externs) mapMemory(slice memory.Slice) {
	ctx := e.ctx
	if b := e.b; b != nil {
		switch e.cmd.(type) {
		case *GlMapBufferRange, *GlMapBufferRangeEXT, *GlMapBufferOES, *GlMapBuffer:
			// Base address is on the stack.
			b.MapMemory(memory.Range{Base: slice.Base(), Size: slice.Size()})

		default:
			log.E(ctx, "mapBuffer extern called for unsupported command: %v", e.cmd)
		}
	}
}

func (e externs) unmapMemory(slice memory.Slice) {
	if b := e.b; b != nil {
		b.UnmapMemory(memory.Range{Base: slice.Base(), Size: slice.Size()})
	}
}

func (e externs) GetEGLStaticContextState(EGLDisplay, EGLContext) StaticContextStateʳ {
	return FindStaticContextState(e.s.Arena, e.cmd.Extras()).Clone(e.s.Arena, api.CloneContext{})
}

func (e externs) GetEGLDynamicContextState(EGLDisplay, EGLSurface, EGLContext) DynamicContextStateʳ {
	return FindDynamicContextState(e.s.Arena, e.cmd.Extras()).Clone(e.s.Arena, api.CloneContext{})
}

func (e externs) GetAndroidNativeBufferExtra(Voidᵖ) AndroidNativeBufferExtraʳ {
	return FindAndroidNativeBufferExtra(e.s.Arena, e.cmd.Extras()).Clone(e.s.Arena, api.CloneContext{})
}

func (e externs) GetEGLImageData(id EGLImageKHR, _ GLsizei, _ GLsizei) {
	if d := FindEGLImageData(e.cmd.Extras()); d != nil {
		if GetState(e.s).EGLImages().Contains(id) {
			ei := GetState(e.s).EGLImages().Get(id)
			for _, img := range ei.Images().All() {
				poolID, pool := e.s.Memory.New()
				pool.Write(0, memory.Resource(d.ID, d.Size))
				data := NewU8ˢ(e.s.Arena, 0, 0, d.Size, d.Size, poolID)
				img.SetWidth(GLsizei(d.Width))
				img.SetHeight(GLsizei(d.Height))
				img.SetData(data)
				img.SetDataFormat(d.Format)
				img.SetDataType(d.Type)
				break // TODO: Support image arrays.
			}
		}
	}
}

func (e externs) calcIndexLimits(data U8ˢ, indexSize int) resolve.IndexRange {
	id := data.ResourceID(e.ctx, e.s)
	count := int(data.Size()) / int(indexSize)
	littleEndian := e.s.MemoryLayout.GetEndian() == device.LittleEndian
	limits, err := resolve.IndexLimits(e.ctx, id, count, indexSize, littleEndian)
	if err != nil {
		if errors.Cause(err) == context.Canceled {
			// TODO: Propagate error
			return resolve.IndexRange{}
		}
		panic(fmt.Errorf("Could not calculate index limits: %v", err))
	}
	return *limits
}

func (e externs) IndexLimits(data U8ˢ, indexSize int32) U32Limits {
	limits := e.calcIndexLimits(data, int(indexSize))
	return NewU32Limits(e.s.Arena, limits.First, limits.Count)
}

func (e externs) substr(str string, start, end int32) string {
	return str[start:end]
}

func (e externs) GetCompileShaderExtra(ctx Contextʳ, obj Shaderʳ, bin BinaryExtraʳ) CompileShaderExtraʳ {
	return FindCompileShaderExtra(e.s.Arena, e.cmd.Extras(), obj).Clone(e.s.Arena, api.CloneContext{})
}

func (e externs) GetLinkProgramExtra(ctx Contextʳ, obj Programʳ, bin BinaryExtraʳ) LinkProgramExtraʳ {
	return FindLinkProgramExtra(e.s.Arena, e.cmd.Extras()).Clone(e.s.Arena, api.CloneContext{})
}

func (e externs) GetValidateProgramExtra(ctx Contextʳ, obj Programʳ) ValidateProgramExtraʳ {
	return FindValidateProgramExtra(e.s.Arena, e.cmd.Extras()).Clone(e.s.Arena, api.CloneContext{})
}

func (e externs) GetValidateProgramPipelineExtra(ctx Contextʳ, obj Pipelineʳ) ValidateProgramPipelineExtraʳ {
	return FindValidateProgramPipelineExtra(e.s.Arena, e.cmd.Extras()).Clone(e.s.Arena, api.CloneContext{})
}

func (e externs) onGlError(err GLenum) {
	// Call the state's callback function for API error.
	if f := e.s.OnError; f != nil {
		f(err)
	}
}

func (e externs) newMsg(severity Severity, message *stringtable.Msg) uint32 {
	// Call the state's callback function for message.
	if f := e.s.NewMessage; f != nil {
		return f(severityFromEnum(severity), message)
	}
	return 0
}

// Maps generated Severity to one of the const values defined in log.
func severityFromEnum(enumValue Severity) log.Severity {
	switch enumValue {
	case Severity_SEVERITY_DEBUG:
		return log.Debug
	case Severity_SEVERITY_INFO:
		return log.Info
	case Severity_SEVERITY_WARNING:
		return log.Warning
	case Severity_SEVERITY_ERROR:
		return log.Error
	case Severity_SEVERITY_FATAL:
		return log.Fatal
	default:
		panic(fmt.Errorf("Bad Severity value %v", enumValue))
	}
}

func (e externs) addTag(msgID uint32, message *stringtable.Msg) {
	// Call the state's callback function for message.
	if f := e.s.AddTag; f != nil {
		f(msgID, message)
	}
}

func (e externs) ReadGPUTextureData(texture Textureʳ, level, layer GLint) U8ˢ {
	poolID, dst := e.s.Memory.New()
	img := texture.Levels().Get(level).Layers().Get(layer)
	dataFormat, dataType := img.getUnsizedFormatAndType()
	format, err := getImageFormat(dataFormat, dataType)
	if err != nil {
		panic(err)
	}
	size := format.Size(int(img.Width()), int(img.Height()), 1)
	device := replay.GetDevice(e.ctx)
	if device == nil {
		log.W(e.ctx, "No device bound for GPU texture read")
		return NewU8ˢ(e.s.Arena, 0, 0, uint64(size), uint64(size), poolID)
	}
	dataID, err := database.Store(e.ctx, &ReadGPUTextureDataResolveable{
		Capture:    path.NewCapture(capture.Get(e.ctx).ID.ID()),
		Device:     device,
		After:      uint64(e.cmdID),
		Thread:     e.cmd.Thread(),
		Texture:    uint32(texture.ID()),
		Level:      uint32(level),
		Layer:      uint32(layer),
		DataFormat: uint32(dataFormat),
		DataType:   uint32(dataType),
	})
	if err != nil {
		panic(err)
	}
	data := memory.Resource(dataID, uint64(size))
	dst.Write(0, data)
	return NewU8ˢ(e.s.Arena, 0, 0, uint64(size), uint64(size), poolID)
}
