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
	"io"
	"time"

	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/log/log_pb"
	"github.com/google/gapid/core/net/grpcutil"
	perfetto "github.com/google/gapid/gapis/perfetto/service"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
	"github.com/pkg/errors"
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

func (c *client) CheckForUpdates(ctx context.Context, includeDevReleases bool) (*service.Releases, error) {
	res, err := c.client.CheckForUpdates(ctx, &service.CheckForUpdatesRequest{
		IncludeDevReleases: includeDevReleases,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetReleases(), nil
}

func (c *client) Get(ctx context.Context, p *path.Any, r *path.ResolveConfig) (interface{}, error) {
	res, err := c.client.Get(ctx, &service.GetRequest{
		Path:   p,
		Config: r,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetValue().Get(), nil
}

func (c *client) Set(ctx context.Context, p *path.Any, v interface{}, r *path.ResolveConfig) (*path.Any, error) {
	res, err := c.client.Set(ctx, &service.SetRequest{
		Path:   p,
		Value:  service.NewValue(v),
		Config: r,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetPath(), nil
}

func (c *client) Delete(ctx context.Context, p *path.Any, r *path.ResolveConfig) (*path.Any, error) {
	res, err := c.client.Delete(ctx, &service.DeleteRequest{
		Path:   p,
		Config: r,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetPath(), nil
}

func (c *client) Follow(ctx context.Context, p *path.Any, r *path.ResolveConfig) (*path.Any, error) {
	res, err := c.client.Follow(ctx, &service.FollowRequest{
		Path:   p,
		Config: r,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetPath(), nil
}

func (c *client) Profile(
	ctx context.Context,
	pprof, trace io.Writer,
	memorySnapshotInterval uint32,
) (stop func() error, err error) {

	stream, err := c.client.Profile(ctx)
	if err != nil {
		return nil, err
	}

	req := &service.ProfileRequest{MemorySnapshotInterval: memorySnapshotInterval}
	if pprof != nil {
		req.Pprof = true
	}
	if trace != nil {
		req.Trace = true
	}

	if err := stream.Send(req); err != nil {
		return nil, err
	}

	waitForEOF := task.Async(ctx, func(ctx context.Context) error {
		for {
			r, err := stream.Recv()
			if err != nil {
				if errors.Cause(err) == io.EOF {
					return nil
				}
				return err
			}
			if err := r.GetError(); err != nil {
				return err.Get()
			}
			if len(r.Pprof) > 0 && pprof != nil {
				pprof.Write(r.Pprof)
			}
			if len(r.Trace) > 0 && trace != nil {
				trace.Write(r.Trace)
			}
		}
	})

	stop = func() error {
		// Tell the server we want to stop profiling.
		if err := stream.Send(&service.ProfileRequest{}); err != nil {
			return err
		}
		return waitForEOF()
	}

	return stop, nil
}

func (c *client) Status(
	ctx context.Context, snapshotInterval time.Duration, statusUpdateFrequency time.Duration, f func(*service.TaskUpdate), m func(*service.MemoryStatus), rs func(t *service.ReplayUpdate)) error {

	req := &service.ServerStatusRequest{MemorySnapshotInterval: float32(snapshotInterval.Seconds()), StatusUpdateFrequency: float32(statusUpdateFrequency.Seconds())}

	stream, err := c.client.Status(ctx, req)
	if err != nil {
		return err
	}

	for {
		r, err := stream.Recv()
		if err != nil {
			if errors.Cause(err) == io.EOF {
				return nil
			}
			return err
		}
		if _, ok := r.Res.(*service.ServerStatusResponse_Task); ok {
			if f != nil {
				f(r.GetTask())
			}
		} else if _, ok := r.Res.(*service.ServerStatusResponse_Memory); ok {
			if m != nil {
				m(r.GetMemory())
			}
		} else if _, ok := r.Res.(*service.ServerStatusResponse_Replay); ok {
			if rs != nil {
				rs(r.GetReplay())
			}
		}
	}

	return nil
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

func (c *client) SaveCapture(ctx context.Context, capture *path.Capture, path string) error {
	res, err := c.client.SaveCapture(ctx, &service.SaveCaptureRequest{
		Capture: capture,
		Path:    path,
	})
	if err != nil {
		return err
	}
	if err := res.GetError(); err != nil {
		return err.Get()
	}
	return nil
}

func (c *client) ExportReplay(ctx context.Context, capture *path.Capture, device *path.Device, path string, opts *service.ExportReplayOptions) error {
	res, err := c.client.ExportReplay(ctx, &service.ExportReplayRequest{
		Capture: capture,
		Path:    path,
		Device:  device,
		Options: opts,
	})
	if err != nil {
		return err
	}
	if err := res.GetError(); err != nil {
		return err.Get()
	}
	return nil
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

func (c *client) GetDevicesForReplay(ctx context.Context, p *path.Capture) ([]*path.Device, []bool, []*stringtable.Msg, error) {
	res, err := c.client.GetDevicesForReplay(ctx, &service.GetDevicesForReplayRequest{
		Capture: p,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, nil, nil, err.Get()
	}
	return res.GetDevices().List, res.GetDevices().Compatibilities, res.GetDevices().Reasons, nil
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

func (c *client) ClientEvent(ctx context.Context, req *service.ClientEventRequest) error {
	_, err := c.client.ClientEvent(ctx, req)
	return err
}

// TraceTargetTreeNode returns a node in the trace target tree for the given device
func (c *client) TraceTargetTreeNode(ctx context.Context, req *service.TraceTargetTreeNodeRequest) (*service.TraceTargetTreeNode, error) {
	res, err := c.client.TraceTargetTreeNode(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetNode(), nil
}

// FindTraceTargets returns trace targets matching the given search parameters.
func (c *client) FindTraceTargets(ctx context.Context, req *service.FindTraceTargetsRequest) ([]*service.TraceTargetTreeNode, error) {
	res, err := c.client.FindTraceTargets(ctx, req)

	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetNodes().GetNodes(), nil
}

type traceHandler struct {
	conn service.Gapid_TraceClient
}

func (c *client) Trace(ctx context.Context) (service.TraceHandler, error) {
	res, err := c.client.Trace(ctx)
	if err != nil {
		return nil, err
	}
	return &traceHandler{res}, nil
}

func (t *traceHandler) Initialize(ctx context.Context, opts *service.TraceOptions) (*service.StatusResponse, error) {
	err := t.conn.Send(
		&service.TraceRequest{
			Action: &service.TraceRequest_Initialize{
				Initialize: opts,
			}},
	)
	if err != nil {
		return nil, err
	}
	res, err := t.conn.Recv()
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetStatus(), nil
}

func (t *traceHandler) Event(ctx context.Context, evt service.TraceEvent) (*service.StatusResponse, error) {
	err := t.conn.Send(
		&service.TraceRequest{
			Action: &service.TraceRequest_QueryEvent{
				QueryEvent: evt,
			}},
	)
	if err != nil {
		return nil, err
	}
	res, err := t.conn.Recv()
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetStatus(), nil
}

func (t *traceHandler) Dispose(ctx context.Context) {
	t.conn.CloseSend()
}

func (c *client) DCECapture(ctx context.Context, capture *path.Capture, commands []*path.Command) (*path.Capture, error) {
	res, err := c.client.DCECapture(ctx, &service.DCECaptureRequest{
		Capture:  capture,
		Commands: commands,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetCapture(), nil
}

func (c *client) SplitCapture(ctx context.Context, rng *path.Commands) (*path.Capture, error) {
	res, err := c.client.SplitCapture(ctx, &service.SplitCaptureRequest{
		Commands: rng,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetCapture(), nil
}

func (c *client) UpdateSettings(ctx context.Context, req *service.UpdateSettingsRequest) error {
	res, err := c.client.UpdateSettings(ctx, req)
	if err != nil {
		return err
	}
	if err := res.GetError(); err != nil {
		return err.Get()
	}
	return nil
}

func (c *client) GpuProfile(ctx context.Context, req *service.GpuProfileRequest) (*service.ProfilingData, error) {
	res, err := c.client.GpuProfile(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetProfilingData(), nil
}

func (c *client) GetTimestamps(ctx context.Context, req *service.GetTimestampsRequest, handler service.TimeStampsHandler) error {
	stream, err := c.client.GetTimestamps(ctx, req)
	if err != nil {
		return err
	}
	h := func(ctx context.Context, m *service.GetTimestampsResponse) error { return handler(m) }
	return event.Feed(ctx, event.AsHandler(ctx, h), grpcutil.ToProducer(stream))
}

func (c *client) GetGraphVisualization(ctx context.Context, capture *path.Capture, format service.GraphFormat) ([]byte, error) {
	res, err := c.client.GetGraphVisualization(ctx, &service.GraphVisualizationRequest{
		Capture: capture,
		Format:  format,
	})
	if err != nil {
		return []byte{}, err
	}
	if err := res.GetError(); err != nil {
		return []byte{}, err.Get()
	}
	return res.GetGraphVisualization(), nil
}

func (c *client) PerfettoQuery(ctx context.Context, capture *path.Capture, query string) (*perfetto.QueryResult, error) {
	res, err := c.client.PerfettoQuery(ctx, &service.PerfettoQueryRequest{
		Capture: capture,
		Query:   query,
	})
	if err != nil {
		return nil, err
	}
	if err := res.GetError(); err != nil {
		return nil, err.Get()
	}
	return res.GetResult(), nil
}

func (c *client) ValidateDevice(ctx context.Context, device *path.Device) error {
	res, err := c.client.ValidateDevice(ctx, &service.ValidateDeviceRequest{
		Device: device,
	})
	if err != nil {
		return err
	}
	if err := res.GetError(); err != nil {
		return err.Get()
	}
	return nil
}

func (c *client) InstallApp(ctx context.Context, d *path.Device, app string) error {
	res, err := c.client.InstallApp(ctx, &service.InstallAppRequest{
		Device:      d,
		Application: app,
	})
	if err != nil {
		return err
	}
	if err := res.GetError(); err != nil {
		return err.Get()
	}
	return nil
}
