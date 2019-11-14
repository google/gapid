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
	"fmt"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/crash/reporting"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/database"
)

// ReplayExecutor handles just the bits related to a
// specific replay.
type ReplayExecutor interface {
	// HandlePostData handles the given post data message.
	HandlePostData(context.Context, *gapir.PostData, gapir.Connection) error
	// HandleNotification handles the given notification message.
	HandleNotification(context.Context, *gapir.Notification, gapir.Connection) error
	// HandleFinished is notified when the given replay is finished.
	HandleFinished(context.Context, error, gapir.Connection) error

	HandleFenceReadyRequest(context.Context, *gapir.FenceReadyRequest, gapir.Connection) error
}

type backgroundConnection struct {
	conn     gapir.Connection
	OS       *device.OS
	ABI      *device.ABI
	executor ReplayExecutor
}

func (e *backgroundConnection) BeginReplay(ctx context.Context, payload string, dependent string) error {
	return e.conn.BeginReplay(ctx, payload, dependent)
}

func (e *backgroundConnection) PrewarmReplay(ctx context.Context, payload string, cleanup string) error {
	return e.conn.PrewarmReplay(ctx, payload, cleanup)
}

func (e *backgroundConnection) SetReplayExecutor(ctx context.Context, x ReplayExecutor) (func(), error) {
	if e.executor != nil {
		return nil, log.Err(ctx, nil, "Cannot set an active replay while one is running")
	}
	e.executor = x
	return func() { e.executor = nil }, nil
}

func (e *backgroundConnection) HandleFinished(ctx context.Context, err error, conn gapir.Connection) error {
	if e.executor == nil {
		return log.Err(ctx, nil, "No active replay connection for this returned data")
	}
	return e.executor.HandleFinished(ctx, err, conn)
}

// HandlePostData handles the given post data message.
func (e *backgroundConnection) HandlePostData(ctx context.Context, pd *gapir.PostData, conn gapir.Connection) error {
	if e.executor == nil {
		return log.Err(ctx, nil, "No active replay connection for this returned data")
	}
	return e.executor.HandlePostData(ctx, pd, conn)
}

// HandleNotification handles the given notification message.
func (e *backgroundConnection) HandleNotification(ctx context.Context, notification *gapir.Notification, conn gapir.Connection) error {
	if e.executor == nil {
		return log.Err(ctx, nil, "No active replay connection for this returned data")
	}
	return e.executor.HandleNotification(ctx, notification, conn)
}

// HandlePayloadRequest implements gapir.ReplayResponseHandler interface.
func (e *backgroundConnection) HandlePayloadRequest(ctx context.Context, payloadID string, conn gapir.Connection) error {
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
	if pl, ok := boxed.(*gapir.Payload); ok {
		return conn.SendPayload(ctx, *pl)
	}
	return log.Errf(ctx, err, "Payload type is unexpected: %T", boxed)
}

// HandleFenceReadyRequest implements gapir.ReplayResponseHandler interface.
func (e *backgroundConnection) HandleFenceReadyRequest(ctx context.Context, req *gapir.FenceReadyRequest, conn gapir.Connection) error {
	if e.executor == nil {
		return log.Err(ctx, nil, "No active replay connection for this returned data")
	}
	return e.executor.HandleFenceReadyRequest(ctx, req, conn)
}

// HandleCrashDump implements gapir.ReplayResponseHandler interface.
func (e *backgroundConnection) HandleCrashDump(ctx context.Context, dump *gapir.CrashDump, conn gapir.Connection) error {
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
func (e *backgroundConnection) HandleResourceRequest(ctx context.Context, req *gapir.ResourceRequest, conn gapir.Connection) error {
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

// MakeBackgroundConnection creates a connection to the replay device that persists in the background.
func MakeBackgroundConnection(ctx context.Context, device bind.Device, conn gapir.Connection, replayABI *device.ABI) (*backgroundConnection, error) {
	bgc := &backgroundConnection{conn: conn, ABI: replayABI, OS: device.Instance().GetConfiguration().GetOS()}
	c := make(chan error)
	cctx := keys.Clone(context.Background(), ctx)
	crash.Go(func() {
		// This shouldn't be sitting on this context
		cctx := status.PutTask(cctx, nil)
		cctx = status.StartBackground(cctx, "Handle Replay Communication")
		defer status.Finish(cctx)
		// Kick the communication handler
		err := conn.HandleReplayCommunication(
			cctx, bgc, c)
		if err != nil {
			log.E(cctx, "Error communication with gapir: %v", err)
		}
	})
	err := <-c
	if err != nil {
		return nil, err
	}
	return bgc, nil
}

// Creates a background connection to execute commands
func (m *manager) connect(ctx context.Context, device bind.Device, replayABI *device.ABI) (*backgroundConnection, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if conn, ok := m.connections[device.Instance().ID.ID()]; ok {
		if conn.ABI.SameAs(replayABI) {
			return conn, nil
		}
		conn.conn.Close()
	}

	conn, err := m.gapir.Connect(ctx, device, replayABI)
	if err != nil {
		return nil, err
	}
	bgc, err := MakeBackgroundConnection(ctx, device, conn, replayABI)
	m.connections[device.Instance().ID.ID()] = bgc
	return bgc, nil
}
