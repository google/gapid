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
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/crash"
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
		log.F(ctx, "Could not start grpc server at %v: %s", addr, err.Error())
	}
	return NewWithListener(ctx, listener, cfg, nil)
}

// NewWithListener starts a new GRPC server listening on l.
// This is a blocking call.
func NewWithListener(ctx context.Context, l net.Listener, cfg Config, srvChan chan<- *grpc.Server) error {
	s := &grpcServer{
		handler:   New(ctx, cfg),
		bindCtx:   func(c context.Context) context.Context { return keys.Clone(c, ctx) },
		keepAlive: make(chan struct{}, 1),
	}
	return grpcutil.ServeWithListener(ctx, l, func(ctx context.Context, listener net.Listener, server *grpc.Server) error {
		if addr, ok := listener.Addr().(*net.TCPAddr); ok {
			// The following message is parsed by launchers to detect the selected port. DO NOT CHANGE!
			fmt.Printf("Bound on port '%d'\n", addr.Port)
		}
		service.RegisterGapidServer(server, s)
		if srvChan != nil {
			srvChan <- server
		}
		if cfg.IdleTimeout != 0 {
			crash.Go(func() { s.stopIfIdle(ctx, server, cfg.IdleTimeout) })
		}
		return nil
	}, grpc.UnaryInterceptor(auth.ServerInterceptor(cfg.AuthToken)))
}

type grpcServer struct {
	handler      Server
	bindCtx      func(context.Context) context.Context
	keepAlive    chan struct{}
	inFlightRPCs uint32
}

// inRPC should be called at the start of an RPC call. The returned function
// should be called when the RPC call finishes.
func (s *grpcServer) inRPC() func() {
	atomic.AddUint32(&s.inFlightRPCs, 1)
	select {
	case s.keepAlive <- struct{}{}:
	default:
	}
	return func() {
		select {
		case s.keepAlive <- struct{}{}:
		default:
		}
		atomic.AddUint32(&s.inFlightRPCs, ^uint32(0))
	}
}

// stopIfIdle calls GracefulStop on server if there are no writes the the
// keepAlive chan within idleTimeout.
// This function blocks until there's an idle timeout, or ctx is cancelled.
func (s *grpcServer) stopIfIdle(ctx context.Context, server *grpc.Server, idleTimeout time.Duration) {
	// Split the idleTimeout into N smaller chunks, and check that there was
	// no activity from the client in a contiguous N chunks of time.
	// This avoids killing the server if the machine is suspended (where the
	// client cannot send hearbeats, and the system clock jumps forward).
	waitTime := idleTimeout / 12
	var idleTime time.Duration

	stoppedSignal, stopped := task.NewSignal()
	defer func() {
		server.GracefulStop()
		stopped(ctx)
	}()

	// Wait for the server to stop before terminating the app.
	app.AddCleanupSignal(stoppedSignal)

	for {
		select {
		case <-task.ShouldStop(ctx):
			return
		case <-time.After(waitTime):
			if rpcs := atomic.LoadUint32(&s.inFlightRPCs); rpcs != 0 {
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
	release, err := s.handler.CheckForUpdates(s.bindCtx(ctx), req.IncludePrereleases)
	if err := service.NewError(err); err != nil {
		return &service.CheckForUpdatesResponse{Res: &service.CheckForUpdatesResponse_Error{Error: err}}, nil
	}
	return &service.CheckForUpdatesResponse{Res: &service.CheckForUpdatesResponse_Release{Release: release}}, nil
}

func (s *grpcServer) Get(ctx xctx.Context, req *service.GetRequest) (*service.GetResponse, error) {
	defer s.inRPC()()
	res, err := s.handler.Get(s.bindCtx(ctx), req.Path)
	if err := service.NewError(err); err != nil {
		return &service.GetResponse{Res: &service.GetResponse_Error{Error: err}}, nil
	}
	val := service.NewValue(res)
	return &service.GetResponse{Res: &service.GetResponse_Value{Value: val}}, nil
}

func (s *grpcServer) Set(ctx xctx.Context, req *service.SetRequest) (*service.SetResponse, error) {
	defer s.inRPC()()
	res, err := s.handler.Set(s.bindCtx(ctx), req.Path, req.Value.Get())
	if err := service.NewError(err); err != nil {
		return &service.SetResponse{Res: &service.SetResponse_Error{Error: err}}, nil
	}
	return &service.SetResponse{Res: &service.SetResponse_Path{Path: res}}, nil
}

func (s *grpcServer) Follow(ctx xctx.Context, req *service.FollowRequest) (*service.FollowResponse, error) {
	defer s.inRPC()()
	res, err := s.handler.Follow(s.bindCtx(ctx), req.Path)
	if err := service.NewError(err); err != nil {
		return &service.FollowResponse{Res: &service.FollowResponse_Error{Error: err}}, nil
	}
	return &service.FollowResponse{Res: &service.FollowResponse_Path{Path: res}}, nil
}

func (s *grpcServer) BeginCPUProfile(ctx xctx.Context, req *service.BeginCPUProfileRequest) (*service.BeginCPUProfileResponse, error) {
	defer s.inRPC()()
	err := s.handler.BeginCPUProfile(s.bindCtx(ctx))
	if err := service.NewError(err); err != nil {
		return &service.BeginCPUProfileResponse{Error: err}, nil
	}
	return &service.BeginCPUProfileResponse{}, nil
}

func (s *grpcServer) EndCPUProfile(ctx xctx.Context, req *service.EndCPUProfileRequest) (*service.EndCPUProfileResponse, error) {
	defer s.inRPC()()
	data, err := s.handler.EndCPUProfile(s.bindCtx(ctx))
	if err := service.NewError(err); err != nil {
		return &service.EndCPUProfileResponse{Res: &service.EndCPUProfileResponse_Error{Error: err}}, nil
	}
	return &service.EndCPUProfileResponse{Res: &service.EndCPUProfileResponse_Data{Data: data}}, nil
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

func (s *grpcServer) GetFramebufferAttachment(ctx xctx.Context, req *service.GetFramebufferAttachmentRequest) (*service.GetFramebufferAttachmentResponse, error) {
	defer s.inRPC()()
	image, err := s.handler.GetFramebufferAttachment(
		s.bindCtx(ctx),
		req.Device,
		req.After,
		req.Attachment,
		req.Settings,
		req.Hints,
	)
	if err := service.NewError(err); err != nil {
		return &service.GetFramebufferAttachmentResponse{Res: &service.GetFramebufferAttachmentResponse_Error{Error: err}}, nil
	}
	return &service.GetFramebufferAttachmentResponse{Res: &service.GetFramebufferAttachmentResponse_Image{Image: image}}, nil
}

func (s *grpcServer) GetLogStream(req *service.GetLogStreamRequest, server service.Gapid_GetLogStreamServer) error {
	defer s.inRPC()()
	ctx := server.Context()
	h := log.NewHandler(func(m *log.Message) { server.Send(log_pb.From(m)) }, nil)
	return s.handler.GetLogStream(s.bindCtx(ctx), h)
}

func (s *grpcServer) Find(req *service.FindRequest, server service.Gapid_FindServer) error {
	defer s.inRPC()()
	ctx := server.Context()
	return s.handler.Find(s.bindCtx(ctx), req, server.Send)
}
