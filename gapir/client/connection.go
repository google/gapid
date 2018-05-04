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
	"time"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapir/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	gapirAuthTokenMetaDataName = "gapir-auth-token"
)

type ResourceInfoList []*service.ResourceInfo

func (l *ResourceInfoList) Append(resourceID string, size uint32) {
	*l = append(*l, &service.ResourceInfo{
		Id:   resourceID,
		Size: size,
	})
}

type Payload service.Payload

func NewPayload(stackSize, volatileMemorySize uint32, constants []byte, resources ResourceInfoList, opcodes []byte) *Payload {
	return (*Payload)(&service.Payload{
		StackSize:          stackSize,
		VolatileMemorySize: volatileMemorySize,
		Constants:          constants,
		Resources:          ([]*service.ResourceInfo)(resources),
		Opcodes:            opcodes,
	})
}

func (p *Payload) ToProto() service.Payload {
	return (service.Payload)(*p)
}

type Resources struct {
	service.Resources
}

type ResourceRequest struct {
	service.ResourceRequest
}

type CrashDump struct {
	service.CrashDump
}

type PostData struct {
	service.PostData
}

type Notification struct {
	service.Notification
}

type Connection struct {
	conn       *grpc.ClientConn
	servClient service.GapirClient
	stream     service.Gapir_ReplayClient
	authToken  auth.Token
}

func newConnection(addr string, authToken auth.Token, timeout time.Duration) (*Connection, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(timeout))
	if err != nil {
		return nil, err
	}
	s := service.NewGapirClient(conn)
	return &Connection{conn: conn, servClient: s, authToken: authToken}, nil
}

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

func (c *Connection) Ping(ctx context.Context) error {
	if c.servClient == nil {
		return fmt.Errorf("Not connected")
	}
	r, err := c.servClient.Ping(ctx, &service.PingRequest{})
	if err != nil {
		return log.Err(ctx, err, "Sending ping request")
	}
	if r.GetPong() != "PONG" {
		return fmt.Errorf("Expected: 'PONG', got: '%v'", r.GetPong())
	}
	return nil
}

func (c *Connection) Shutdown(ctx context.Context) error {
	if c.servClient == nil {
		return fmt.Errorf("Not connected")
	}
	_, err := c.servClient.Shutdown(ctx, &service.ShutdownRequest{})
	if err != nil {
		return log.Err(ctx, err, "Sending shutdown request")
	}
	return nil
}

func (c *Connection) SendResources(ctx context.Context, resources []byte) error {
	if c.conn == nil || c.servClient == nil {
		return fmt.Errorf("Not connected")
	}
	if c.stream == nil {
		return fmt.Errorf("Replay stream not opened")
	}
	resReq := service.ReplayRequest{
		Req: &service.ReplayRequest_Resources{
			Resources: &service.Resources{Data: resources},
		},
	}
	if err := c.stream.Send(&resReq); err != nil {
		return log.Err(ctx, err, "Sneding resources")
	}
	return nil
}

func (c *Connection) HandleReplayCommunication(
	ctx context.Context,
	replayID string,
	payload Payload,
	handleResourceRequest func(context.Context, *ResourceRequest, *Connection) error,
	handleCrashDump func(context.Context, *CrashDump, *Connection) error,
	handlePostData func(context.Context, *PostData, *Connection) error,
	handleNotification func(context.Context, *Notification, *Connection) error) error {
	ctx = log.Enter(ctx, "HandleReplayCommunication")
	if c.conn == nil || c.servClient == nil {
		return log.Errf(ctx, nil, "Gapir not connected")
	}
	// Drop the unfinished replay with on connection.
	if c.stream != nil {
		c.stream.CloseSend()
		c.stream = nil
	}
	if err := c.beginReplay(ctx, replayID, payload); err != nil {
		return err
	}
	for {
		if c.stream == nil {
			return log.Errf(ctx, nil, "Replay stream connection lost")
		}
		r, err := c.stream.Recv()
		if err != nil {
			return log.Errf(ctx, err, "Recv")
		}
		switch r.Res.(type) {
		case *service.ReplayResponse_ResourceRequest:
			if err := handleResourceRequest(ctx, &ResourceRequest{*r.GetResourceRequest()}, c); err != nil {
				return log.Errf(ctx, err, "Handling replay resource request")
			}
		case *service.ReplayResponse_CrashDump:
			if err := handleCrashDump(ctx, &CrashDump{*r.GetCrashDump()}, c); err != nil {
				return log.Errf(ctx, err, "Handling replay crash dump")
			}
			// No valid replay response after crash dump.
			c.stream.CloseSend()
			c.stream = nil
			return nil
		case *service.ReplayResponse_PostData:
			if err := handlePostData(ctx, &PostData{*r.GetPostData()}, c); err != nil {
				return log.Errf(ctx, err, "Handing post data")
			}
		case *service.ReplayResponse_Notification:
			if err := handleNotification(ctx, &Notification{*r.GetNotification()}, c); err != nil {
				return log.Errf(ctx, err, "Handling notification")
			}
		case *service.ReplayResponse_Finished:
			log.D(ctx, "Replay Finished Response received")
			c.stream.CloseSend()
			c.stream = nil
			return nil
		default:
			return log.Errf(ctx, nil, "Unhandled ReplayResponse type")
		}
	}
}

func (c *Connection) beginReplay(ctx context.Context, id string, payload Payload) error {
	ctx = log.Enter(ctx, "Starting replay on gapir device")
	if c.servClient == nil || c.conn == nil {
		return log.Errf(ctx, nil, "Gapir not connected")
	}
	// If there is an authentication token, attach it in the metadata
	if len(c.authToken) != 0 {
		ctx = metadata.NewContext(ctx,
			metadata.Pairs(gapirAuthTokenMetaDataName, string(c.authToken)))
	}
	replayStream, err := c.servClient.Replay(ctx)
	if err != nil {
		return log.Err(ctx, err, "Gettting replay stream client")
	}
	idReq := service.ReplayRequest{
		Req: &service.ReplayRequest_ReplayId{ReplayId: id},
	}
	err = replayStream.Send(&idReq)
	if err != nil {
		return log.Err(ctx, err, "Sending replay id")
	}
	c.stream = replayStream

	res, err := replayStream.Recv()
	if err != nil {
		return log.Err(ctx, err, "Expecting payload request")
	}
	if res.GetPayloadRequest() == nil {
		return log.Errf(ctx, nil, "Expecting payload request, actually got: '%T'", res.Res)
	}
	payloadReq := service.ReplayRequest{
		Req: &service.ReplayRequest_Payload{
			Payload: (*service.Payload)(&payload),
		},
	}
	err = replayStream.Send(&payloadReq)
	if err != nil {
		return log.Err(ctx, err, "Sending replay payload")
	}
	return nil
}
