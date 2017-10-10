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
	"strings"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
)

func readString(ϟctx context.Context, ϟa api.Cmd, ϟs *api.GlobalState, at memory.Pointer, length GLsizei) string {
	ptr := NewCharᵖ(at)
	if length > 0 {
		chars, err := ptr.Slice(0, uint64(length), ϟs.MemoryLayout).Read(ϟctx, ϟa, ϟs, nil)
		if err != nil {
			return ""
		}
		return string(memory.CharToBytes(chars))
	}
	chars, err := ptr.StringSlice(ϟctx, ϟs).Read(ϟctx, ϟa, ϟs, nil)
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(memory.CharToBytes(chars)), "\x00")
}

// Label returns the user maker name.
func (ϟa *GlPushGroupMarkerEXT) Label(ϟctx context.Context, ϟs *api.GlobalState) string {
	return readString(ϟctx, ϟa, ϟs, ϟa.Marker, ϟa.Length)
}

// Label returns the user maker name.
func (ϟa *GlInsertEventMarkerEXT) Label(ϟctx context.Context, ϟs *api.GlobalState) string {
	return readString(ϟctx, ϟa, ϟs, ϟa.Marker, ϟa.Length)
}

// Label returns the user maker name.
func (ϟa *GlPushDebugGroup) Label(ϟctx context.Context, ϟs *api.GlobalState) string {
	// This is incorrect, fudging for a bug in Unity which has been fixed but not
	// rolled out.
	// See https://github.com/google/gapid/issues/459 for reference.
	//
	// ϟa.Length should only be treated as null-terminated if ϟa.Length is < 0.
	//
	// TODO: Consider removing once the fixed version is mainstream.
	return readString(ϟctx, ϟa, ϟs, ϟa.Message, ϟa.Length)
}

// Label returns the user maker name.
func (ϟa *GlPushDebugGroupKHR) Label(ϟctx context.Context, ϟs *api.GlobalState) string {
	// This is incorrect, fudging for a bug in Unity which has been fixed but not
	// rolled out.
	// See https://github.com/google/gapid/issues/459 for reference.
	//
	// ϟa.Length should only be treated as null-terminated if ϟa.Length is < 0.
	//
	// TODO: Consider removing once the fixed version is mainstream.
	return readString(ϟctx, ϟa, ϟs, ϟa.Message, ϟa.Length)
}
