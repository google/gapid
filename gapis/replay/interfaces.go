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

package replay

import (
	"context"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/service"
)

// Support is the optional interface implemented by APIs that can describe
// replay support for particular devices and device types.
type Support interface {
	// GetReplayPriority returns a uint32 representing the preference for
	// replaying this trace on the given device.
	// A lower number represents a higher priority, and Zero represents
	// an inability for the trace to be replayed on the given device.
	GetReplayPriority(context.Context, *device.Instance, *device.MemoryLayout) uint32
}

// QueryIssues is the interface implemented by types that can verify the replay
// performs as expected and without errors.
// If the capture includes FramebufferObservation atoms, this also includes
// checking the replayed framebuffer matches (within reasonable error) the
// framebuffer observed at capture time.
type QueryIssues interface {
	QueryIssues(
		ctx context.Context,
		intent Intent,
		mgr *Manager) ([]Issue, error)
}

// QueryFramebufferAttachment is the interface implemented by types that can
// return the content of a framebuffer attachment at a particular point in a
// capture.
type QueryFramebufferAttachment interface {
	QueryFramebufferAttachment(
		ctx context.Context,
		intent Intent,
		mgr *Manager,
		after atom.ID,
		width, height uint32,
		attachment gfxapi.FramebufferAttachment,
		wireframeMode WireframeMode,
		hints *service.UsageHints) (*image.Image2D, error)
}

// Issue represents a single replay issue reported by QueryIssues.
type Issue struct {
	Atom     atom.ID          // The atom that reported the issue.
	Severity service.Severity // The severity of the issue.
	Error    error            // The issue's error.
}
