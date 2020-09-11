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

package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/log/log_pb"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/gapis/service"

	"google.golang.org/grpc"

	xctx "golang.org/x/net/context"
)

// Listen starts a new GRPC server listening on addr.
// This is a blocking call.
func Listen(ctx context.Context, addr string, cfg Config) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.F(ctx, true, "Could not start grpc server at %v: %s", addr, err.Error())
	}
	return NewWithListener(ctx, listener, cfg, nil)
}

// NewWithListener starts a new GRPC server listening on l.
// This is a blocking call.
func NewWithListener(ctx context.Context, l net.Listener, cfg Config, srvChan chan<- *grpc.Server) error {
	s := &grpcServer{
		handler:      New(ctx, cfg),
		bindCtx:      func(c context.Context) context.Context { return keys.Clone(c, ctx) },
		keepAlive:    make(chan struct{}, 1),
		interrupters: map[int]func(){},
	}

	done := make(chan error)
	ctx, stop := task.WithCancel(ctx)
	crash.Go(func() {
		done <- grpcutil.ServeWithListener(ctx, l, func(ctx context.Context, listener net.Listener, server *grpc.Server) error {
			if addr, ok := listener.Addr().(*net.TCPAddr); ok {
				// The following message is parsed by launchers to detect the selected port. DO NOT CHANGE!
				fmt.Printf("Bound on port '%d'\n", addr.Port)
			}
			service.RegisterGapidServer(server, s)
			if srvChan != nil {
				srvChan <- server
			}
			if cfg.IdleTimeout != 0 {
				crash.Go(func() { s.stopIfIdle(ctx, server, cfg.IdleTimeout, stop) })
			} else {
				crash.Go(func() { s.stopOnInterrupt(ctx, server, stop) })
			}
			return nil
		}, grpc.UnaryInterceptor(auth.ServerInterceptor(cfg.AuthToken)))
	})

	select {
	case err := <-done:
		return err
	case <-task.ShouldStop(ctx):
		s.mutex.Lock()
		defer s.mutex.Unlock()
		for _, f := range s.interrupters {
			f()
		}
		return <-done
	}
}

type grpcServer struct {
	handler         Server
	bindCtx         func(context.Context) context.Context
	keepAlive       chan struct{}
	inFlightRPCs    int64
	interrupters    map[int]func()
	lastInterrupter int
	mutex           sync.Mutex
}

// inRPC should be called at the start of an RPC call. The returned function
// should be called when the RPC call finishes.
func (s *grpcServer) inRPC() func() {
	atomic.AddInt64(&s.inFlightRPCs, 1)
	select {
	case s.keepAlive <- struct{}{}:
	default:
	}
	return func() {
		select {
		case s.keepAlive <- struct{}{}:
		default:
		}
		if atomic.LoadInt64(&s.inFlightRPCs) == 0 {
			panic("Should never happen: inFlightRPCs counter is going below zero")
		}
		atomic.AddInt64(&s.inFlightRPCs, -1)
	}
}

// stopIfIdle calls GracefulStop on server if there are no writes the the
// keepAlive chan within idleTimeout or if the current process is interrupted.
// This function blocks until there's an idle timeout, or ctx is cancelled.
func (s *grpcServer) stopIfIdle(ctx context.Context, server *grpc.Server, idleTimeout time.Duration, stop func()) {
	// Split the idleTimeout into N smaller chunks, and check that there was
	// no activity from the client in a contiguous N chunks of time.
	// This avoids killing the server if the machine is suspended (where the
	// client cannot send hearbeats, and the system clock jumps forward).
	waitTime := idleTimeout / 12
	var idleTime time.Duration

	stoppedSignal, stopped := task.NewSignal()
	defer func() {
		stop()
		server.Stop()
		stopped(ctx)
	}()

	// Wait for the server to stop before terminating the app.
	app.AddCleanupSignal(stoppedSignal)

	for {
		select {
		case <-task.ShouldStop(ctx):
			return
		case <-time.After(waitTime):
			if rpcs := atomic.LoadInt64(&s.inFlightRPCs); rpcs != 0 {
				continue
			}
			idleTime += waitTime
			if idleTime >= idleTimeout {
				log.E(ctx, "Stopping GAPIS server: No communication with the client for %v (--idle-timeout %v)", idleTime, idleTimeout)
				time.Sleep(time.Second * 3) // Wait a little in the hope this message makes its way to the client(s).
				return
			}
			log.W(ctx, "No communication with the client for %v (--idle-timeout %v)", idleTime, idleTimeout)
		case <-s.keepAlive:
			idleTime = 0
		}
	}
}

// stopOnInterrupt calls GracefulStop on server if the current process is interrupted.
func (s *grpcServer) stopOnInterrupt(ctx context.Context, server *grpc.Server, stop func()) {
	stoppedSignal, stopped := task.NewSignal()
	defer func() {
		stop()
		server.Stop()
		stopped(ctx)
	}()

	// Wait for the server to stop before terminating the app.
	app.AddCleanupSignal(stoppedSignal)

	<-task.ShouldStop(ctx)
}

func (s *grpcServer) addInterrupter(f func()) (remove func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	li := s.lastInterrupter
	s.lastInterrupter++
	s.interrupters[li] = f
	return func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		delete(s.interrupters, li)
	}
}

func (s *grpcServer) Ping(ctx xctx.Context, req *service.PingRequest) (*service.PingResponse, error) {
	defer s.inRPC()()
	err := s.handler.Ping(s.bindCtx(ctx))
	if err := service.NewError(err); err != nil {
		return &service.PingResponse{}, nil
	}
	return &service.PingResponse{}, nil
}

func (s *grpcServer) GetServerInfo(ctx xctx.Context, req *service.GetServerInfoRequest) (*service.GetServerInfoResponse, error) {
	defer s.inRPC()()
	info, err := s.handler.GetServerInfo(s.bindCtx(ctx))
	if err := service.NewError(err); err != nil {
		return &service.GetServerInfoResponse{Res: &service.GetServerInfoResponse_Error{Error: err}}, nil
	}
	return &service.GetServerInfoResponse{Res: &service.GetServerInfoResponse_Info{Info: info}}, nil
}

func (s *grpcServer) CheckForUpdates(ctx xctx.Context, req *service.CheckForUpdatesRequest) (*service.CheckForUpdatesResponse, error) {
	defer s.inRPC()()
	release, err := s.handler.CheckForUpdates(s.bindCtx(ctx), req.IncludeDevReleases)
	if err := service.NewError(err); err != nil {
		return &service.CheckForUpdatesResponse{Res: &service.CheckForUpdatesResponse_Error{Error: err}}, nil
	}
	return &service.CheckForUpdatesResponse{Res: &service.CheckForUpdatesResponse_Release{Release: release}}, nil
}

func (s *grpcServer) Get(ctx xctx.Context, req *service.GetRequest) (*service.GetResponse, error) {
	defer s.inRPC()()
	res, err := s.handler.Get(s.bindCtx(ctx), req.Path, req.Config)
	if err := service.NewError(err); err != nil {
		return &service.GetResponse{Res: &service.GetResponse_Error{Error: err}}, nil
	}
	val := service.NewValue(res)
	return &service.GetResponse{Res: &service.GetResponse_Value{Value: val}}, nil
}

func (s *grpcServer) Set(ctx xctx.Context, req *service.SetRequest) (*service.SetResponse, error) {
	defer s.inRPC()()
	res, err := s.handler.Set(s.bindCtx(ctx), req.Path, req.Value.Get(), req.Config)
	if err := service.NewError(err); err != nil {
		return &service.SetResponse{Res: &service.SetResponse_Error{Error: err}}, nil
	}
	return &service.SetResponse{Res: &service.SetResponse_Path{Path: res}}, nil
}

func (s *grpcServer) Delete(ctx xctx.Context, req *service.DeleteRequest) (*service.DeleteResponse, error) {
	defer s.inRPC()()
	res, err := s.handler.Delete(s.bindCtx(ctx), req.Path, req.Config)
	if err := service.NewError(err); err != nil {
		return &service.DeleteResponse{Res: &service.DeleteResponse_Error{Error: err}}, nil
	}
	return &service.DeleteResponse{Res: &service.DeleteResponse_Path{Path: res}}, nil
}

func (s *grpcServer) Follow(ctx xctx.Context, req *service.FollowRequest) (*service.FollowResponse, error) {
	defer s.inRPC()()
	res, err := s.handler.Follow(s.bindCtx(ctx), req.Path, req.Config)
	if err := service.NewError(err); err != nil {
		return &service.FollowResponse{Res: &service.FollowResponse_Error{Error: err}}, nil
	}
	return &service.FollowResponse{Res: &service.FollowResponse_Path{Path: res}}, nil
}

type syncBuffer struct {
	bytes.Buffer
	sync.Mutex
}

func (b *syncBuffer) Write(p []byte) (n int, err error) {
	b.Lock()
	defer b.Unlock()
	return b.Buffer.Write(p)
}

func (s *grpcServer) Profile(stream service.Gapid_ProfileServer) error {
	defer s.inRPC()()
	ctx := s.bindCtx(stream.Context())

	// stop stops any running profiles, waiting for them to complete.
	var stop func() error

	// flush writes out any pending data to pprofBuf and traceBuf.
	var pprofBuf, traceBuf syncBuffer
	flush := func() error {
		pprofBuf.Lock()
		traceBuf.Lock()
		defer pprofBuf.Unlock()
		defer traceBuf.Unlock()

		if len(pprofBuf.Bytes()) == 0 && len(traceBuf.Bytes()) == 0 {
			return nil // Nothing to send.
		}

		err := stream.Send(&service.ProfileResponse{
			Pprof: pprofBuf.Bytes(),
			Trace: traceBuf.Bytes(),
		})
		if err != nil {
			return log.Err(ctx, err, "stream.Send")
		}
		pprofBuf.Reset()
		traceBuf.Reset()
		return nil
	}

	// Flush the pending data every second.
	stopPolling := task.Async(ctx, func(ctx context.Context) error {
		return task.Poll(ctx, time.Second, func(context.Context) error { return flush() })
	})
	defer stopPolling()

	// sendErr attempts to send the err as a response. If succesfully sent, then
	// sendErr returns nil, otherwise err.
	sendErr := func(err error) error {
		if err == nil {
			return nil
		}
		if err := service.NewError(err); err != nil {
			if err2 := stream.Send(&service.ProfileResponse{Error: err}); err2 == nil {
				return nil
			}
		}
		return err
	}

	// stopAndFlush stops any running profile, and flushes any pending data.
	stopAndFlush := func() error {
		if stop != nil {
			if err := stop(); err != nil && err != context.Canceled {
				return log.Err(ctx, err, "profile stop")
			}
			stop = nil
		}
		if err := flush(); err != nil {
			return log.Err(ctx, err, "profile flush")
		}
		return nil
	}
	defer stopAndFlush()

	for {
		// Grab an incoming request.
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		// Stop and flush any existing profiles.
		if err := stopAndFlush(); err != nil {
			return sendErr(err)
		}

		// If there are no profile modes in the request, then the RPC can finish.
		if !req.Pprof && !req.Trace {
			return sendErr(stopAndFlush())
		}

		var pprof, trace io.Writer
		if req.Pprof {
			pprof = &pprofBuf
		}
		if req.Trace {
			trace = &traceBuf
		}

		// Start the profile.
		stop, err = s.handler.Profile(ctx, pprof, trace, req.MemorySnapshotInterval)
		if err != nil {
			return err
		}
	}
}

func (s *grpcServer) Status(req *service.ServerStatusRequest, stream service.Gapid_StatusServer) error {
	// defer s.inRPC()() -- don't consider the log stream an inflight RPC.
	ctx, cancel := task.WithCancel(stream.Context())
	defer s.addInterrupter(cancel)()

	c := make(chan error)
	f := func(t *service.TaskUpdate) {
		if err := stream.Send(&service.ServerStatusResponse{
			Res: &service.ServerStatusResponse_Task{t},
		}); err != nil {
			c <- err
			cancel()
		}
	}
	m := func(t *service.MemoryStatus) {
		if err := stream.Send(&service.ServerStatusResponse{
			Res: &service.ServerStatusResponse_Memory{t},
		}); err != nil {
			c <- err
			cancel()
		}
	}
	r := func(t *service.ReplayUpdate) {
		if err := stream.Send(&service.ServerStatusResponse{
			Res: &service.ServerStatusResponse_Replay{t},
		}); err != nil {
			c <- err
			cancel()
		}
	}
	err := s.handler.Status(s.bindCtx(ctx),
		time.Duration(float32(time.Second)*req.MemorySnapshotInterval),
		time.Duration(float32(time.Second)*req.StatusUpdateFrequency),
		f, m, r)

	if err == nil {
		select {
		case err = <-c:
		}
	}
	return err
}

func (s *grpcServer) GetPerformanceCounters(ctx xctx.Context, req *service.GetPerformanceCountersRequest) (*service.GetPerformanceCountersResponse, error) {
	defer s.inRPC()()
	data, err := s.handler.GetPerformanceCounters(s.bindCtx(ctx))
	if err := service.NewError(err); err != nil {
		return &service.GetPerformanceCountersResponse{Res: &service.GetPerformanceCountersResponse_Error{Error: err}}, nil
	}
	return &service.GetPerformanceCountersResponse{Res: &service.GetPerformanceCountersResponse_Data{Data: data}}, nil
}

func (s *grpcServer) GetProfile(ctx xctx.Context, req *service.GetProfileRequest) (*service.GetProfileResponse, error) {
	defer s.inRPC()()
	data, err := s.handler.GetProfile(s.bindCtx(ctx), req.Name, req.Debug)
	if err := service.NewError(err); err != nil {
		return &service.GetProfileResponse{Res: &service.GetProfileResponse_Error{Error: err}}, nil
	}
	return &service.GetProfileResponse{Res: &service.GetProfileResponse_Data{Data: data}}, nil
}

func (s *grpcServer) GetAvailableStringTables(ctx xctx.Context, req *service.GetAvailableStringTablesRequest) (*service.GetAvailableStringTablesResponse, error) {
	defer s.inRPC()()
	tables, err := s.handler.GetAvailableStringTables(s.bindCtx(ctx))
	if err := service.NewError(err); err != nil {
		return &service.GetAvailableStringTablesResponse{Res: &service.GetAvailableStringTablesResponse_Error{Error: err}}, nil
	}
	return &service.GetAvailableStringTablesResponse{
		Res: &service.GetAvailableStringTablesResponse_Tables{
			Tables: &service.StringTableInfos{List: tables},
		},
	}, nil
}

func (s *grpcServer) GetStringTable(ctx xctx.Context, req *service.GetStringTableRequest) (*service.GetStringTableResponse, error) {
	defer s.inRPC()()
	table, err := s.handler.GetStringTable(s.bindCtx(ctx), req.Table)
	if err := service.NewError(err); err != nil {
		return &service.GetStringTableResponse{Res: &service.GetStringTableResponse_Error{Error: err}}, nil
	}
	return &service.GetStringTableResponse{Res: &service.GetStringTableResponse_Table{Table: table}}, nil
}

func (s *grpcServer) ImportCapture(ctx xctx.Context, req *service.ImportCaptureRequest) (*service.ImportCaptureResponse, error) {
	defer s.inRPC()()
	capture, err := s.handler.ImportCapture(s.bindCtx(ctx), req.Name, req.Data)
	if err := service.NewError(err); err != nil {
		return &service.ImportCaptureResponse{Res: &service.ImportCaptureResponse_Error{Error: err}}, nil
	}
	return &service.ImportCaptureResponse{Res: &service.ImportCaptureResponse_Capture{Capture: capture}}, nil
}

func (s *grpcServer) ExportCapture(ctx xctx.Context, req *service.ExportCaptureRequest) (*service.ExportCaptureResponse, error) {
	defer s.inRPC()()
	data, err := s.handler.ExportCapture(s.bindCtx(ctx), req.Capture)
	if err := service.NewError(err); err != nil {
		return &service.ExportCaptureResponse{Res: &service.ExportCaptureResponse_Error{Error: err}}, nil
	}
	return &service.ExportCaptureResponse{Res: &service.ExportCaptureResponse_Data{Data: data}}, nil
}

func (s *grpcServer) LoadCapture(ctx xctx.Context, req *service.LoadCaptureRequest) (*service.LoadCaptureResponse, error) {
	defer s.inRPC()()
	capture, err := s.handler.LoadCapture(s.bindCtx(ctx), req.Path)
	if err := service.NewError(err); err != nil {
		return &service.LoadCaptureResponse{Res: &service.LoadCaptureResponse_Error{Error: err}}, nil
	}
	return &service.LoadCaptureResponse{Res: &service.LoadCaptureResponse_Capture{Capture: capture}}, nil
}

func (s *grpcServer) SaveCapture(ctx xctx.Context, req *service.SaveCaptureRequest) (*service.SaveCaptureResponse, error) {
	defer s.inRPC()()
	err := s.handler.SaveCapture(s.bindCtx(ctx), req.Capture, req.Path)
	if err := service.NewError(err); err != nil {
		return &service.SaveCaptureResponse{Error: err}, nil
	}
	return &service.SaveCaptureResponse{}, nil
}

func (s *grpcServer) ExportReplay(ctx xctx.Context, req *service.ExportReplayRequest) (*service.ExportReplayResponse, error) {
	defer s.inRPC()()
	err := s.handler.ExportReplay(s.bindCtx(ctx), req.Capture, req.Device, req.Path, req.Options)
	if err := service.NewError(err); err != nil {
		return &service.ExportReplayResponse{Error: err}, nil
	}
	return &service.ExportReplayResponse{}, nil
}

func (s *grpcServer) DCECapture(ctx xctx.Context, req *service.DCECaptureRequest) (*service.DCECaptureResponse, error) {
	defer s.inRPC()()
	capture, err := s.handler.DCECapture(s.bindCtx(ctx), req.Capture, req.Commands)
	if err := service.NewError(err); err != nil {
		return &service.DCECaptureResponse{Res: &service.DCECaptureResponse_Error{Error: err}}, nil
	}
	return &service.DCECaptureResponse{Res: &service.DCECaptureResponse_Capture{Capture: capture}}, nil
}

func (s *grpcServer) GetGraphVisualization(ctx xctx.Context, req *service.GraphVisualizationRequest) (*service.GraphVisualizationResponse, error) {
	defer s.inRPC()()
	graphVisualization, err := s.handler.GetGraphVisualization(s.bindCtx(ctx), req.Capture, req.Format)
	if err := service.NewError(err); err != nil {
		return &service.GraphVisualizationResponse{Res: &service.GraphVisualizationResponse_Error{Error: err}}, nil
	}
	return &service.GraphVisualizationResponse{Res: &service.GraphVisualizationResponse_GraphVisualization{GraphVisualization: graphVisualization}}, nil
}

func (s *grpcServer) GetDevices(ctx xctx.Context, req *service.GetDevicesRequest) (*service.GetDevicesResponse, error) {
	defer s.inRPC()()
	devices, err := s.handler.GetDevices(s.bindCtx(ctx))
	if err := service.NewError(err); err != nil {
		return &service.GetDevicesResponse{Res: &service.GetDevicesResponse_Error{Error: err}}, nil
	}
	return &service.GetDevicesResponse{
		Res: &service.GetDevicesResponse_Devices{
			Devices: &service.Devices{List: devices},
		},
	}, nil
}

func (s *grpcServer) GetDevicesForReplay(ctx xctx.Context, req *service.GetDevicesForReplayRequest) (*service.GetDevicesForReplayResponse, error) {
	defer s.inRPC()()
	devices, err := s.handler.GetDevicesForReplay(s.bindCtx(ctx), req.Capture)
	if err := service.NewError(err); err != nil {
		return &service.GetDevicesForReplayResponse{Res: &service.GetDevicesForReplayResponse_Error{Error: err}}, nil
	}
	return &service.GetDevicesForReplayResponse{
		Res: &service.GetDevicesForReplayResponse_Devices{
			Devices: &service.Devices{List: devices},
		},
	}, nil
}

func (s *grpcServer) GetLogStream(req *service.GetLogStreamRequest, server service.Gapid_GetLogStreamServer) error {
	// defer s.inRPC()() -- don't consider the log stream an inflight RPC.
	ctx, cancel := task.WithCancel(server.Context())
	defer s.addInterrupter(cancel)()

	h := log.NewHandler(func(m *log.Message) { server.Send(log_pb.From(m)) }, nil)
	return s.handler.GetLogStream(s.bindCtx(ctx), h)
}

func (s *grpcServer) Find(req *service.FindRequest, server service.Gapid_FindServer) error {
	defer s.inRPC()()
	ctx := server.Context()
	return s.handler.Find(s.bindCtx(ctx), req, server.Send)
}

func (s *grpcServer) GpuProfile(ctx xctx.Context, req *service.GpuProfileRequest) (*service.GpuProfileResponse, error) {
	defer s.inRPC()()
	res, err := s.handler.GpuProfile(s.bindCtx(ctx), req)
	if err := service.NewError(err); err != nil {
		return &service.GpuProfileResponse{Res: &service.GpuProfileResponse_Error{Error: err}}, nil
	}
	return &service.GpuProfileResponse{Res: &service.GpuProfileResponse_ProfilingData{ProfilingData: res}}, nil
}

func (s *grpcServer) UpdateSettings(ctx xctx.Context, req *service.UpdateSettingsRequest) (*service.UpdateSettingsResponse, error) {
	defer s.inRPC()()
	err := s.handler.UpdateSettings(s.bindCtx(ctx), req)
	if err := service.NewError(err); err != nil {
		return &service.UpdateSettingsResponse{Error: err}, nil
	}
	return &service.UpdateSettingsResponse{}, nil
}

func (s *grpcServer) ClientEvent(ctx xctx.Context, req *service.ClientEventRequest) (*service.ClientEventResponse, error) {
	defer s.inRPC()()
	err := s.handler.ClientEvent(s.bindCtx(ctx), req)
	if err != nil {
		return nil, err
	}
	return &service.ClientEventResponse{}, nil
}

func (s *grpcServer) SplitCapture(ctx xctx.Context, req *service.SplitCaptureRequest) (*service.SplitCaptureResponse, error) {
	defer s.inRPC()()
	res, err := s.handler.SplitCapture(s.bindCtx(ctx), req.Commands)
	if err := service.NewError(err); err != nil {
		return &service.SplitCaptureResponse{Res: &service.SplitCaptureResponse_Error{Error: err}}, nil
	}
	return &service.SplitCaptureResponse{Res: &service.SplitCaptureResponse_Capture{Capture: res}}, nil
}

func (s *grpcServer) TraceTargetTreeNode(ctx xctx.Context, req *service.TraceTargetTreeNodeRequest) (*service.TraceTargetTreeNodeResponse, error) {
	defer s.inRPC()()
	res, err := s.handler.TraceTargetTreeNode(s.bindCtx(ctx), req)
	if err := service.NewError(err); err != nil {
		return &service.TraceTargetTreeNodeResponse{Val: &service.TraceTargetTreeNodeResponse_Error{Error: err}}, nil
	}
	return &service.TraceTargetTreeNodeResponse{Val: &service.TraceTargetTreeNodeResponse_Node{Node: res}}, nil
}

func (s *grpcServer) FindTraceTargets(ctx xctx.Context, req *service.FindTraceTargetsRequest) (*service.FindTraceTargetsResponse, error) {
	defer s.inRPC()()
	res, err := s.handler.FindTraceTargets(s.bindCtx(ctx), req)
	if err := service.NewError(err); err != nil {
		return &service.FindTraceTargetsResponse{Val: &service.FindTraceTargetsResponse_Error{Error: err}}, nil
	}
	return &service.FindTraceTargetsResponse{
		Val: &service.FindTraceTargetsResponse_Nodes{
			Nodes: &service.TraceTargetTreeNodes{
				Nodes: res,
			},
		},
	}, nil
}

func (s *grpcServer) Trace(conn service.Gapid_TraceServer) error {
	ctx := s.bindCtx(conn.Context())
	ctx = status.Start(ctx, "Tracing")
	defer status.Finish(ctx)
	t, err := s.handler.Trace(ctx)
	if err != nil {
		return err
	}
	defer t.Dispose(ctx)

	for {
		req, err := conn.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		switch r := req.Action.(type) {
		case *service.TraceRequest_Initialize:
			resp, err := t.Initialize(ctx, r.Initialize)
			if err := service.NewError(err); err != nil {
				r := service.TraceResponse{Res: &service.TraceResponse_Error{
					Error: err,
				}}
				conn.Send(&r)
				return nil
			}
			conn.Send(&service.TraceResponse{Res: &service.TraceResponse_Status{Status: resp}})
		case *service.TraceRequest_QueryEvent:
			resp, err := t.Event(ctx, r.QueryEvent)
			if err := service.NewError(err); err != nil {
				r := service.TraceResponse{Res: &service.TraceResponse_Error{
					Error: err,
				}}
				conn.Send(&r)
				return nil
			}
			conn.Send(&service.TraceResponse{Res: &service.TraceResponse_Status{Status: resp}})
		}
	}
	return nil
}

func (s *grpcServer) GetTimestamps(req *service.GetTimestampsRequest, server service.Gapid_GetTimestampsServer) error {
	defer s.inRPC()()
	ctx := server.Context()
	return s.handler.GetTimestamps(s.bindCtx(ctx), req, server.Send)
}

func (s *grpcServer) PerfettoQuery(ctx xctx.Context, req *service.PerfettoQueryRequest) (*service.PerfettoQueryResponse, error) {
	data, err := s.handler.PerfettoQuery(s.bindCtx(ctx), req.Capture, req.Query)
	if err := service.NewError(err); err != nil {
		return &service.PerfettoQueryResponse{Res: &service.PerfettoQueryResponse_Error{Error: err}}, nil
	}
	return &service.PerfettoQueryResponse{Res: &service.PerfettoQueryResponse_Result{Result: data}}, nil
}

func (s *grpcServer) ValidateDevice(ctx xctx.Context, req *service.ValidateDeviceRequest) (*service.ValidateDeviceResponse, error) {
	err := s.handler.ValidateDevice(s.bindCtx(ctx), req.Device)
	if err := service.NewError(err); err != nil {
		return &service.ValidateDeviceResponse{Error: err}, nil
	}
	return &service.ValidateDeviceResponse{}, nil
}
