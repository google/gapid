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

package local

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/robot/stash"
)

type memoryStore struct {
	entityIndex
	data map[string][]byte
}

func init() {
	stash.RegisterHandler("memory", DialMemoryService)
}

// DialMemoryService returns a file backed implementation of stash.Service from a url.
func DialMemoryService(ctx context.Context, location *url.URL) (*stash.Client, error) {
	if location.Host != "" {
		return nil, log.Err(ctx, nil, "Host not supported for memory servers")
	}
	if location.Path != "" {
		//TODO: keep a persistent map of memory services by name?
		return nil, log.Err(ctx, nil, "Path not supported for memory servers")
	}
	return NewMemoryService(), nil
}

// NewMemoryService returns a new purely in memory implementation of Store.
func NewMemoryService() *stash.Client {
	store := &memoryStore{data: map[string][]byte{}}
	store.entityIndex.init()
	return &stash.Client{Service: store}
}

func (s *memoryStore) Close() {}

func (s *memoryStore) Open(ctx context.Context, id string) (io.ReadSeeker, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if data, found := s.data[id]; found {
		return bytes.NewReader(data), nil
	}
	return nil, stash.ErrEntityNotFound
}

func (s *memoryStore) Read(ctx context.Context, id string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, found := s.data[id]
	if !found {
		return nil, stash.ErrEntityNotFound
	}
	return data, nil
}

func (s *memoryStore) Create(ctx context.Context, info *stash.Upload) (io.WriteCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, found := s.byID[info.Id]; found {
		return nil, log.Err(ctx, nil, "Stash entity already exists")
	}
	now, _ := ptypes.TimestampProto(time.Now())
	w := &memoryStoreWriter{
		store: s,
		entity: &stash.Entity{
			Upload:    info,
			Status:    stash.Uploading,
			Length:    0,
			Timestamp: now,
		},
	}
	s.lockedAddEntry(ctx, w.entity)
	return w, nil
}

type memoryStoreWriter struct {
	store  *memoryStore
	entity *stash.Entity
	buf    bytes.Buffer
}

func (w *memoryStoreWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *memoryStoreWriter) Close() error {
	data := w.buf.Bytes()
	w.entity.Status = stash.Present
	w.entity.Length = int64(len(data))
	w.store.data[w.entity.Upload.Id] = data
	return nil
}
