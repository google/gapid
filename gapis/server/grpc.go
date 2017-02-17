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
	"fmt"
	"net"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/gapis/service"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Listen starts a new GRPC server listening on addr.
// This is a blocking call.
func Listen(ctx log.Context, addr string, cfg Config) error {
	s := NewGapidServer(ctx, cfg)
	return grpcutil.Serve(ctx, addr, func(ctx log.Context, listener net.Listener, server *grpc.Server) error {
		// The following message is parsed by launchers to detect the selected port. DO NOT CHANGE!
		fmt.Printf("Bound on port '%d'\n", listener.Addr().(*net.TCPAddr).Port)
		service.RegisterGapidServer(server, s)
		return nil
	}, grpc.UnaryInterceptor(auth.ServerInterceptor(cfg.AuthToken)))
}

// NewGapidServer returns a GapidServer interface to a new server instace.
func NewGapidServer(ctx log.Context, cfg Config) service.GapidServer {
	outer := ctx.Unwrap()
	return &grpcServer{
		handler: New(ctx, cfg),
		bindCtx: func(ctx log.Context) log.Context { return log.Wrap(keys.Clone(ctx, outer)) },
	}
}

type grpcServer struct {
	handler Server
	bindCtx func(log.Context) log.Context
}

func (s *grpcServer) GetServerInfo(ctx context.Context, req *service.GetServerInfoRequest) (*service.GetServerInfoResponse, error) {
	info, err := s.handler.GetServerInfo(s.bindCtx(log.Wrap(ctx)))
	if err := service.NewError(err); err != nil {
		return &service.GetServerInfoResponse{Res: &service.GetServerInfoResponse_Error{Error: err}}, nil
	}
	return &service.GetServerInfoResponse{Res: &service.GetServerInfoResponse_Info{Info: info}}, nil
}

func (s *grpcServer) Get(ctx context.Context, req *service.GetRequest) (*service.GetResponse, error) {
	res, err := s.handler.Get(s.bindCtx(log.Wrap(ctx)), req.Path)
	if err := service.NewError(err); err != nil {
		return &service.GetResponse{Res: &service.GetResponse_Error{Error: err}}, nil
	}
	val := service.NewValue(res)
	return &service.GetResponse{Res: &service.GetResponse_Value{Value: val}}, nil
}

func (s *grpcServer) Set(ctx context.Context, req *service.SetRequest) (*service.SetResponse, error) {
	res, err := s.handler.Set(s.bindCtx(log.Wrap(ctx)), req.Path, req.Value.Get())
	if err := service.NewError(err); err != nil {
		return &service.SetResponse{Res: &service.SetResponse_Error{Error: err}}, nil
	}
	return &service.SetResponse{Res: &service.SetResponse_Path{Path: res}}, nil
}

func (s *grpcServer) Follow(ctx context.Context, req *service.FollowRequest) (*service.FollowResponse, error) {
	res, err := s.handler.Follow(s.bindCtx(log.Wrap(ctx)), req.Path)
	if err := service.NewError(err); err != nil {
		return &service.FollowResponse{Res: &service.FollowResponse_Error{Error: err}}, nil
	}
	return &service.FollowResponse{Res: &service.FollowResponse_Path{Path: res}}, nil
}

func (s *grpcServer) BeginCPUProfile(ctx context.Context, req *service.BeginCPUProfileRequest) (*service.BeginCPUProfileResponse, error) {
	err := s.handler.BeginCPUProfile(s.bindCtx(log.Wrap(ctx)))
	if err := service.NewError(err); err != nil {
		return &service.BeginCPUProfileResponse{Error: err}, nil
	}
	return &service.BeginCPUProfileResponse{}, nil
}

func (s *grpcServer) EndCPUProfile(ctx context.Context, req *service.EndCPUProfileRequest) (*service.EndCPUProfileResponse, error) {
	data, err := s.handler.EndCPUProfile(s.bindCtx(log.Wrap(ctx)))
	if err := service.NewError(err); err != nil {
		return &service.EndCPUProfileResponse{Res: &service.EndCPUProfileResponse_Error{Error: err}}, nil
	}
	return &service.EndCPUProfileResponse{Res: &service.EndCPUProfileResponse_Data{Data: data}}, nil
}

func (s *grpcServer) GetPerformanceCounters(ctx context.Context, req *service.GetPerformanceCountersRequest) (*service.GetPerformanceCountersResponse, error) {
	data, err := s.handler.GetPerformanceCounters(s.bindCtx(log.Wrap(ctx)))
	if err := service.NewError(err); err != nil {
		return &service.GetPerformanceCountersResponse{Res: &service.GetPerformanceCountersResponse_Error{Error: err}}, nil
	}
	return &service.GetPerformanceCountersResponse{Res: &service.GetPerformanceCountersResponse_Data{Data: data}}, nil
}

func (s *grpcServer) GetProfile(ctx context.Context, req *service.GetProfileRequest) (*service.GetProfileResponse, error) {
	data, err := s.handler.GetProfile(s.bindCtx(log.Wrap(ctx)), req.Name, req.Debug)
	if err := service.NewError(err); err != nil {
		return &service.GetProfileResponse{Res: &service.GetProfileResponse_Error{Error: err}}, nil
	}
	return &service.GetProfileResponse{Res: &service.GetProfileResponse_Data{Data: data}}, nil
}

func (s *grpcServer) GetSchema(ctx context.Context, req *service.GetSchemaRequest) (*service.GetSchemaResponse, error) {
	msg, err := s.handler.GetSchema(s.bindCtx(log.Wrap(ctx)))
	if err := service.NewError(err); err != nil {
		return &service.GetSchemaResponse{Res: &service.GetSchemaResponse_Error{Error: err}}, nil
	}
	obj := &service.Object{}
	if err := obj.Encode(msg); err != nil {
		return nil, err
	}
	return &service.GetSchemaResponse{Res: &service.GetSchemaResponse_Object{Object: obj}}, nil
}

func (s *grpcServer) GetAvailableStringTables(ctx context.Context, req *service.GetAvailableStringTablesRequest) (*service.GetAvailableStringTablesResponse, error) {
	tables, err := s.handler.GetAvailableStringTables(s.bindCtx(log.Wrap(ctx)))
	if err := service.NewError(err); err != nil {
		return &service.GetAvailableStringTablesResponse{Res: &service.GetAvailableStringTablesResponse_Error{Error: err}}, nil
	}
	return &service.GetAvailableStringTablesResponse{
		Res: &service.GetAvailableStringTablesResponse_Tables{
			Tables: &service.StringTableInfos{List: tables},
		},
	}, nil
}

func (s *grpcServer) GetStringTable(ctx context.Context, req *service.GetStringTableRequest) (*service.GetStringTableResponse, error) {
	table, err := s.handler.GetStringTable(s.bindCtx(log.Wrap(ctx)), req.Table)
	if err := service.NewError(err); err != nil {
		return &service.GetStringTableResponse{Res: &service.GetStringTableResponse_Error{Error: err}}, nil
	}
	return &service.GetStringTableResponse{Res: &service.GetStringTableResponse_Table{Table: table}}, nil
}

func (s *grpcServer) ImportCapture(ctx context.Context, req *service.ImportCaptureRequest) (*service.ImportCaptureResponse, error) {
	capture, err := s.handler.ImportCapture(s.bindCtx(log.Wrap(ctx)), req.Name, req.Data)
	if err := service.NewError(err); err != nil {
		return &service.ImportCaptureResponse{Res: &service.ImportCaptureResponse_Error{Error: err}}, nil
	}
	return &service.ImportCaptureResponse{Res: &service.ImportCaptureResponse_Capture{Capture: capture}}, nil
}

func (s *grpcServer) LoadCapture(ctx context.Context, req *service.LoadCaptureRequest) (*service.LoadCaptureResponse, error) {
	capture, err := s.handler.LoadCapture(s.bindCtx(log.Wrap(ctx)), req.Path)
	if err := service.NewError(err); err != nil {
		return &service.LoadCaptureResponse{Res: &service.LoadCaptureResponse_Error{Error: err}}, nil
	}
	return &service.LoadCaptureResponse{Res: &service.LoadCaptureResponse_Capture{Capture: capture}}, nil
}

func (s *grpcServer) GetDevices(ctx context.Context, req *service.GetDevicesRequest) (*service.GetDevicesResponse, error) {
	devices, err := s.handler.GetDevices(s.bindCtx(log.Wrap(ctx)))
	if err := service.NewError(err); err != nil {
		return &service.GetDevicesResponse{Res: &service.GetDevicesResponse_Error{Error: err}}, nil
	}
	return &service.GetDevicesResponse{
		Res: &service.GetDevicesResponse_Devices{
			Devices: &service.Devices{List: devices},
		},
	}, nil
}

func (s *grpcServer) GetDevicesForReplay(ctx context.Context, req *service.GetDevicesForReplayRequest) (*service.GetDevicesForReplayResponse, error) {
	devices, err := s.handler.GetDevicesForReplay(s.bindCtx(log.Wrap(ctx)), req.Capture)
	if err := service.NewError(err); err != nil {
		return &service.GetDevicesForReplayResponse{Res: &service.GetDevicesForReplayResponse_Error{Error: err}}, nil
	}
	return &service.GetDevicesForReplayResponse{
		Res: &service.GetDevicesForReplayResponse_Devices{
			Devices: &service.Devices{List: devices},
		},
	}, nil
}

func (s *grpcServer) GetFramebufferAttachment(ctx context.Context, req *service.GetFramebufferAttachmentRequest) (*service.GetFramebufferAttachmentResponse, error) {
	image, err := s.handler.GetFramebufferAttachment(s.bindCtx(log.Wrap(ctx)), req.Device, req.After, req.Attachment, req.Settings)
	if err := service.NewError(err); err != nil {
		return &service.GetFramebufferAttachmentResponse{Res: &service.GetFramebufferAttachmentResponse_Error{Error: err}}, nil
	}
	return &service.GetFramebufferAttachmentResponse{Res: &service.GetFramebufferAttachmentResponse_Image{Image: image}}, nil
}
