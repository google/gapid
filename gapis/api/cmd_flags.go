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

package api

// CmdFlags is a bitfield describing characteristics of a command.
type CmdFlags uint32

const (
	DrawCall CmdFlags = 1 << iota
	TransformFeedback
	Clear
	EndOfFrame
	PushUserMarker
	PopUserMarker
	UserMarker
	ExecutedDraw
	ExecutedDispatch
	ExecutedCommandBuffer
	Submission
)

// IsDrawCall returns true if the command is a draw call.
func (f CmdFlags) IsDrawCall() bool { return (f & DrawCall) != 0 }

// IsTransformFeedback returns true if the command is a transform-feedback call.
func (f CmdFlags) IsTransformFeedback() bool { return (f & TransformFeedback) != 0 }

// IsClear returns true if the command is a clear call.
func (f CmdFlags) IsClear() bool { return (f & Clear) != 0 }

// IsEndOfFrame returns true if the command represents the end of a frame.
func (f CmdFlags) IsEndOfFrame() bool { return (f & EndOfFrame) != 0 }

// IsPushUserMarker returns true if the command represents the start of a user
// marker group. The command may implement the Labeled interface to expose the
// marker name.
func (f CmdFlags) IsPushUserMarker() bool { return (f & PushUserMarker) != 0 }

// IsPopUserMarker returns true if the command represents the end of the last
// pushed user marker.
func (f CmdFlags) IsPopUserMarker() bool { return (f & PopUserMarker) != 0 }

// IsUserMarker returns true if the command represents a non-grouping user
// marker.
// The command may implement the Labeled interface to expose the marker name.
func (f CmdFlags) IsUserMarker() bool { return (f & UserMarker) != 0 }

// IsExecutedDraw returns true if the command is a draw call that gets executed
// as a subcommand.
func (f CmdFlags) IsExecutedDraw() bool { return (f & ExecutedDraw) != 0 }

// IsExecutedDispatch returns true if the command is a dispatch that gets executed
// as a subcommand.
func (f CmdFlags) IsExecutedDispatch() bool { return (f & ExecutedDispatch) != 0 }

// IsExecutedCommandBuffer returns true if the command executes a prerecorded command
// buffer.
func (f CmdFlags) IsExecutedCommandBuffer() bool { return (f & ExecutedCommandBuffer) != 0 }

// IsSubmission returns true if the command is a submission
func (f CmdFlags) IsSubmission() bool { return (f & Submission) != 0 }
