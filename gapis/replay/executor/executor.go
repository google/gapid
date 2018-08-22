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

// Package executor contains the Execute function for sending a replay to a device.
package executor

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash/reporting"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/file"
	gapir "github.com/google/gapid/gapir/client"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/replay/builder"
)

type executor struct {
	payload            gapir.Payload
	handlePost         builder.PostDataHandler
	handleNotification builder.NotificationHandler
	memoryLayout       *device.MemoryLayout
	OS                 *device.OS
}

// Execute sends the replay payload for execution on the target replay device
// communicating on connection.
// decoder will be used for decoding all postback reponses. Once a postback
// response is decoded, the corresponding handler in the handlers map will be
// called.
func Execute(
	ctx context.Context,
	payload gapir.Payload,
	handlePost builder.PostDataHandler,
	handleNotification builder.NotificationHandler,
	connection *gapir.Connection,
	memoryLayout *device.MemoryLayout,
	os *device.OS) error {

	ctx = status.Start(ctx, "Execute")
	defer status.Finish(ctx)

	// The memoryLayout is specific to the ABI of the requested capture,
	// while the OS is not. Thus a device.Configuration is not applicable here.
	return executor{
		payload:            payload,
		handlePost:         handlePost,
		handleNotification: handleNotification,
		memoryLayout:       memoryLayout,
		OS:                 os,
	}.execute(ctx, connection)
}

func (e executor) execute(ctx context.Context, connection *gapir.Connection) error {
	id, err := database.Store(ctx, &e.payload)
	if err != nil {
		return log.Errf(ctx, err, "Storing replay payload")
	}

	// Kick the communication handler
	err = connection.HandleReplayCommunication(
		ctx, id.String(), e)
	if err != nil {
		log.E(ctx, "Error communication with gapir: %v", err)
		return log.Err(ctx, err, "Communicating with gapir")
	}
	return nil
}

// HandlePayloadRequest implements gapir.ReplayResponseHandler interface.
func (e executor) HandlePayloadRequest(ctx context.Context, conn *gapir.Connection) error {
	ctx = status.Start(ctx, "Payload Request")
	defer status.Finish(ctx)

	return conn.SendPayload(ctx, e.payload)
}

// HandlePostData implements gapir.ReplayResponseHandler interface.
func (e executor) HandlePostData(ctx context.Context, postData *gapir.PostData, conn *gapir.Connection) error {
	ctx = status.Start(ctx, "Post Data (count: %d)", len(postData.PostDataPieces))
	defer status.Finish(ctx)

	e.handlePost(postData)
	return nil
}

// HandleNotification implements gapir.ReplayResponseHandler interface.
func (e executor) HandleNotification(ctx context.Context, notification *gapir.Notification, conn *gapir.Connection) error {
	e.handleNotification(notification)
	return nil
}

// HandleCrashDump implements gapir.ReplayResponseHandler interface.
func (e executor) HandleCrashDump(ctx context.Context, dump *gapir.CrashDump, conn *gapir.Connection) error {
	if dump == nil {
		return fmt.Errorf("Nil crash dump")
	}
	filepath := dump.GetFilepath()
	crashData := dump.GetCrashData()
	// TODO(baldwinn860): get the actual version from GAPIR in case it ever goes out of sync
	if res, err := reporting.ReportMinidump(reporting.Reporter{
		AppName:    "GAPIR",
		AppVersion: app.Version.String(),
		OSName:     e.OS.GetName(),
		OSVersion:  fmt.Sprintf("%v %v.%v.%v", e.OS.GetBuild(), e.OS.GetMajorVersion(), e.OS.GetMinorVersion(), e.OS.GetPointVersion()),
	}, filepath, crashData); err != nil {
		return log.Err(ctx, err, "Failed to report crash in GAPIR")
	} else if res != "" {
		log.I(ctx, "Crash Report Uploaded; ID: %v", res)
		file.Remove(file.Abs(filepath))
	}
	return nil
}

// HandleResourceRequest implements gapir.ReplayResponseHandler interface.
func (e executor) HandleResourceRequest(ctx context.Context, req *gapir.ResourceRequest, conn *gapir.Connection) error {
	ctx = status.Start(ctx, "Resources Request (count: %d)", len(req.GetIds()))
	defer status.Finish(ctx)

	ctx = log.Enter(ctx, "handleResourceRequest")
	if req == nil {
		return log.Err(ctx, nil, "Cannot handle nil resource request")
	}
	ids := req.GetIds()
	totalExpectedSize := req.GetExpectedTotalSize()
	totalReturnedSize := uint64(0)
	response := make([]byte, 0, totalExpectedSize)
	db := database.Get(ctx)
	for _, idStr := range ids {
		rID, err := id.Parse(idStr)
		if err != nil {
			return log.Errf(ctx, err, "Failed to parse resource id: %v", idStr)
		}
		obj, err := db.Resolve(ctx, rID)
		if err != nil {
			return log.Errf(ctx, err, "Failed to parse resource id: %v", idStr)
		}
		objData := obj.([]byte)
		response = append(response, objData...)
		totalReturnedSize += uint64(len(objData))
	}
	if totalReturnedSize != totalExpectedSize {
		return log.Errf(ctx, nil, "Total resource size mismatch. expected: %v, got: %v", totalExpectedSize, totalReturnedSize)
	}
	if err := conn.SendResources(ctx, response); err != nil {
		log.Errf(ctx, err, "Failed to send resources")
	}
	return nil
}
