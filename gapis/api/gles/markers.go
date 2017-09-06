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

// Label returns the user maker name.
func (ϟa *GlPushGroupMarkerEXT) Label(ϟctx context.Context, ϟs *api.GlobalState) string {
	ptr := Charᵖ(ϟa.Marker)
	if ϟa.Length > 0 {
		return string(memory.CharToBytes(ptr.Slice(0, uint64(ϟa.Length), ϟs.MemoryLayout).Read(ϟctx, ϟa, ϟs, nil)))
	}
	return strings.TrimRight(string(memory.CharToBytes(ptr.StringSlice(ϟctx, ϟs).Read(ϟctx, ϟa, ϟs, nil))), "\x00")
}

// Label returns the user maker name.
func (ϟa *GlInsertEventMarkerEXT) Label(ϟctx context.Context, ϟs *api.GlobalState) string {
	ptr := Charᵖ(ϟa.Marker)
	if ϟa.Length > 0 {
		return string(memory.CharToBytes(ptr.Slice(0, uint64(ϟa.Length), ϟs.MemoryLayout).Read(ϟctx, ϟa, ϟs, nil)))
	}
	return strings.TrimRight(string(memory.CharToBytes(ptr.StringSlice(ϟctx, ϟs).Read(ϟctx, ϟa, ϟs, nil))), "\x00")
}

// Label returns the user maker name.
func (ϟa *GlPushDebugGroup) Label(ϟctx context.Context, ϟs *api.GlobalState) string {
	ptr := Charᵖ(ϟa.Message)
	// This is incorrect, fudging for a bug in Unity which has been fixed but not
	// rolled out.
	// See https://github.com/google/gapid/issues/459 for reference.
	// TODO: Consider removing once the fixed version is mainstream.
	// if ϟa.Length >= 0 {
	if ϟa.Length > 0 {
		return string(memory.CharToBytes(ptr.Slice(0, uint64(ϟa.Length), ϟs.MemoryLayout).Read(ϟctx, ϟa, ϟs, nil)))
	}
	return strings.TrimRight(string(memory.CharToBytes(ptr.StringSlice(ϟctx, ϟs).Read(ϟctx, ϟa, ϟs, nil))), "\x00")
}

// Label returns the user maker name.
func (ϟa *GlPushDebugGroupKHR) Label(ϟctx context.Context, ϟs *api.GlobalState) string {
	ptr := Charᵖ(ϟa.Message)
	// This is incorrect, fudging for a bug in Unity which has been fixed but not
	// rolled out.
	// See https://github.com/google/gapid/issues/459 for reference.
	// TODO: Consider removing once the fixed version is mainstream.
	// if ϟa.Length >= 0 {
	if ϟa.Length > 0 {
		return string(memory.CharToBytes(ptr.Slice(0, uint64(ϟa.Length), ϟs.MemoryLayout).Read(ϟctx, ϟa, ϟs, nil)))
	}
	return strings.TrimRight(string(memory.CharToBytes(ptr.StringSlice(ϟctx, ϟs).Read(ϟctx, ϟa, ϟs, nil))), "\x00")
}
