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
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
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
}

func (e externs) mapMemory(slice memory.Slice) {
	ctx := e.ctx
	if b := e.b; b != nil {
		switch e.cmd.(type) {
		case *GlMapBufferRange, *GlMapBufferRangeEXT, *GlMapBufferOES, *GlMapBuffer:
			// Base address is on the stack.
			b.MapMemory(slice.Range(e.s.MemoryLayout))

		default:
			log.E(ctx, "mapBuffer extern called for unsupported command: %v", e.cmd)
		}
	}
}

func (e externs) unmapMemory(slice memory.Slice) {
	if b := e.b; b != nil {
		b.UnmapMemory(slice.Range(e.s.MemoryLayout))
	}
}

func (e externs) GetEGLStaticContextState(EGLDisplay, EGLContext) *StaticContextState {
	return FindStaticContextState(e.cmd.Extras())
}

func (e externs) GetEGLDynamicContextState(EGLDisplay, EGLSurface, EGLContext) *DynamicContextState {
	return FindDynamicContextState(e.cmd.Extras())
}

func (e externs) GetAndroidNativeBufferExtra(Voidᵖ) *AndroidNativeBufferExtra {
	return FindAndroidNativeBufferExtra(e.cmd.Extras())
}

func (e externs) GetEGLImageData(id EGLImageKHR, _ GLsizei, _ GLsizei) {
	if d := FindEGLImageData(e.cmd.Extras()); d != nil {
		if GetState(e.s).EGLImages.Contains(id) {
			ei := GetState(e.s).EGLImages.Get(id)
			if img := ei.Image; img != nil {
				poolID, pool := e.s.Memory.New()
				pool.Write(0, memory.Resource(d.ID, d.Size))
				data := U8ˢ{pool: poolID, count: d.Size}
				img.Width = GLsizei(d.Width)
				img.Height = GLsizei(d.Height)
				img.Data = data
				img.DataFormat = d.Format
				img.DataType = d.Type
			}
		}
	}
}

func (e externs) calcIndexLimits(data U8ˢ, indexSize int) resolve.IndexRange {
	id := data.ResourceID(e.ctx, e.s)
	count := int(data.count) / int(indexSize)
	littleEndian := e.s.MemoryLayout.GetEndian() == device.LittleEndian
	limits, err := resolve.IndexLimits(e.ctx, id, count, indexSize, littleEndian)
	if err != nil {
		if errors.Cause(err) == context.Canceled {
			// TODO: Propagate error
			return resolve.IndexRange{}
		} else {
			panic(fmt.Errorf("Could not calculate index limits: %v", err))
		}
	}
	return *limits
}

func (e externs) IndexLimits(data U8ˢ, indexSize int32) U32Limits {
	limits := e.calcIndexLimits(data, int(indexSize))
	return U32Limits{First: limits.First, Count: limits.Count}
}

func (e externs) substr(str string, start, end int32) string {
	return str[start:end]
}

func (e externs) GetProgramInfoExtra(ctx *Context, pid ProgramId) *ProgramInfo {
	return FindProgramInfo(e.cmd.Extras())
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

func (e externs) ReadGPUTextureData(texture *Texture, level, layer GLint) U8ˢ {
	poolID, dst := e.s.Memory.New()
	registry := bind.GetRegistry(e.ctx)
	img := texture.Levels.Get(level).Layers.Get(layer)
	dataFormat, dataType := img.getUnsizedFormatAndType()
	format, err := getImageFormat(dataFormat, dataType)
	if err != nil {
		panic(err)
	}
	size := format.Size(int(img.Width), int(img.Height), 1)
	device := registry.DefaultDevice() // TODO: Device selection.
	if device == nil {
		log.W(e.ctx, "No device found for GPU texture read")
		return U8ˢ{count: uint64(size), pool: poolID}
	}
	dataID, err := database.Store(e.ctx, &ReadGPUTextureDataResolveable{
		Capture:    path.NewCapture(capture.Get(e.ctx).Id.ID()),
		Device:     path.NewDevice(device.Instance().Id.ID()),
		After:      uint64(e.cmdID),
		Thread:     e.cmd.Thread(),
		Texture:    uint32(texture.ID),
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
	return U8ˢ{count: uint64(size), pool: poolID}
}
