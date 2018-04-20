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

func readString(ctx context.Context, c api.Cmd, s *api.GlobalState, at memory.Pointer, length GLsizei) string {
	ptr := NewCharᵖ(at)
	if length > 0 {
		chars, err := ptr.Slice(0, uint64(length), s.MemoryLayout).Read(ctx, c, s, nil)
		if err != nil {
			return ""
		}
		return string(memory.CharToBytes(chars))
	}
	chars, err := ptr.StringSlice(ctx, s).Read(ctx, c, s, nil)
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(memory.CharToBytes(chars)), "\x00")
}

// Label returns the user maker name.
func (c *GlPushGroupMarkerEXT) Label(ctx context.Context, s *api.GlobalState) string {
	return readString(ctx, c, s, c.Marker(), c.Length())
}

// Label returns the user maker name.
func (c *GlInsertEventMarkerEXT) Label(ctx context.Context, s *api.GlobalState) string {
	return readString(ctx, c, s, c.Marker(), c.Length())
}

// Label returns the user maker name.
func (c *GlPushDebugGroup) Label(ctx context.Context, s *api.GlobalState) string {
	// This is incorrect, fudging for a bug in Unity which has been fixed but not
	// rolled out.
	// See https://github.com/google/gapid/issues/459 for reference.
	//
	// c.Length() should only be treated as null-terminated if c.Length() is < 0.
	//
	// TODO: Consider removing once the fixed version is mainstream.
	return readString(ctx, c, s, c.Message(), c.Length())
}

// Label returns the user maker name.
func (c *GlPushDebugGroupKHR) Label(ctx context.Context, s *api.GlobalState) string {
	// This is incorrect, fudging for a bug in Unity which has been fixed but not
	// rolled out.
	// See https://github.com/google/gapid/issues/459 for reference.
	//
	// c.Length() should only be treated as null-terminated if c.Length() is < 0.
	//
	// TODO: Consider removing once the fixed version is mainstream.
	return readString(ctx, c, s, c.Message(), c.Length())
}
