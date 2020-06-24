// Copyright (C) 2019 Google Inc.
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

// Package gapir provides the gapir API.
package gapir

import (
	replaysrv "github.com/google/gapid/gapir/replay_service"
	"github.com/google/gapid/gapis/service/severity"
)

// Type aliases to avoid GAPIS code from using gRPC generated code directly.
// Only the types aliased here can be used by GAPIS code.
type (
	// ResourceInfo contains Id and Size information of a resource.
	ResourceInfo = replaysrv.ResourceInfo
	// Resources contains a list of byte arrays Data each represent the data of a resource
	Resources = replaysrv.Resources
	// Payload contains StackSize, VolatileMemorySize, Constants, a list of information of Resources, and Opcodes for replay in bytes.
	Payload = replaysrv.Payload
	// ResourceRequest contains the total expected size of requested resources data in bytes and the Ids of the resources to be requested.
	ResourceRequest = replaysrv.ResourceRequest
	// CrashDump contains the Filepath of the crash dump file on GAPIR device, and the CrashData in bytes
	CrashDump = replaysrv.CrashDump
	// PostData contains a list of PostDataPieces, each piece contains an Id in string and Data in bytes
	PostData = replaysrv.PostData
	// Notification contains an Id, the ApiIndex, Label, Msg in string and arbitary Data in bytes.
	Notification = replaysrv.Notification
	// Severity represents the severity level of notification messages. It uses the same enum as gapis
	Severity = severity.Severity
	// FenceReadyRequest is sent when the device is waiting for the server perform a task
	FenceReadyRequest = replaysrv.FenceReadyRequest
	// FenceReady signals that the server finished a task and replay can continue
	FenceReady = replaysrv.FenceReady
)
