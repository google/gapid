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
	"context"

	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/log/log_pb"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/gapis/api"
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

func (c *client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &service.PingRequest{})
	return err
}

func (c *client) GetServerInfo(ctx context.Context) (*service.ServerInfo, error) {
	res, err := c.client.GetServerInfo(ctx, &service.GetServerInfoRequest{})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetInfo(), nil
}

func (c *client) CheckForUpdates(ctx context.Context, includePrereleases bool) (*service.Release, error) {
	res, err := c.client.CheckForUpdates(ctx, &service.CheckForUpdatesRequest{
		IncludePrereleases: includePrereleases,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetRelease(), nil
}

func (c *client) Get(ctx context.Context, p *path.Any) (interface{}, error) {
	res, err := c.client.Get(ctx, &service.GetRequest{Path: p})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetValue().Get(), nil
}

func (c *client) Set(ctx context.Context, p *path.Any, v interface{}) (*path.Any, error) {
	res, err := c.client.Set(ctx, &service.SetRequest{
		Path:  p,
		Value: service.NewValue(v),
	})
	if err != nil {
		return nil, err
	}
	return res.GetPath(), nil
}

func (c *client) Follow(ctx context.Context, p *path.Any) (*path.Any, error) {
	res, err := c.client.Follow(ctx, &service.FollowRequest{Path: p})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetPath(), nil
}

func (c *client) BeginCPUProfile(ctx context.Context) error {
	res, err := c.client.BeginCPUProfile(ctx, &service.BeginCPUProfileRequest{})
	if err != nil {
		return err
	}
	if err := res.GetError(); err != nil {
		return err.Get()
	}
	return nil
}

func (c *client) EndCPUProfile(ctx context.Context) ([]byte, error) {
	res, err := c.client.EndCPUProfile(ctx, &service.EndCPUProfileRequest{})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetData(), nil
}

func (c *client) GetPerformanceCounters(ctx context.Context) (string, error) {
	res, err := c.client.GetPerformanceCounters(ctx, &service.GetPerformanceCountersRequest{})
	if err != nil {
		return "", err
	}
	if err := res.GetError(); err != nil {
		return "", err.Get()
	}
	return res.GetData(), nil
}

func (c *client) GetProfile(ctx context.Context, name string, debug int32) ([]byte, error) {
	res, err := c.client.GetProfile(ctx, &service.GetProfileRequest{
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

func (c *client) GetAvailableStringTables(ctx context.Context) ([]*stringtable.Info, error) {
	res, err := c.client.GetAvailableStringTables(ctx, &service.GetAvailableStringTablesRequest{})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetTables().List, nil
}

func (c *client) GetStringTable(ctx context.Context, i *stringtable.Info) (*stringtable.StringTable, error) {
	res, err := c.client.GetStringTable(ctx, &service.GetStringTableRequest{
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

func (c *client) ImportCapture(ctx context.Context, name string, data []byte) (*path.Capture, error) {
	res, err := c.client.ImportCapture(ctx, &service.ImportCaptureRequest{
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

func (c *client) ExportCapture(ctx context.Context, p *path.Capture) ([]byte, error) {
	res, err := c.client.ExportCapture(ctx, &service.ExportCaptureRequest{
		Capture: p,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetData(), nil
}

func (c *client) LoadCapture(ctx context.Context, path string) (*path.Capture, error) {
	res, err := c.client.LoadCapture(ctx, &service.LoadCaptureRequest{
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

func (c *client) GetDevices(ctx context.Context) ([]*path.Device, error) {
	res, err := c.client.GetDevices(ctx, &service.GetDevicesRequest{})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetDevices().List, nil
}

func (c *client) GetDevicesForReplay(ctx context.Context, p *path.Capture) ([]*path.Device, error) {
	res, err := c.client.GetDevicesForReplay(ctx, &service.GetDevicesForReplayRequest{
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
	ctx context.Context,
	dev *path.Device,
	cmd *path.Command,
	att api.FramebufferAttachment,
	rs *service.RenderSettings,
	hints *service.UsageHints) (*path.ImageInfo, error) {

	res, err := c.client.GetFramebufferAttachment(ctx, &service.GetFramebufferAttachmentRequest{
		Device:     dev,
		After:      cmd,
		Attachment: att,
		Settings:   rs,
		Hints:      hints,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetImage(), nil
}

func (c *client) GetLogStream(ctx context.Context, handler log.Handler) error {
	stream, err := c.client.GetLogStream(ctx, &service.GetLogStreamRequest{})
	if err != nil {
		return err
	}
	h := func(ctx context.Context, m *log_pb.Message) error {
		handler.Handle(m.Message())
		return nil
	}
	return event.Feed(ctx, event.AsHandler(ctx, h), grpcutil.ToProducer(stream))
}

func (c *client) Find(ctx context.Context, req *service.FindRequest, handler service.FindHandler) error {
	stream, err := c.client.Find(ctx, req)
	if err != nil {
		return err
	}
	h := func(ctx context.Context, m *service.FindResponse) error { return handler(m) }
	return event.Feed(ctx, event.AsHandler(ctx, h), grpcutil.ToProducer(stream))
}
