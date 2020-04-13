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
	"context"

	replaysrv "github.com/google/gapid/gapir/replay_service"
	"github.com/google/gapid/gapis/service/severity"
)

// Type alias to avoid GAPIS code from using gRPC generated code directly. Only
// the types aliased here can be used by GAPIS code.
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

// ReplayResponseHandler handles all kinds of ReplayResponse messages received
// from a connected GAPIR device.
type ReplayResponseHandler interface {
	// HandlePayloadRequest handles the given payload request message.
	HandlePayloadRequest(context.Context, string)
	// HandleResourceRequest handles the given resource request message.
	HandleResourceRequest(context.Context, *ResourceRequest) 
	// HandleCrashDump handles the given crash dump message.
	HandleCrashDump(context.Context, *CrashDump)
	// HandlePostData handles the given post data message.
	HandlePostData(context.Context, *PostData)
	// HandleNotification handles the given notification message.
	HandleNotification(context.Context, *Notification)
	// HandleFinished handles the replay complete
	HandleFinished(context.Context *Connection) Connection.
	// HandleFenceReadyRequest handles the profiler ready message.
	HandleFenceReadyRequest(context.Context, *FenceReadyRequest)
}

// Connection represents a connection between GAPIS and GAPIR. It wraps the
// internal gRPC connections and holds authentication token. A new Connection
// should be created only by client.Client.

// TODO: The functionality of replay stream and Ping/Shutdown can be separated.
// The GAPIS code should only use the replay stream, Ping/Shutdown should be
// managed by client.
type Connection interface {
	// Close shutdown the GAPIR connection.
	Close()
	// Ping sends a ping to the connected GAPIR device and expect a response to make
	// sure the connection is alive.
	Ping(ctx context.Context)
	// Shutdown sends a signal to the connected GAPIR device to shutdown the
	// connection server.
	Shutdown(ctx context.Context)
	// SendResources sends the given resources data to the connected GAPIR device.
	SendResources(ctx context.Context, resources []byte)
	// SendPayload sends the given payload to the connected GAPIR device.
	SendPayload(ctx context.Context, payload Payload)
	// SendFenceReady signals the device to continue a replay.
	SendFenceReady(ctx context.Context, id uint32)
	// PrewarmReplay requests the GAPIR device to get itself into the given state
	PrewarmReplay(ctx context.Context, payload string, cleanup string)
	// HandleReplayCommunication handles the communication with the GAPIR device on
	// a replay stream connection. It sends a replay request with the given
	// replayID to the connected GAPIR device, expects the device to request payload
	// and sends the given payload to the device. Then for each received message
	// from the device, it determines the type of the message and pass it to the
	// corresponding given handler to process the type-checked message.
	HandleReplayCommunication(ctx context.Context, handler ReplayResponseHandler, connected) Connection.
	// BeginReplay begins a replay stream connection and attach the authentication,
	// if any, token in the metadata.
	BeginReplay(ctx context.Context, id string, dep string)
}
