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
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/service"
)

// Support is the optional interface implemented by APIs that can describe
// replay support for particular devices and device types.
type Support interface {
	// CanReplayOnLocalAndroidDevice returns true if the API can be replayed on
	// a locally connected Android device.
	CanReplayOnLocalAndroidDevice(log.Context) bool

	// CanReplayOn returns true if the API can be replayed on the specified
	// device.
	CanReplayOn(log.Context, *device.Instance) bool
}

// QueryIssues is the interface implemented by types that can verify the replay
// performs as expected and without errors. The returned chan will receive a
// stream of issues detected while replaying the capture and the chan will close
// after the last issue is sent (if any).
// If the capture includes FramebufferObservation atoms, this also includes
// checking the replayed framebuffer matches (within reasonable error) the
// framebuffer observed at capture time.
type QueryIssues interface {
	QueryIssues(
		ctx log.Context,
		intent Intent,
		mgr *Manager,
		out chan<- Issue)
}

// QueryFramebufferAttachment is the interface implemented by types that can
// return the content of a framebuffer attachment at a particular point in a
// capture.
type QueryFramebufferAttachment interface {
	QueryFramebufferAttachment(
		ctx log.Context,
		intent Intent,
		mgr *Manager,
		after atom.ID,
		width, height uint32,
		attachment gfxapi.FramebufferAttachment,
		wireframeMode WireframeMode) (*image.Image2D, error)
}

// Issue represents a single replay issue reported by QueryIssues.
type Issue struct {
	Atom     atom.ID          // The atom that reported the issue.
	Severity service.Severity // The severity of the issue.
	Error    error            // The issue's error.
}
