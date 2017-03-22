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

package record

import (
	"context"
	"os"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
)

var (
	fileSearchOrder = []Kind{JSON, Text, Proto}
)

// FileShelf is an implementation of Shelf that stores it's ledgers in files.
// Each ledger goes in it's own files, and records are appended to the file as they arrive.
// The actual storage format depends on the Kind of the ledger, which is detected on open.
type FileShelf struct {
	path file.Path
	// Kind specifies the default ledger storage for new ledgers that are created.
	// It has no impact on existing ledgers.
	Kind Kind
}

// fileType implements file handling for a specific file Kind.
type fileType interface {
	Ext() string
	Open(ctx context.Context, f *os.File, null interface{}) (LedgerInstance, error)
}

type readAt struct {
	f      *os.File
	offset int64
}

var (
	// fileTypes holds the map of file handlers for the various kinds.
	fileTypes = map[Kind]fileType{
		JSON:  jsonFileType{},
		Text:  pbtxtFileType{},
		Proto: pbbFileType{},
	}
)

// NewFileShelf build a new FileShelf backed implementation of a Shelf where the files will be stored
// in the specified path.
func NewFileShelf(ctx context.Context, path file.Path) (Shelf, error) {
	os.MkdirAll(path.System(), 0755)
	return &FileShelf{
		path: path,
		Kind: Proto,
	}, nil
}

// Open implements Shelf.Open for a file backed shelf.
// It will attempt to open and read the specified ledger, returning an error only
// if the ledger exists but is invalid.
// It will try all known ledger types in priority order, if you have more than
// one type for the ledger, only the first file will be found.
func (s *FileShelf) Open(ctx context.Context, name string, null interface{}) (Ledger, error) {
	// see if we can find a file that matches the name
	for _, kind := range fileSearchOrder {
		ft := fileTypes[kind]
		l, err := s.tryOpen(ctx, name, null, ft)
		if err != nil {
			return nil, err
		}
		if l != nil {
			return l, nil
		}
	}
	return nil, nil
}

// Create implements Shelf.Create for a file backed shelf, creating a new ledger file of the
// current default kind.
// It will return an error if for any reason the ledger cannot be created.
func (s *FileShelf) Create(ctx context.Context, name string, null interface{}) (Ledger, error) {
	// Create the default file type
	ft := fileTypes[s.Kind]
	filename := s.path.Join(name).ChangeExt(ft.Ext())
	f, err := os.OpenFile(filename.System(), os.O_RDWR|os.O_CREATE|os.O_EXCL|os.O_APPEND, 0660)
	if err != nil {
		return nil, err
	}
	log.I(ctx, "Created file record ledger: %v", filename)
	h, err := ft.Open(ctx, f, null)
	if err != nil {
		return nil, err
	}
	return NewLedger(ctx, h), nil
}

// tryOpen is used to attempt to open and read a ledger file.
// It returns an error only if the ledger file exists, but is not valid.
func (s *FileShelf) tryOpen(ctx context.Context, name string, null interface{}, ft fileType) (Ledger, error) {
	filename := s.path.Join(name).ChangeExt(ft.Ext())
	f, err := os.OpenFile(filename.System(), os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		return nil, nil
	}
	log.I(ctx, "Open file record ledger: %v", filename)
	h, err := ft.Open(ctx, f, null)
	if err != nil {
		return nil, err
	}
	return NewLedger(ctx, h), nil
}

func (r *readAt) Read(buf []byte) (int, error) {
	n, err := r.f.ReadAt(buf, r.offset)
	r.offset += int64(n)
	return n, err
}
