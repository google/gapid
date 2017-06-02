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

package grpc

import (
	"context"
	"io"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/stash"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type storeServer struct {
	service stash.Service
}

// Serve wraps a store in a grpc server.
func Serve(ctx context.Context, grpcServer *grpc.Server, service stash.Service) error {
	RegisterServiceServer(grpcServer, &storeServer{service: service})
	return nil
}

// Search scans the underlying store for matching entities.
// See ServiceServer for more information.
func (s *storeServer) Search(query *search.Query, stream Service_SearchServer) error {
	ctx := stream.Context()
	return s.service.Search(ctx, query, func(ctx context.Context, e *stash.Entity) error {
		return stream.Send(e)
	})
}

type uploader struct {
	w io.WriteCloser
}

func (u *uploader) Open(ctx context.Context, service stash.Service, upload *stash.Upload) error {
	u.Close(ctx)
	w, err := service.Create(ctx, upload)
	if err != nil {
		return log.Err(ctx, err, "Opening upload")
	}
	u.w = w
	return nil
}

func (u *uploader) Write(ctx context.Context, data []byte) error {
	if _, err := u.w.Write(data); err != nil {
		return log.Err(ctx, err, "Appending upload")
	}
	return nil
}

func (u *uploader) Close(ctx context.Context) {
	if u.w == nil {
		return
	}
	u.w.Close()
}

// Upload adds entities to the underlying store.
// See ServiceServer for more information.
func (s *storeServer) Upload(stream Service_UploadServer) error {
	ctx := stream.Context()
	u := &uploader{}
	defer u.Close(ctx)
	for {
		chunk, err := stream.Recv()
		switch {
		case errors.Cause(err) == io.EOF:
			return stream.SendAndClose(&UploadResponse{})
		case err != nil:
			return err
		default:
			switch c := chunk.Of.(type) {
			case *UploadChunk_Upload:
				u.Open(ctx, s.service, c.Upload)
			case *UploadChunk_Data:
				u.Write(ctx, c.Data)
			default:
				return log.Err(ctx, nil, "Unknown upload chunk type")
			}
		}
	}
}

// Download fetches entities from the underlying store.
// See ServiceServer for more information.
func (s *storeServer) Download(request *DownloadRequest, stream Service_DownloadServer) error {
	ctx := stream.Context()
	readSeeker, err := s.service.Open(ctx, request.Id)
	if err != nil {
		return log.Err(ctx, err, "Entity open")
	}

	if request.Offset > 0 {
		if _, err := readSeeker.Seek(int64(request.Offset), io.SeekStart); err != nil {
			return log.Err(ctx, err, "Seek")
		}
	}

	var reader io.Reader = readSeeker
	bufSize := downloadLimit
	if request.Length > 0 {
		reader = io.LimitReader(readSeeker, int64(request.Length))
		if int(request.Length) < bufSize {
			bufSize = int(request.Length)
		}
	}

	buf := make([]byte, bufSize)
	chunk := &DownloadChunk{}
	for {
		if task.Stopped(ctx) {
			return nil
		}

		n, err := reader.Read(buf)
		if errors.Cause(err) == io.EOF {
			return nil
		}
		if err != nil {
			return log.Err(ctx, err, "Data read")
		}
		chunk.Data = buf[:n]
		err = stream.Send(chunk)
		if err != nil {
			return log.Err(ctx, err, "Stash download chunk")
		}
	}
}
