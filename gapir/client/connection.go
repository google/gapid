// Copyright (C) 2018 Google Inc.
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
	"time"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/log"
	replaysrv "github.com/google/gapid/gapir/replay_service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// The key of the metadata value that contains authentication token. This is
	// common knowledge shared between GAPIR client (which is GAPIS) and GAPIR
	// server (which is GAPIR device)
	gapirAuthTokenMetaDataName = "gapir-auth-token"
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
)

// Connection represents a connection between GAPIS and GAPIR. It wraps the
// internal gRPC connections and holds authentication token. A new Connection
// should be created only by client.Client.
// TODO: The functionality of replay stream and Ping/Shutdown can be separated.
// The GAPIS code should only use the replay stream, Ping/Shutdown should be
// managed by client.session.
type Connection struct {
	conn       *grpc.ClientConn
	servClient replaysrv.GapirClient
	stream     replaysrv.Gapir_ReplayClient
	authToken  auth.Token
}

func newConnection(addr string, authToken auth.Token, timeout time.Duration) (*Connection, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(timeout))
	if err != nil {
		return nil, err
	}
	s := replaysrv.NewGapirClient(conn)
	return &Connection{conn: conn, servClient: s, authToken: authToken}, nil
}

// Close shutdown the GAPIR connection.
func (c *Connection) Close() {
	if c.stream != nil {
		c.stream.CloseSend()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	c.conn = nil
	c.servClient = nil
	c.stream = nil
}

// Ping sends a ping to the connected GAPIR device and expect a response to make
// sure the connection is alive.
func (c *Connection) Ping(ctx context.Context) error {
	if c.servClient == nil {
		return log.Err(ctx, nil, "Gapir not connected")
	}
	ctx = c.attachAuthToken(ctx)
	r, err := c.servClient.Ping(ctx, &replaysrv.PingRequest{})
	if err != nil {
		return log.Err(ctx, err, "Sending ping")
	}
	if r == nil {
		return log.Err(ctx, nil, "No response for ping")
	}
	return nil
}

// Shutdown sends a signal to the connected GAPIR device to shutdown the
// connection server.
func (c *Connection) Shutdown(ctx context.Context) error {
	if c.servClient == nil {
		return log.Err(ctx, nil, "Gapir not connected")
	}
	ctx = c.attachAuthToken(ctx)
	_, err := c.servClient.Shutdown(ctx, &replaysrv.ShutdownRequest{})
	if err != nil {
		return log.Err(ctx, err, "Sending shutdown request")
	}
	return nil
}

// SendResources sends the given resources data to the connected GAPIR device.
func (c *Connection) SendResources(ctx context.Context, resources []byte) error {
	if c.conn == nil || c.servClient == nil {
		return log.Err(ctx, nil, "Gapir not connected")
	}
	if c.stream == nil {
		return log.Err(ctx, nil, "Replay communication not initiated")
	}
	resReq := replaysrv.ReplayRequest{
		Req: &replaysrv.ReplayRequest_Resources{
			Resources: &replaysrv.Resources{Data: resources},
		},
	}
	if err := c.stream.Send(&resReq); err != nil {
		return log.Err(ctx, err, "Sneding resources")
	}
	return nil
}

// SendPayload sends the given payload to the connected GAPIR device.
func (c *Connection) SendPayload(ctx context.Context, payload Payload) error {
	if c.conn == nil || c.servClient == nil {
		return log.Err(ctx, nil, "Gapir not connected")
	}
	if c.stream == nil {
		return log.Err(ctx, nil, "Replay Communication not initiated")
	}
	payloadReq := replaysrv.ReplayRequest{
		Req: &replaysrv.ReplayRequest_Payload{
			Payload: &payload,
		},
	}
	err := c.stream.Send(&payloadReq)
	if err != nil {
		return log.Err(ctx, err, "Sending replay payload")
	}
	return nil
}

// ReplayResponseHandler handles all kinds of ReplayResponse messages received
// from a connected GAPIR device.
type ReplayResponseHandler interface {
	// HandlePayloadRequest handles the given payload request message.
	HandlePayloadRequest(context.Context, *Connection) error
	// HandlePayloadRequest handles the given resource request message.
	HandleResourceRequest(context.Context, *ResourceRequest, *Connection) error
	// HandlePayloadRequest handles the given crash dump message.
	HandleCrashDump(context.Context, *CrashDump, *Connection) error
	// HandlePayloadRequest handles the given post data message.
	HandlePostData(context.Context, *PostData, *Connection) error
	// HandleNotification handles the given notification message.
	HandleNotification(context.Context, *Notification, *Connection) error
}

// HandleReplayCommunication handles the communication with the GAPIR device on
// a replay stream connection. It sends a replay request with the given
// replayID to the connected GAPIR device, expects the device to request payload
// and sends the given payload to the device. Then for each received message
// from the device, it determines the type of the message and pass it to the
// corresponding given handler to process the type-checked message.
func (c *Connection) HandleReplayCommunication(
	ctx context.Context,
	replayID string,
	// payload Payload,
	handler ReplayResponseHandler) error {
	ctx = log.Enter(ctx, "HandleReplayCommunication")
	if c.conn == nil || c.servClient == nil {
		return log.Errf(ctx, nil, "Gapir not connected")
	}
	// One Connection is only supposed to be used to handle replay communication
	// in one thread. Initiating another replay communication with a connection
	// which is handling another replay communication will mess up the package
	// order.
	if c.stream != nil {
		return log.Errf(ctx, nil, "Connection: %v is handling another replay communication in another thread. Initiating a new replay on this Connection will mess up the package order for both the existing replay and the new replay", c)
	}
	if err := c.beginReplay(ctx, replayID); err != nil {
		return err
	}
	defer func() {
		if c.stream != nil {
			c.stream.CloseSend()
			c.stream = nil
		}
	}()
	for {
		if c.stream == nil {
			return log.Errf(ctx, nil, "Replay stream connection lost")
		}
		r, err := c.stream.Recv()
		if err != nil {
			return log.Errf(ctx, err, "Recv")
		}
		switch r.Res.(type) {
		case *replaysrv.ReplayResponse_PayloadRequest:
			if err := handler.HandlePayloadRequest(ctx, c); err != nil {
				return log.Errf(ctx, err, "Handling replay payload request")
			}
		case *replaysrv.ReplayResponse_ResourceRequest:
			if err := handler.HandleResourceRequest(ctx, r.GetResourceRequest(), c); err != nil {
				return log.Errf(ctx, err, "Handling replay resource request")
			}
		case *replaysrv.ReplayResponse_CrashDump:
			if err := handler.HandleCrashDump(ctx, r.GetCrashDump(), c); err != nil {
				return log.Errf(ctx, err, "Handling replay crash dump")
			}
			// No valid replay response after crash dump.
			return nil
		case *replaysrv.ReplayResponse_PostData:
			if err := handler.HandlePostData(ctx, r.GetPostData(), c); err != nil {
				return log.Errf(ctx, err, "Handling post data")
			}
		case *replaysrv.ReplayResponse_Notification:
			if err := handler.HandleNotification(ctx, r.GetNotification(), c); err != nil {
				return log.Errf(ctx, err, "Handling notification")
			}
		case *replaysrv.ReplayResponse_Finished:
			log.D(ctx, "Replay Finished Response received")
			return nil
		default:
			return log.Errf(ctx, nil, "Unhandled ReplayResponse type")
		}
	}
}

func (c *Connection) beginReplay(ctx context.Context, id string) error {
	ctx = log.Enter(ctx, "Starting replay on gapir device")
	if c.servClient == nil || c.conn == nil {
		return log.Errf(ctx, nil, "Gapir not connected")
	}
	ctx = c.attachAuthToken(ctx)
	replayStream, err := c.servClient.Replay(ctx)
	if err != nil {
		return log.Err(ctx, err, "Gettting replay stream client")
	}
	idReq := replaysrv.ReplayRequest{
		Req: &replaysrv.ReplayRequest_ReplayId{ReplayId: id},
	}
	err = replayStream.Send(&idReq)
	if err != nil {
		return log.Err(ctx, err, "Sending replay id")
	}
	c.stream = replayStream
	return nil
}

// attachAuthToken attaches authentication token to the context as metadata, if
// the authentication token is not empty, and returns the new context. If the
// authentication token is empty, returns the original context.
func (c *Connection) attachAuthToken(ctx context.Context) context.Context {
	if len(c.authToken) != 0 {
		return metadata.NewContext(ctx,
			metadata.Pairs(gapirAuthTokenMetaDataName, string(c.authToken)))
	}
	return ctx
}
