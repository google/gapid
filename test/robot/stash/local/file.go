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
	"context"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/test/robot/stash"
)

type fileStore struct {
	entityIndex
	directory file.Path
}

const metaExtension = ".meta"

func init() {
	stash.RegisterHandler("file", DialFileService)
}

// DialFileService returns a file backed implementation of stash.Service from a url.
func DialFileService(ctx context.Context, location *url.URL) (*stash.Client, error) {
	if location.Host != "" {
		return nil, log.Err(ctx, nil, "Host not supported for file servers")
	}
	if location.Path == "" {
		return nil, log.Err(ctx, nil, "Path must be specified for file servers")
	}
	return NewFileService(ctx, file.Abs(location.Path))
}

// NewFileService returns a file backed implementation of stash.Service for a path.
func NewFileService(ctx context.Context, directory file.Path) (*stash.Client, error) {
	s := &fileStore{directory: directory}
	s.entityIndex.init()
	os.MkdirAll(directory.System(), 0755)
	files, err := ioutil.ReadDir(directory.System())
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		filename := directory.Join(file.Name())
		if filename.Ext() != metaExtension {
			continue
		}
		data, err := ioutil.ReadFile(filename.System())
		if err != nil {
			return nil, err
		}
		entity := &stash.Entity{}
		if err := proto.Unmarshal(data, entity); err != nil {
			return nil, err
		}
		s.lockedAddEntry(ctx, entity)
	}
	return &stash.Client{Service: s}, nil
}

func (c *fileStore) Close() {}

func (s *fileStore) Open(ctx context.Context, id string) (io.ReadSeeker, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, found := s.byID[id]; !found {
		return nil, stash.ErrEntityNotFound
	}
	return os.Open(s.directory.Join(id).System())
}

func (s *fileStore) Read(ctx context.Context, id string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, found := s.byID[id]; !found {
		return nil, stash.ErrEntityNotFound
	}
	return ioutil.ReadFile(s.directory.Join(id).System())
}

func (s *fileStore) Create(ctx context.Context, info *stash.Upload) (io.WriteCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, found := s.byID[info.Id]; found {
		return nil, log.Err(ctx, nil, "Stash entity already exists")
	}
	filename := s.directory.Join(info.Id)
	mode := os.FileMode(0666)
	if info.Executable {
		mode = 0777
	}
	f, err := os.OpenFile(filename.System(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return nil, log.Err(ctx, err, "Stash could not create file")
	}
	now, _ := ptypes.TimestampProto(time.Now())
	w := &fileStoreWriter{
		f:    f,
		meta: filename.ChangeExt(metaExtension),
		entity: &stash.Entity{
			Upload:    info,
			Status:    stash.Uploading,
			Length:    0,
			Timestamp: now,
		},
	}
	// Write the meta data to the disk
	meta, err := proto.Marshal(w.entity)
	if err != nil {
		return nil, log.Err(ctx, err, "Stash could not marshal meta data")
	}
	err = ioutil.WriteFile(w.meta.System(), meta, 0666)
	if err != nil {
		return nil, log.Err(ctx, err, "Stash could not save meta data")
	}
	// and finally add the entry into the map
	s.lockedAddEntry(ctx, w.entity)
	return w, nil
}

type fileStoreWriter struct {
	entity *stash.Entity
	meta   file.Path
	f      *os.File
	size   int64
}

func (w *fileStoreWriter) Write(b []byte) (int, error) {
	n, err := w.f.Write(b)
	w.size += int64(n)
	return n, err
}

func (w *fileStoreWriter) Close() error {
	// Write the finalized meta data to the disk
	w.entity.Status = stash.Present
	w.entity.Length = w.size
	meta, err := proto.Marshal(w.entity)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(w.meta.System(), meta, 0666)
	if err != nil {
		return err
	}
	return w.f.Close()
}
