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

	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
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
	connection *backgroundConnection,
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

func (e executor) execute(ctx context.Context, connection *backgroundConnection) error {
	id, err := database.Store(ctx, &e.payload)
	if err != nil {
		return log.Errf(ctx, err, "Storing replay payload")
	}
	clean, err := connection.SetReplayExecutor(ctx, e)
	if err != nil {
		return err
	}
	defer clean()

	// Kick the communication handler
	err = connection.conn.HandleReplayCommunication(
		ctx, id.String(), connection)
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
