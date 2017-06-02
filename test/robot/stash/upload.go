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

package stash

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"hash"
	"io"
	"mime"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
)

var sha1Pool = sync.Pool{New: func() interface{} { return sha1.New() }}

func hashStream(r io.Reader) string {
	h := sha1Pool.Get().(hash.Hash)
	defer sha1Pool.Put(h)
	h.Reset()
	io.Copy(h, r)
	return hex.EncodeToString(h.Sum(nil))
}

func uploadStream(ctx context.Context, service Service, info Upload, r Uploadable) (string, error) {
	// first calculate an id for the file
	info.Id = hashStream(r)
	// now check if the file is already in the stash
	if entity, _ := service.Lookup(ctx, info.Id); entity != nil {
		return info.Id, nil
	}
	// reset the stream and then upload the file to the specified id
	if err := r.Reset(); err != nil {
		return "", err
	}
	if len(info.Type) == 0 {
		for _, name := range info.Name {
			info.Type = append(info.Type, mime.TypeByExtension(filepath.Ext(name)))
		}
	}
	w, err := service.Create(ctx, &info)
	if err != nil {
		return "", err
	}
	defer w.Close()
	_, err = io.Copy(w, r)
	if errors.Cause(err) == io.EOF {
		err = nil
	}
	return info.Id, err
}

type seekadapter struct {
	io.ReadSeeker
}

func (s seekadapter) Reset() error {
	_, err := s.Seek(0, io.SeekStart)
	return err
}

type bytesAdapter struct {
	data   []byte
	offset int
}

func (b *bytesAdapter) Read(to []byte) (int, error) {
	if b.offset >= len(b.data) {
		return 0, io.EOF
	}
	count := copy(to, b.data[b.offset:])
	b.offset += count
	return count, nil
}

func (b *bytesAdapter) Reset() error {
	b.offset = 0
	return nil
}
