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

	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	rb "github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/stringtable"
	"github.com/pkg/errors"
)

type externs struct {
	ctx log.Context // Allowed because the externs struct is only a parameter proxy for a single call
	a   atom.Atom
	s   *gfxapi.State
	b   *rb.Builder
}

func (e externs) mapMemory(slice slice) {
	ctx := e.ctx
	if b := e.b; b != nil {
		switch e.a.(type) {
		case *GlMapBufferRange, *GlMapBufferRangeEXT, *GlMapBufferOES, *GlMapBuffer:
			// Base address is on the stack.
			b.MapMemory(slice.Range(e.s))

		default:
			ctx.Error().V("atom", e.a).Log("mapBuffer extern called for unsupported atom")
		}
	}
}

func (e externs) unmapMemory(slice slice) {
	if b := e.b; b != nil {
		b.UnmapMemory(slice.Range(e.s))
	}
}

func (e externs) GetEGLStaticContextState(EGLDisplay, EGLSurface, EGLContext) *StaticContextState {
	return FindStaticContextState(e.a.Extras())
}

func (e externs) GetEGLDynamicContextState(EGLDisplay, EGLSurface, EGLContext) *DynamicContextState {
	return FindDynamicContextState(e.a.Extras())
}

func (e externs) GetAndroidNativeBufferExtra(Voidᵖ) *AndroidNativeBufferExtra {
	return FindAndroidNativeBufferExtra(e.a.Extras())
}

func (e externs) elSize(ty GLenum) uint64 {
	return uint64(DataTypeSize(ty))
}

func (e externs) calcIndexLimits(data U8ᵖ, ty GLenum, offset, count uint32) resolve.MinMax {
	elSize := e.elSize(ty)
	id := data.Slice(uint64(offset), uint64(offset)+uint64(count)*elSize, e.s).ResourceID(e.ctx, e.s)
	littleEndian := e.s.MemoryLayout.GetEndian() == device.LittleEndian
	limits, err := resolve.IndexLimits(e.ctx, id, int(count), int(elSize), littleEndian)
	if err != nil {
		if errors.Cause(err) == context.Canceled {
			// TODO: Propagate error
			return resolve.MinMax{}
		} else {
			panic(fmt.Errorf("Could not calculate index limits: %v", err))
		}
	}
	return *limits
}

func (e externs) IndexLimits(data U8ᵖ, ty GLenum, offset, count uint32) u32Limits {
	limits := e.calcIndexLimits(data, ty, offset, count)
	return u32Limits{First: limits.Min, Last: limits.Max}
}

func (e externs) substr(str string, start, end int32) string {
	return str[start:end]
}

func (e externs) GetProgramInfoExtra(pid ProgramId) *ProgramInfo {
	return FindProgramInfo(e.a.Extras())
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
func severityFromEnum(enumValue Severity) severity.Level {
	switch enumValue {
	case Severity_SEVERITY_EMERGENCY:
		return severity.Emergency
	case Severity_SEVERITY_ALERT:
		return severity.Alert
	case Severity_SEVERITY_CRITICAL:
		return severity.Critical
	case Severity_SEVERITY_ERROR:
		return severity.Error
	case Severity_SEVERITY_WARNING:
		return severity.Warning
	case Severity_SEVERITY_NOTICE:
		return severity.Notice
	case Severity_SEVERITY_INFO:
		return severity.Info
	case Severity_SEVERITY_DEBUG:
		return severity.Debug
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
