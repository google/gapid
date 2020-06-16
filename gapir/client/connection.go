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
	"fmt"
	"math"
	"time"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapir"
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

// connection implements the gapir.Connection interface.
type connection struct {
	conn       *grpc.ClientConn
	servClient replaysrv.GapirClient
	stream     replaysrv.Gapir_ReplayClient
	authToken  auth.Token
}

func newConnection(addr string, authToken auth.Token, timeout time.Duration) (*connection, error) {
	conn, err := grpc.Dial(addr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)))
	if err != nil {
		return nil, err
	}
	s := replaysrv.NewGapirClient(conn)
	return &connection{conn: conn, servClient: s, authToken: authToken}, nil
}

// Close shutdown the GAPIR connection.
func (c *connection) Close() {
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
func (c *connection) Ping(ctx context.Context) error {
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
func (c *connection) Shutdown(ctx context.Context) error {
	if c.servClient == nil {
		return log.Err(ctx, nil, "Gapir not connected")
	}

	// Use a clean context, since ctx is most likely already cancelled.
	sdCtx := c.attachAuthToken(context.Background())
	_, err := c.servClient.Shutdown(sdCtx, &replaysrv.ShutdownRequest{})
	if err != nil {
		return log.Err(ctx, err, "Sending shutdown request")
	}
	return nil
}

// SendResources sends the given resources data to the connected GAPIR device.
func (c *connection) SendResources(ctx context.Context, resources []byte) error {
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
		return log.Err(ctx, err, "Sending resources")
	}
	return nil
}

// SendPayload sends the given payload to the connected GAPIR device.
func (c *connection) SendPayload(ctx context.Context, payload gapir.Payload) error {
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

// SendFenceReady signals the device to continue a replay.
func (c *connection) SendFenceReady(ctx context.Context, id uint32) error {
	if c.conn == nil || c.servClient == nil {
		return log.Err(ctx, nil, "Gapir not connected")
	}
	if c.stream == nil {
		return log.Err(ctx, nil, "Replay Communication not initiated")
	}
	fenceReadyReq := replaysrv.ReplayRequest{
		Req: &replaysrv.ReplayRequest_FenceReady{
			FenceReady: &replaysrv.FenceReady{
				Id: id,
			},
		},
	}
	err := c.stream.Send(&fenceReadyReq)
	if err != nil {
		return log.Errf(ctx, err, "Sending replay fence %v ready", id)
	}
	return nil
}

// PrewarmReplay requests the GAPIR device to get itself into the given state
func (c *connection) PrewarmReplay(ctx context.Context, payload string, cleanup string) error {
	if c.conn == nil || c.servClient == nil {
		return log.Err(ctx, nil, "Gapir not connected")
	}
	if c.stream == nil {
		return log.Err(ctx, nil, "Replay Communication not initiated")
	}
	PrerunReq := replaysrv.ReplayRequest{
		Req: &replaysrv.ReplayRequest_Prewarm{
			Prewarm: &replaysrv.PrewarmRequest{
				PrerunId:  payload,
				CleanupId: cleanup,
			},
		},
	}
	err := c.stream.Send(&PrerunReq)
	if err != nil {
		return log.Err(ctx, err, "Sending replay payload")
	}
	return nil
}

// HandleReplayCommunication handles the communication with the GAPIR device on
// a replay stream connection. It sends a replay request with the given
// replayID to the connected GAPIR device, expects the device to request payload
// and sends the given payload to the device. Then for each received message
// from the device, it determines the type of the message and pass it to the
// corresponding given handler to process the type-checked message.
func (c *connection) HandleReplayCommunication(
	ctx context.Context,
	handler gapir.ReplayResponseHandler,
	connected chan error) error {
	ctx = log.Enter(ctx, "HandleReplayCommunication")
	if c.conn == nil || c.servClient == nil {
		return log.Errf(ctx, nil, "Gapir not connected")
	}
	// One Connection is only supposed to be used to handle replay communication
	// in one thread. Initiating another replay communication with a connection
	// which is handling another replay communication will mess up the package
	// order.
	if c.stream != nil {
		connected <- log.Errf(ctx, nil, "Connection: %v is handling another replay communication in another thread. Initiating a new replay on this Connection will mess up the package order for both the existing replay and the new replay", c)
		return log.Errf(ctx, nil, "Connection: %v is handling another replay communication in another thread. Initiating a new replay on this Connection will mess up the package order for both the existing replay and the new replay", c)
	}

	ctx = c.attachAuthToken(ctx)
	replayStream, err := c.servClient.Replay(ctx)
	if err != nil {
		return log.Err(ctx, err, "Getting replay stream client")
	}
	c.stream = replayStream
	connected <- nil
	defer func() {
		if c.stream != nil {
			c.stream.CloseSend()
			c.stream = nil
		}
	}()
	for {
		if c.stream == nil {
			return log.Errf(ctx, nil, "No connection to replayer")
		}
		r, err := c.stream.Recv()
		if err != nil {
			return log.Errf(ctx, err, "Replayer connection lost")
		}
		switch r.Res.(type) {
		case *replaysrv.ReplayResponse_PayloadRequest:
			if err := handler.HandlePayloadRequest(ctx, r.GetPayloadRequest().GetPayloadId()); err != nil {
				return log.Errf(ctx, err, "Handling replay payload request")
			}
		case *replaysrv.ReplayResponse_ResourceRequest:
			if err := handler.HandleResourceRequest(ctx, r.GetResourceRequest()); err != nil {
				return log.Errf(ctx, err, "Handling replay resource request")
			}
		case *replaysrv.ReplayResponse_CrashDump:
			if err := handler.HandleCrashDump(ctx, r.GetCrashDump()); err != nil {
				return log.Errf(ctx, err, "Handling replay crash dump")
			}
			// No valid replay response after crash dump.
			return fmt.Errorf("Replay crash")
		case *replaysrv.ReplayResponse_PostData:
			if err := handler.HandlePostData(ctx, r.GetPostData()); err != nil {
				return log.Errf(ctx, err, "Handling post data")
			}
		case *replaysrv.ReplayResponse_Notification:
			if err := handler.HandleNotification(ctx, r.GetNotification()); err != nil {
				return log.Errf(ctx, err, "Handling notification")
			}
		case *replaysrv.ReplayResponse_Finished:
			if err := handler.HandleFinished(ctx, nil); err != nil {
				return log.Errf(ctx, err, "Handling finished")
			}
		case *replaysrv.ReplayResponse_FenceReadyRequest:
			if err := handler.HandleFenceReadyRequest(ctx, r.GetFenceReadyRequest()); err != nil {
				return log.Errf(ctx, err, "Handling replay fence ready request")
			}
		default:
			return log.Errf(ctx, nil, "Unhandled ReplayResponse type")
		}
	}
}

// BeginReplay begins a replay stream connection and attach the authentication,
// if any, token in the metadata.
func (c *connection) BeginReplay(ctx context.Context, id string, dep string) error {
	ctx = log.Enter(ctx, "Starting replay on gapir device")
	if c.servClient == nil || c.conn == nil || c.stream == nil {
		return log.Errf(ctx, nil, "Gapir not connected")
	}

	idReq := replaysrv.ReplayRequest{
		Req: &replaysrv.ReplayRequest_Replay{
			Replay: &replaysrv.Replay{
				ReplayId:    id,
				DependentId: dep,
			},
		},
	}
	err := c.stream.Send(&idReq)
	if err != nil {
		return log.Err(ctx, err, "Sending replay id")
	}

	return nil
}

// attachAuthToken attaches authentication token to the context as metadata, if
// the authentication token is not empty, and returns the new context. If the
// authentication token is empty, returns the original context.
func (c *connection) attachAuthToken(ctx context.Context) context.Context {
	if len(c.authToken) != 0 {
		return metadata.NewOutgoingContext(ctx,
			metadata.Pairs(gapirAuthTokenMetaDataName, string(c.authToken)))
	}
	return ctx
}
