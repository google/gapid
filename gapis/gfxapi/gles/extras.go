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

	"github.com/google/gapid/core/data/deep"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/atom_pb"
	"github.com/google/gapid/gapis/gfxapi/gles/gles_pb"
)

// ErrorState is an atom extra used to describe the GLES error state after
// the atom has been executed. It is optional - we use it only for testing.
type ErrorState struct {
	TraceDriversGlError GLenum
	InterceptorsGlError GLenum
}

func (s *ErrorState) Convert(ctx context.Context, out atom_pb.Handler) error {
	return out(ctx, &gles_pb.ErrorState{
		TraceDriversGlError: uint32(s.TraceDriversGlError),
		InterceptorsGlError: uint32(s.InterceptorsGlError),
	})
}

// FindProgramInfo searches for the ProgramInfo in the extras, returning the
// ProgramInfo if found, otherwise nil.
func FindProgramInfo(extras *atom.Extras) *ProgramInfo {
	for _, e := range extras.All() {
		if pi, ok := e.(*ProgramInfo); ok {
			clone, err := deep.Clone(pi)
			if err != nil {
				panic(err)
			}
			return clone.(*ProgramInfo)
		}
	}
	return nil
}

// FindErrorState searches for the ErrorState in the extras, returning the
// ErrorState if found, otherwise nil.
func FindErrorState(extras *atom.Extras) *ErrorState {
	for _, e := range extras.All() {
		if pi, ok := e.(*ErrorState); ok {
			return pi
		}
	}
	return nil
}

// FindStaticContextState searches for the StaticContextState in the extras,
// returning the StaticContextState if found, otherwise nil.
func FindStaticContextState(extras *atom.Extras) *StaticContextState {
	for _, e := range extras.All() {
		if cs, ok := e.(*StaticContextState); ok {
			clone, err := deep.Clone(cs)
			if err != nil {
				panic(err)
			}
			return clone.(*StaticContextState)
		}
	}
	return nil
}

// FindDynamicContextState searches for the DynamicContextState in the extras,
// returning the DynamicContextState if found, otherwise nil.
func FindDynamicContextState(extras *atom.Extras) *DynamicContextState {
	for _, e := range extras.All() {
		if cs, ok := e.(*DynamicContextState); ok {
			clone, err := deep.Clone(cs)
			if err != nil {
				panic(err)
			}
			return clone.(*DynamicContextState)
		}
	}
	return nil
}

// FindAndroidNativeBufferExtra searches for the AndroidNativeBufferExtra in the extras,
// returning the AndroidNativeBufferExtra if found, otherwise nil.
func FindAndroidNativeBufferExtra(extras *atom.Extras) *AndroidNativeBufferExtra {
	for _, e := range extras.All() {
		if di, ok := e.(*AndroidNativeBufferExtra); ok {
			clone, err := deep.Clone(di)
			if err != nil {
				panic(err)
			}
			return clone.(*AndroidNativeBufferExtra)
		}
	}
	return nil
}
