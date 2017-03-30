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

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/context/keys"
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

func NewWithListener(ctx context.Context, l net.Listener, cfg Config, srvChan chan<- *grpc.Server) error {
	s := NewGapidServer(ctx, cfg)
	return grpcutil.ServeWithListener(ctx, l, func(ctx context.Context, listener net.Listener, server *grpc.Server) error {
		if addr, ok := listener.Addr().(*net.TCPAddr); ok {
			// The following message is parsed by launchers to detect the selected port. DO NOT CHANGE!
			fmt.Printf("Bound on port '%d'\n", addr.Port)
		}
		service.RegisterGapidServer(server, s)

		if srvChan != nil {
			srvChan <- server
		}
		return nil
	}, grpc.UnaryInterceptor(auth.ServerInterceptor(cfg.AuthToken)))
}

// NewGapidServer returns a GapidServer interface to a new server instace.
func NewGapidServer(ctx context.Context, cfg Config) service.GapidServer {
	outer := ctx
	return &grpcServer{
		handler: New(ctx, cfg),
		bindCtx: func(ctx context.Context) context.Context { return keys.Clone(ctx, outer) },
	}
}

type grpcServer struct {
	handler Server
	bindCtx func(context.Context) context.Context
}

func (s *grpcServer) GetServerInfo(ctx xctx.Context, req *service.GetServerInfoRequest) (*service.GetServerInfoResponse, error) {
	info, err := s.handler.GetServerInfo(s.bindCtx(ctx))
	if err := service.NewError(err); err != nil {
		return &service.GetServerInfoResponse{Res: &service.GetServerInfoResponse_Error{Error: err}}, nil
	}
	return &service.GetServerInfoResponse{Res: &service.GetServerInfoResponse_Info{Info: info}}, nil
}

func (s *grpcServer) Get(ctx xctx.Context, req *service.GetRequest) (*service.GetResponse, error) {
	res, err := s.handler.Get(s.bindCtx(ctx), req.Path)
	if err := service.NewError(err); err != nil {
		return &service.GetResponse{Res: &service.GetResponse_Error{Error: err}}, nil
	}
	val := service.NewValue(res)
	return &service.GetResponse{Res: &service.GetResponse_Value{Value: val}}, nil
}

func (s *grpcServer) Set(ctx xctx.Context, req *service.SetRequest) (*service.SetResponse, error) {
	res, err := s.handler.Set(s.bindCtx(ctx), req.Path, req.Value.Get())
	if err := service.NewError(err); err != nil {
		return &service.SetResponse{Res: &service.SetResponse_Error{Error: err}}, nil
	}
	return &service.SetResponse{Res: &service.SetResponse_Path{Path: res}}, nil
}

func (s *grpcServer) Follow(ctx xctx.Context, req *service.FollowRequest) (*service.FollowResponse, error) {
	res, err := s.handler.Follow(s.bindCtx(ctx), req.Path)
	if err := service.NewError(err); err != nil {
		return &service.FollowResponse{Res: &service.FollowResponse_Error{Error: err}}, nil
	}
	return &service.FollowResponse{Res: &service.FollowResponse_Path{Path: res}}, nil
}

func (s *grpcServer) BeginCPUProfile(ctx xctx.Context, req *service.BeginCPUProfileRequest) (*service.BeginCPUProfileResponse, error) {
	err := s.handler.BeginCPUProfile(s.bindCtx(ctx))
	if err := service.NewError(err); err != nil {
		return &service.BeginCPUProfileResponse{Error: err}, nil
	}
	return &service.BeginCPUProfileResponse{}, nil
}

func (s *grpcServer) EndCPUProfile(ctx xctx.Context, req *service.EndCPUProfileRequest) (*service.EndCPUProfileResponse, error) {
	data, err := s.handler.EndCPUProfile(s.bindCtx(ctx))
	if err := service.NewError(err); err != nil {
		return &service.EndCPUProfileResponse{Res: &service.EndCPUProfileResponse_Error{Error: err}}, nil
	}
	return &service.EndCPUProfileResponse{Res: &service.EndCPUProfileResponse_Data{Data: data}}, nil
}

func (s *grpcServer) GetPerformanceCounters(ctx xctx.Context, req *service.GetPerformanceCountersRequest) (*service.GetPerformanceCountersResponse, error) {
	data, err := s.handler.GetPerformanceCounters(s.bindCtx(ctx))
	if err := service.NewError(err); err != nil {
		return &service.GetPerformanceCountersResponse{Res: &service.GetPerformanceCountersResponse_Error{Error: err}}, nil
	}
	return &service.GetPerformanceCountersResponse{Res: &service.GetPerformanceCountersResponse_Data{Data: data}}, nil
}

func (s *grpcServer) GetProfile(ctx xctx.Context, req *service.GetProfileRequest) (*service.GetProfileResponse, error) {
	data, err := s.handler.GetProfile(s.bindCtx(ctx), req.Name, req.Debug)
	if err := service.NewError(err); err != nil {
		return &service.GetProfileResponse{Res: &service.GetProfileResponse_Error{Error: err}}, nil
	}
	return &service.GetProfileResponse{Res: &service.GetProfileResponse_Data{Data: data}}, nil
}

func (s *grpcServer) GetSchema(ctx xctx.Context, req *service.GetSchemaRequest) (*service.GetSchemaResponse, error) {
	msg, err := s.handler.GetSchema(s.bindCtx(ctx))
	if err := service.NewError(err); err != nil {
		return &service.GetSchemaResponse{Res: &service.GetSchemaResponse_Error{Error: err}}, nil
	}
	obj := &service.Object{}
	if err := obj.Encode(msg); err != nil {
		return nil, err
	}
	return &service.GetSchemaResponse{Res: &service.GetSchemaResponse_Object{Object: obj}}, nil
}

func (s *grpcServer) GetAvailableStringTables(ctx xctx.Context, req *service.GetAvailableStringTablesRequest) (*service.GetAvailableStringTablesResponse, error) {
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
	table, err := s.handler.GetStringTable(s.bindCtx(ctx), req.Table)
	if err := service.NewError(err); err != nil {
		return &service.GetStringTableResponse{Res: &service.GetStringTableResponse_Error{Error: err}}, nil
	}
	return &service.GetStringTableResponse{Res: &service.GetStringTableResponse_Table{Table: table}}, nil
}

func (s *grpcServer) ImportCapture(ctx xctx.Context, req *service.ImportCaptureRequest) (*service.ImportCaptureResponse, error) {
	capture, err := s.handler.ImportCapture(s.bindCtx(ctx), req.Name, req.Data)
	if err := service.NewError(err); err != nil {
		return &service.ImportCaptureResponse{Res: &service.ImportCaptureResponse_Error{Error: err}}, nil
	}
	return &service.ImportCaptureResponse{Res: &service.ImportCaptureResponse_Capture{Capture: capture}}, nil
}

func (s *grpcServer) ExportCapture(ctx xctx.Context, req *service.ExportCaptureRequest) (*service.ExportCaptureResponse, error) {
	data, err := s.handler.ExportCapture(s.bindCtx(ctx), req.Capture)
	if err := service.NewError(err); err != nil {
		return &service.ExportCaptureResponse{Res: &service.ExportCaptureResponse_Error{Error: err}}, nil
	}
	return &service.ExportCaptureResponse{Res: &service.ExportCaptureResponse_Data{Data: data}}, nil
}

func (s *grpcServer) LoadCapture(ctx xctx.Context, req *service.LoadCaptureRequest) (*service.LoadCaptureResponse, error) {
	capture, err := s.handler.LoadCapture(s.bindCtx(ctx), req.Path)
	if err := service.NewError(err); err != nil {
		return &service.LoadCaptureResponse{Res: &service.LoadCaptureResponse_Error{Error: err}}, nil
	}
	return &service.LoadCaptureResponse{Res: &service.LoadCaptureResponse_Capture{Capture: capture}}, nil
}

func (s *grpcServer) GetDevices(ctx xctx.Context, req *service.GetDevicesRequest) (*service.GetDevicesResponse, error) {
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
	ctx := server.Context()
	h := log.NewHandler(func(m *log.Message) { server.Send(log_pb.From(m)) }, nil)
	return s.handler.GetLogStream(s.bindCtx(ctx), h)
}
