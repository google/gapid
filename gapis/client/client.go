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

package client

import (
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary/schema"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
	"google.golang.org/grpc"
)

// Client is the interface used for service calls.
type Client interface {
	service.Service

	// Close closes the client connection.
	Close() error
}

// Bind creates a new rpc client using conn for communication.
func Bind(conn *grpc.ClientConn) Client {
	return &client{service.NewGapidClient(conn), conn.Close}
}

// New creates a new client using c for communication.
func New(c service.GapidClient) service.Service {
	return &client{c, func() error { return nil }}
}

type client struct {
	client service.GapidClient
	close  func() error
}

func (c *client) Close() error { return c.close() }

func (c *client) GetServerInfo(ctx log.Context) (*service.ServerInfo, error) {
	res, err := c.client.GetServerInfo(ctx.Unwrap(), &service.GetServerInfoRequest{})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetInfo(), nil
}

func (c *client) Get(ctx log.Context, p *path.Any) (interface{}, error) {
	res, err := c.client.Get(ctx.Unwrap(), &service.GetRequest{Path: p})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetValue().Get(), nil
}

func (c *client) Set(ctx log.Context, p *path.Any, v interface{}) (*path.Any, error) {
	res, err := c.client.Set(ctx.Unwrap(), &service.SetRequest{
		Path:  p,
		Value: service.NewValue(v),
	})
	if err != nil {
		return nil, err
	}
	return res.GetPath(), nil
}

func (c *client) Follow(ctx log.Context, p *path.Any) (*path.Any, error) {
	res, err := c.client.Follow(ctx.Unwrap(), &service.FollowRequest{Path: p})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetPath(), nil
}

func (c *client) BeginCPUProfile(ctx log.Context) error {
	res, err := c.client.BeginCPUProfile(ctx.Unwrap(), &service.BeginCPUProfileRequest{})
	if err != nil {
		return err
	}
	if err := res.GetError(); err != nil {
		return err.Get()
	}
	return nil
}

func (c *client) EndCPUProfile(ctx log.Context) ([]byte, error) {
	res, err := c.client.EndCPUProfile(ctx.Unwrap(), &service.EndCPUProfileRequest{})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetData(), nil
}

func (c *client) GetPerformanceCounters(ctx log.Context) ([]byte, error) {
	res, err := c.client.GetPerformanceCounters(ctx.Unwrap(), &service.GetPerformanceCountersRequest{})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetData(), nil
}

func (c *client) GetProfile(ctx log.Context, name string, debug int32) ([]byte, error) {
	res, err := c.client.GetProfile(ctx.Unwrap(), &service.GetProfileRequest{
		Name:  name,
		Debug: debug,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetData(), nil
}

func (c *client) GetSchema(ctx log.Context) (*schema.Message, error) {
	res, err := c.client.GetSchema(ctx.Unwrap(), &service.GetSchemaRequest{})
	if err != nil {
		return nil, err
	}
	obj, err := res.GetObject().Decode()
	if err != nil {
		return nil, err
	}
	return obj.(*schema.Message), nil
}

func (c *client) GetAvailableStringTables(ctx log.Context) ([]*stringtable.Info, error) {
	res, err := c.client.GetAvailableStringTables(ctx.Unwrap(), &service.GetAvailableStringTablesRequest{})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetTables().List, nil
}

func (c *client) GetStringTable(ctx log.Context, i *stringtable.Info) (*stringtable.StringTable, error) {
	res, err := c.client.GetStringTable(ctx.Unwrap(), &service.GetStringTableRequest{
		Table: i,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetTable(), nil
}

func (c *client) ImportCapture(ctx log.Context, name string, data []byte) (*path.Capture, error) {
	res, err := c.client.ImportCapture(ctx.Unwrap(), &service.ImportCaptureRequest{
		Name: name,
		Data: data,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetCapture(), nil
}

func (c *client) LoadCapture(ctx log.Context, path string) (*path.Capture, error) {
	res, err := c.client.LoadCapture(ctx.Unwrap(), &service.LoadCaptureRequest{
		Path: path,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetCapture(), nil
}

func (c *client) GetDevices(ctx log.Context) ([]*path.Device, error) {
	res, err := c.client.GetDevices(ctx.Unwrap(), &service.GetDevicesRequest{})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetDevices().List, nil
}

func (c *client) GetDevicesForReplay(ctx log.Context, p *path.Capture) ([]*path.Device, error) {
	res, err := c.client.GetDevicesForReplay(ctx.Unwrap(), &service.GetDevicesForReplayRequest{
		Capture: p,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetDevices().List, nil
}

func (c *client) GetFramebufferAttachment(
	ctx log.Context,
	dev *path.Device,
	cmd *path.Command,
	att gfxapi.FramebufferAttachment,
	rs *service.RenderSettings) (*path.ImageInfo, error) {

	res, err := c.client.GetFramebufferAttachment(ctx.Unwrap(), &service.GetFramebufferAttachmentRequest{
		Device:     dev,
		After:      cmd,
		Attachment: att,
		Settings:   rs,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetImage(), nil
}
