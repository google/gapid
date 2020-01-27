// Copyright (C) 2020 Google Inc.
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

package client

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
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/database"
)

// ReplayExecutor handles just the bits related to a
// specific replay.
type ReplayExecutor interface {
	// HandlePostData handles the given post data message.
	HandlePostData(context.Context, *gapir.PostData) error
	// HandleNotification handles the given notification message.
	HandleNotification(context.Context, *gapir.Notification) error
	// HandleFinished is notified when the given replay is finished.
	HandleFinished(context.Context, error) error

	// HandleFenceReadyRequest handles when the replayer is waiting for the server
	// to execute the registered FenceReadyRequestCallback for the provided fence ID
	HandleFenceReadyRequest(context.Context, *gapir.FenceReadyRequest) error
}

type backgroundConnection struct {
	conn     gapir.Connection
	OS       *device.OS
	executor ReplayExecutor
}

func (bgc *backgroundConnection) BeginReplay(ctx context.Context, payload string, dependent string) error {
	return bgc.conn.BeginReplay(ctx, payload, dependent)
}

func (bgc *backgroundConnection) PrewarmReplay(ctx context.Context, payload string, cleanup string) error {
	return bgc.conn.PrewarmReplay(ctx, payload, cleanup)
}

func (bgc *backgroundConnection) SetReplayExecutor(ctx context.Context, executor ReplayExecutor) (func(), error) {
	if bgc.executor != nil {
		return nil, log.Err(ctx, nil, "Cannot set an active replay while one is running")
	}
	bgc.executor = executor
	return func() { bgc.executor = nil }, nil
}

func (bgc *backgroundConnection) HandleFinished(ctx context.Context, err error) error {
	if bgc.executor == nil {
		return log.Err(ctx, nil, "No active replay connection for this returned data")
	}
	return bgc.executor.HandleFinished(ctx, err)
}

// HandlePostData handles the given post data message.
func (bgc *backgroundConnection) HandlePostData(ctx context.Context, postData *gapir.PostData) error {
	if bgc.executor == nil {
		return log.Err(ctx, nil, "No active replay connection for this returned data")
	}
	return bgc.executor.HandlePostData(ctx, postData)
}

// HandleNotification handles the given notification message.
func (bgc *backgroundConnection) HandleNotification(ctx context.Context, notification *gapir.Notification) error {
	if bgc.executor == nil {
		return log.Err(ctx, nil, "No active replay connection for this returned data")
	}
	return bgc.executor.HandleNotification(ctx, notification)
}

// HandlePayloadRequest implements gapir.ReplayResponseHandler interface.
func (bgc *backgroundConnection) HandlePayloadRequest(ctx context.Context, payloadID string) error {
	ctx = status.Start(ctx, "Payload Request")
	defer status.Finish(ctx)

	pid, err := id.Parse(payloadID)
	if err != nil {
		return log.Errf(ctx, err, "Parsing payload ID")
	}
	boxed, err := database.Resolve(ctx, pid)
	if err != nil {
		return log.Errf(ctx, err, "Getting replay payload")
	}
	if payload, ok := boxed.(*gapir.Payload); ok {
		return bgc.conn.SendPayload(ctx, *payload)
	}
	return log.Errf(ctx, err, "Payload type is unexpected: %T", boxed)
}

// HandleFenceReadyRequest implements gapir.ReplayResponseHandler interface.
func (bgc *backgroundConnection) HandleFenceReadyRequest(ctx context.Context, req *gapir.FenceReadyRequest) error {
	if bgc.executor == nil {
		return log.Err(ctx, nil, "No active replay connection for this returned data")
	}

	err := bgc.executor.HandleFenceReadyRequest(ctx, req)
	if err != nil {
		return err
	}

	return bgc.conn.SendFenceReady(ctx, req.GetId())
}

// HandleCrashDump implements gapir.ReplayResponseHandler interface.
func (bgc *backgroundConnection) HandleCrashDump(ctx context.Context, dump *gapir.CrashDump) error {
	if dump == nil {
		return fmt.Errorf("Nil crash dump")
	}
	filepath := dump.GetFilepath()
	crashData := dump.GetCrashData()
	// TODO(baldwinn860): get the actual version from GAPIR in case it ever goes out of sync
	if res, err := reporting.ReportMinidump(reporting.Reporter{
		AppName:    "GAPIR",
		AppVersion: app.Version.String(),
		OSName:     bgc.OS.GetName(),
		OSVersion:  fmt.Sprintf("%v %v.%v.%v", bgc.OS.GetBuild(), bgc.OS.GetMajorVersion(), bgc.OS.GetMinorVersion(), bgc.OS.GetPointVersion()),
	}, filepath, crashData); err != nil {
		return log.Err(ctx, err, "Failed to report crash in GAPIR")
	} else if res != "" {
		log.I(ctx, "Crash Report Uploaded; ID: %v", res)
		file.Remove(file.Abs(filepath))
	}
	return nil
}

// HandleResourceRequest implements gapir.ReplayResponseHandler interface.
func (bgc *backgroundConnection) HandleResourceRequest(ctx context.Context, req *gapir.ResourceRequest) error {
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
	if err := bgc.conn.SendResources(ctx, response); err != nil {
		log.Errf(ctx, err, "Failed to send resources")
	}
	return nil
}
