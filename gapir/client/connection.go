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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapir/service"
	"google.golang.org/grpc"
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
}

func NewConnection(addr string) (*Connection, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	s := service.NewGapirClient(conn)
	return &Connection{conn: conn, servClient: s}, nil
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
	if c.conn == nil || c.servClient == nil {
		return fmt.Errorf("Not connected")
	}
	if c.stream != nil {
		c.stream.CloseSend()
		c.stream = nil
	}
	log.W(ctx, "in HandleReplayCommunication")
	for count := 0; count < 10; count++ {
		if err := c.beginReplay(ctx, replayID, payload); err != nil {
			if count == 10 {
				return err
			}
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	log.W(ctx, "begin replay returns successfully")
	for {
		r, err := c.stream.Recv()
		if err != nil {
			return err
		}
		switch r.Res.(type) {
		case *service.ReplayResponse_ResourceRequest:
			log.W(ctx, "handle resource request")
			if err := handleResourceRequest(ctx, &ResourceRequest{*r.GetResourceRequest()}, c); err != nil {
				return fmt.Errorf("Failed to handle resource request :%v", err)
			}
		case *service.ReplayResponse_CrashDump:
			log.W(ctx, "handle crash dump")
			if err := handleCrashDump(ctx, &CrashDump{*r.GetCrashDump()}, c); err != nil {
				return fmt.Errorf("Failed to handle crash dump: %v", err)
			}
			c.Close()
		case *service.ReplayResponse_PostData:
			log.W(ctx, "handle post data")
			if err := handlePostData(ctx, &PostData{*r.GetPostData()}, c); err != nil {
				return fmt.Errorf("Failed to handle post data: %v", err)
			}
		case *service.ReplayResponse_Notification:
			log.W(ctx, "handle notification")
			if err := handleNotification(ctx, &Notification{*r.GetNotification()}, c); err != nil {
				return fmt.Errorf("Failed to handle notification: %v", err)
			}
		case *service.ReplayResponse_Finished:
			log.W(ctx, "handle finished")
			c.stream.CloseSend()
			c.stream = nil
			return nil
		}
	}
}

func (c *Connection) beginReplay(ctx context.Context, id string, payload Payload) error {
	if c.servClient == nil || c.conn == nil {
		return fmt.Errorf("Not connected")
	}
	ctx = log.Enter(ctx, "Requesting replay on gapir device")
	log.W(ctx, "trying to begin replay")
	replayStream, err := c.servClient.Replay(ctx)
	if err != nil {
		return log.Err(ctx, err, "Gettting replay stream client")
	}
	log.W(ctx, "Replay stream opened")
	idReq := service.ReplayRequest{
		Req: &service.ReplayRequest_ReplayId{ReplayId: id},
	}
	err = replayStream.Send(&idReq)
	if err != nil {
		return log.Err(ctx, err, "Sending replay id")
	}
	log.W(ctx, "Replay id sent")
	res, err := replayStream.Recv()
	if err != nil {
		return log.Err(ctx, err, "Expecting payload request")
	}
	if res.GetPayloadRequest() == nil {
		return fmt.Errorf("Expected payload request, actual got: '%T'", res.Res)
	}
	log.W(ctx, "payload request received")
	payloadReq := service.ReplayRequest{
		Req: &service.ReplayRequest_Payload{
			Payload: (*service.Payload)(&payload),
		},
	}
	err = replayStream.Send(&payloadReq)
	if err != nil {
		return log.Err(ctx, err, "Sending replay payload")
	}
	log.W(ctx, "payload sent")
	c.stream = replayStream
	return nil
}
