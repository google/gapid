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

package capture

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/pkg/errors"
)

// The list of captures currently imported.
// TODO: This needs to be moved to persistent storage.
var (
	capturesLock sync.RWMutex
	captures     = []id.ID{}
)

// Capture represents data from a trace.
type Capture interface {
	// Name returns the name of the capture.
	Name() string
	// Service returns the service.Capture description for this capture.
	Service(ctx context.Context, p *path.Capture) *service.Capture
	// Export exports the capture in binary format to the given writer.
	Export(ctx context.Context, w io.Writer) error
}

func init() {
	protoconv.Register(toProtoWrapped, fromProtoWrapped)
}

// New returns a path to a new capture stored in the database.
func New(ctx context.Context, c Capture) (*path.Capture, error) {
	id, err := database.Store(ctx, wrapper{c})
	if err != nil {
		return nil, err
	}

	capturesLock.Lock()
	captures = append(captures, id)
	capturesLock.Unlock()

	return &path.Capture{ID: path.NewID(id)}, nil
}

// Captures returns all the captures stored by the database by identifier.
func Captures() []*path.Capture {
	capturesLock.RLock()
	defer capturesLock.RUnlock()
	out := make([]*path.Capture, len(captures))
	for i, c := range captures {
		out[i] = &path.Capture{ID: path.NewID(c)}
	}
	return out
}

// ResolveFromID resolves a single capture with the ID id.
func ResolveFromID(ctx context.Context, id id.ID) (Capture, error) {
	obj, err := database.Resolve(ctx, id)
	if err != nil {
		return nil, log.Err(ctx, err, "Error resolving capture")
	}
	return obj.(wrapper).c, nil
}

// ResolveGraphicsFromID resolves a single graphics capture with the ID id.
func ResolveGraphicsFromID(ctx context.Context, id id.ID) (*GraphicsCapture, error) {
	c, err := ResolveFromID(ctx, id)
	if err != nil {
		return nil, err
	}
	if gc, ok := c.(*GraphicsCapture); ok {
		return gc, nil
	}
	return nil, errors.New("not a graphics capture")
}

// ResolvePerfettoFromID resolves a single perfetto capture with the ID id.
func ResolvePerfettoFromID(ctx context.Context, id id.ID) (*PerfettoCapture, error) {
	c, err := ResolveFromID(ctx, id)
	if err != nil {
		return nil, err
	}
	if pc, ok := c.(*PerfettoCapture); ok {
		return pc, nil
	}
	return nil, errors.New("not a Perfetto capture")
}

// ResolveFromPath resolves a single capture with the path p.
func ResolveFromPath(ctx context.Context, p *path.Capture) (Capture, error) {
	return ResolveFromID(ctx, p.ID.ID())
}

// ResolveGraphicsFromPath resolves a single graphics capture with the path p.
func ResolveGraphicsFromPath(ctx context.Context, p *path.Capture) (*GraphicsCapture, error) {
	return ResolveGraphicsFromID(ctx, p.ID.ID())
}

// ResolvePerfettoFromPath resolves a single perfetto capture with the path p.
func ResolvePerfettoFromPath(ctx context.Context, p *path.Capture) (*PerfettoCapture, error) {
	return ResolvePerfettoFromID(ctx, p.ID.ID())
}

// Import imports the capture by name and data, and stores it in the database.
func Import(ctx context.Context, name string, key string, src Source) (*path.Capture, error) {
	dataID, err := database.Store(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("Unable to store capture data source: %v", err)
	}

	id, err := database.Store(ctx, &Record{
		Key:  key,
		Name: name,
		Data: dataID[:],
	})
	if err != nil {
		return nil, err
	}

	capturesLock.Lock()
	captures = append(captures, id)
	capturesLock.Unlock()

	return &path.Capture{ID: path.NewID(id)}, nil
}

// Export encodes the given capture and associated resources
// and writes it to the supplied io.Writer in the pack file format,
// producing output suitable for use with Import or opening in the trace editor.
func Export(ctx context.Context, p *path.Capture, w io.Writer) error {
	c, err := ResolveFromPath(ctx, p)
	if err != nil {
		return err
	}
	return c.Export(ctx, w)
}

// Source represents the source of capture data.
type Source interface {
	// ReadCloser returns an io.ReadCloser instance, from which capture data
	// can be read and closed after reading.
	ReadCloser() (io.ReadCloser, error)
	// Size returns the total size in bytes of this source.
	Size() (uint64, error)
}

// blobReadCloser implements the Source interface, it represents the capture
// data in raw byte form.
type blobReadCloser struct {
	*bytes.Reader
}

// Close implements the io.ReadCloser interface.
func (blobReadCloser) Close() error { return nil }

// ReadCloser implements the Source interface.
func (b *Blob) ReadCloser() (io.ReadCloser, error) {
	return blobReadCloser{bytes.NewReader(b.GetData())}, nil
}

// Size implements the Source interface.
func (b *Blob) Size() (uint64, error) {
	return uint64(len(b.GetData())), nil
}

// ReadCloser implements the Source interface.
func (f *File) ReadCloser() (io.ReadCloser, error) {
	o, err := os.Open(f.GetPath())
	if err != nil {
		return nil, &service.ErrDataUnavailable{
			Reason: messages.ErrFileCannotBeRead(),
		}
	}
	return o, nil
}

// Size implements the Source interface.
func (f *File) Size() (uint64, error) {
	fi, err := os.Stat(f.GetPath())
	if err != nil {
		return 0, err
	}
	return uint64(fi.Size()), nil
}

func toProto(ctx context.Context, c Capture) (*Record, error) {
	buf := bytes.Buffer{}
	if err := c.Export(ctx, &buf); err != nil {
		return nil, err
	}
	id, err := database.Store(ctx, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("Unable to store capture data: %v", err)
	}
	return &Record{
		Name: c.Name(),
		Data: id[:],
	}, nil
}

type loggingRC struct {
	onProgress func(uint64)
	total      uint64
	rc         io.ReadCloser
}

// Read implements the io.Reader interface.
func (l *loggingRC) Read(p []byte) (n int, err error) {
	n, err = l.rc.Read(p)
	l.total += uint64(n)
	l.onProgress(l.total)
	return n, err
}

// Close implements the io.Closer interface.
func (l *loggingRC) Close() error {
	return l.rc.Close()
}

func open(ctx context.Context, src Source) (r *bufio.Reader, close func() error, err error) {
	in, err := src.ReadCloser()
	if err != nil {
		return nil, nil, err
	}
	if size, err := src.Size(); err == nil {
		in = &loggingRC{
			onProgress: func(p uint64) { status.UpdateProgress(ctx, p, size) },
			rc:         in,
		}
	}
	return bufio.NewReader(in), in.Close, nil
}

func fromProto(ctx context.Context, r *Record) (Capture, error) {
	ctx = status.Start(ctx, "Loading capture '%v'", r.Name)
	defer status.Finish(ctx)

	var dataID id.ID
	copy(dataID[:], r.Data)
	data, err := database.Resolve(ctx, dataID)
	if err != nil {
		return nil, fmt.Errorf("Unable to load capture data source: %v", err)
	}
	src, ok := data.(Source)
	if !ok {
		return nil, fmt.Errorf("Unable to load capture data source: Failed to resolve capture.Source")
	}

	in, close, err := open(ctx, src)
	if err != nil {
		return nil, err
	}
	defer close()

	switch {
	case isGFXTraceFormat(in):
		return deserializeGFXTrace(ctx, r, in)
	case isPerfettoTraceFormat(in):
		return deserializePerfettoTrace(ctx, r, in)
	default:
		return nil, fmt.Errorf("Not a recognized capture format")
	}
}

// wrapper wraps a Capture. This is needed because protoconv, which is used by
// the database, doesn't support interfaces.
type wrapper struct {
	c Capture
}

func toProtoWrapped(ctx context.Context, w wrapper) (*Record, error) {
	return toProto(ctx, w.c)
}

func fromProtoWrapped(ctx context.Context, r *Record) (wrapper, error) {
	c, err := fromProto(ctx, r)
	return wrapper{c}, err
}
