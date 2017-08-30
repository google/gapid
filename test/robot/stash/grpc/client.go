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
	"io/ioutil"
	"net/url"

	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/search/script"
	"github.com/google/gapid/test/robot/stash"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const (
	uploadLimit   = 1 * 1024 * 1024
	downloadLimit = uploadLimit

	ErrInvalidOffset = fault.Const("invalid seek offset")
)

type (
	remoteStore struct {
		client ServiceClient
		temp   file.Path
	}

	connectedStore struct {
		remoteStore
		conn *grpc.ClientConn
	}
)

func init() {
	stash.RegisterHandler("grpc", Dial)
}

// Connect returns a remote grpc backed implementation of stash.Service using the supplied connection.
func Connect(ctx context.Context, conn *grpc.ClientConn) (*stash.Client, error) {
	remote, err := connect(ctx, conn)
	if err != nil {
		return nil, err
	}
	return &stash.Client{Service: &remote}, nil
}

// MustConnect returns a remote grpc backed implementation of a stash client using the supplied connection.
// It panics if the connection fails for any reason.
func MustConnect(ctx context.Context, conn *grpc.ClientConn) *stash.Client {
	s, err := Connect(ctx, conn)
	if err != nil {
		panic(err)
	}
	return s
}

// Dial returns a remote grpc backed stash client from a url.
func Dial(ctx context.Context, location *url.URL) (*stash.Client, error) {
	if location.Host == "" {
		return nil, log.Err(ctx, nil, "Host not supported for memory servers")
	}
	if location.Path != "" {
		return nil, log.Err(ctx, nil, "Path not supported for grpc servers")
	}
	conn, err := grpcutil.Dial(ctx, location.Host, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	remote, err := connect(ctx, conn)
	if err != nil {
		return nil, err
	}
	return &stash.Client{Service: &connectedStore{
		remoteStore: remote,
		conn:        conn,
	}}, nil
}

func connect(ctx context.Context, conn *grpc.ClientConn) (remoteStore, error) {
	tmp, err := ioutil.TempDir("", "stash_")
	if err != nil {
		return remoteStore{}, err
	}
	return remoteStore{
		client: NewServiceClient(conn),
		temp:   file.Abs(tmp),
	}, nil
}

func (s *remoteStore) Close()    {}
func (s *connectedStore) Close() { s.conn.Close() }

var uploadQuery = script.MustParse("Upload.Id == $").Using("$")

func (s *remoteStore) Lookup(ctx context.Context, id string) (*stash.Entity, error) {
	query := uploadQuery(id).Query()
	stream, err := s.client.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	entity, err := stream.Recv()
	if errors.Cause(err) == io.EOF {
		if entity == nil {
			err = stash.ErrEntityNotFound
		} else {
			err = nil
		}
	}
	return entity, err
}

func (s *remoteStore) Search(ctx context.Context, query *search.Query, handler stash.EntityHandler) error {
	stream, err := s.client.Search(ctx, query)
	if err != nil {
		return err
	}
	p := grpcutil.ToProducer(stream)
	return event.Feed(ctx, event.AsHandler(ctx, handler), p)
}

func (s *remoteStore) Open(ctx context.Context, id string) (io.ReadSeeker, error) {
	e, err := s.Lookup(ctx, id)
	if err != nil {
		return nil, log.Err(ctx, err, "entity lookup")
	}
	if e.Status != stash.Status_Present {
		return nil, log.Err(ctx, err, "entity not ready")
	}
	return &remoteStoreReadSeeker{ctx: ctx, id: id, len: e.GetLength(), s: s}, nil
}

type remoteStoreReadSeeker struct {
	ctx context.Context
	s   *remoteStore
	id  string
	len int64

	offset int64
	cancel context.CancelFunc
	stream Service_DownloadClient

	data    []byte
	recvbuf []byte
}

func (r *remoteStoreReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = r.offset + offset
	case io.SeekEnd:
		newOffset = r.len + offset
	}

	if newOffset < 0 {
		return 0, ErrInvalidOffset
	}

	delta := newOffset - r.offset
	bufoffset := int64(len(r.recvbuf) - len(r.data))
	if 0 < delta && delta < int64(len(r.data)) {
		r.data = r.data[delta:]
	} else if -bufoffset < delta && delta < 0 {
		r.data = r.recvbuf[int(bufoffset+delta):]
	} else if delta != 0 {
		if r.cancel != nil {
			r.cancel()
		}
		r.stream = nil
		r.data = nil
		r.recvbuf = nil
	}

	r.offset = newOffset
	return newOffset, nil
}

func (r *remoteStoreReadSeeker) Read(b []byte) (int, error) {
	for len(r.data) == 0 {
		if r.stream == nil {
			ctx, cancel := context.WithCancel(r.ctx)
			stream, err := r.s.client.Download(ctx, &DownloadRequest{Id: r.id, Offset: uint64(r.offset)})
			r.cancel = cancel
			r.stream = stream
			if err != nil {
				return 0, log.Err(r.ctx, err, "Remote store download")
			}
		}

		chunk, err := r.stream.Recv()
		if err != nil {
			return 0, err
		}
		r.data = chunk.Data
		r.recvbuf = r.data
	}
	n := copy(b, r.data)
	if n == len(r.data) {
		r.data = nil
	} else {
		r.data = r.data[n:]
	}
	r.offset += int64(n)
	return n, nil
}

func (s *remoteStore) Read(ctx context.Context, id string) ([]byte, error) {
	r, err := s.Open(ctx, id)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

func (s *remoteStore) Create(ctx context.Context, info *stash.Upload) (io.WriteCloser, error) {
	stream, err := s.client.Upload(ctx)
	if err != nil {
		return nil, log.Err(ctx, err, "Remote store upload start")
	}
	if err := stream.Send(&UploadChunk{Of: &UploadChunk_Upload{Upload: info}}); err != nil {
		return nil, log.Err(ctx, err, "Remote store upload header")
	}
	return &remoteStoreWriter{stream: stream}, nil
}

type remoteStoreWriter struct {
	stream Service_UploadClient
}

func (w *remoteStoreWriter) Write(b []byte) (int, error) {
	n := 0
	for len(b) > 0 {
		data := b
		if len(b) > uploadLimit {
			data = b[:uploadLimit]
			b = b[uploadLimit:]
		} else {
			b = nil
		}
		err := w.stream.Send(&UploadChunk{Of: &UploadChunk_Data{Data: data}})
		if err != nil {
			return 0, err
		}
		n += len(data)
	}
	return n, nil
}

func (w *remoteStoreWriter) Close() error {
	// Use CloseAndRecv() to block until the server acknowledges the close.
	// CloseSend() would return without waiting for the server to complete the
	// call.
	_, err := w.stream.CloseAndRecv()
	return err
}

func (s *remoteStore) Upload(ctx context.Context, info *stash.Upload, reader io.Reader) error {
	stream, err := s.client.Upload(ctx)
	if err != nil {
		return log.Err(ctx, err, "Remote store upload")
	}
	buf := make([]byte, uploadLimit)
	chunk := &UploadChunk{
		Of: &UploadChunk_Upload{Upload: info},
	}
	for {
		err = stream.Send(chunk)
		if err != nil {
			return log.Err(ctx, err, "Remote store upload")
		}
		n, err := reader.Read(buf)
		if errors.Cause(err) == io.EOF {
			stream.CloseSend()
			return nil
		}
		if err != nil {
			return log.Err(ctx, err, "Data read")
		}
		chunk.Of = &UploadChunk_Data{Data: buf[:n]}
	}
}
