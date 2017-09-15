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

package build

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/test/robot/record"
	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/search/eval"
	"github.com/google/gapid/test/robot/stash"
)

type artifacts struct {
	mu      sync.Mutex
	store   *stash.Client
	ledger  record.Ledger
	entries []*Artifact
	byID    map[string]*Artifact
	onAdd   event.Broadcast
}

func (a *artifacts) init(ctx context.Context, store *stash.Client, library record.Library) error {
	ledger, err := library.Open(ctx, "artifacts", &Artifact{})
	if err != nil {
		return err
	}
	a.store = store
	a.ledger = ledger
	a.byID = map[string]*Artifact{}
	apply := event.AsHandler(ctx, a.apply)
	if err := ledger.Read(ctx, apply); err != nil {
		return err
	}
	ledger.Watch(ctx, apply)
	return nil
}

// apply is called with items coming out of the ledger
// it should be called with the mutation lock already held.
func (a *artifacts) apply(ctx context.Context, artifact *Artifact) error {
	old := a.byID[artifact.Id]
	if old == nil {
		a.entries = append(a.entries, artifact)
		a.byID[artifact.Id] = artifact
	} else {
		*old = *artifact
	}
	a.onAdd.Send(ctx, artifact)
	return nil
}

func (a *artifacts) search(ctx context.Context, query *search.Query, handler ArtifactHandler) error {
	filter := eval.Filter(ctx, query, reflect.TypeOf(&Artifact{}), event.AsHandler(ctx, handler))
	initial := event.AsProducer(ctx, a.entries)
	if query.Monitor {
		return event.Monitor(ctx, &a.mu, a.onAdd.Listen, initial, filter)
	}
	return event.Feed(ctx, filter, initial)
}

type zipEntry struct {
	file *zip.File
	id   string
}

type zipReader struct {
	file *zip.File
	r    io.ReadCloser
}

func (z *zipEntry) Open() (*zipReader, error) {
	r := &zipReader{file: z.file}
	return r, r.Reset()
}

func (z *zipReader) Read(data []byte) (int, error) {
	return z.r.Read(data)
}

func (z *zipReader) Reset() error {
	if err := z.Close(); err != nil {
		return err
	}
	r, err := z.file.Open()
	if err != nil {
		return err
	}
	z.r = r
	return nil
}

func (z *zipReader) Close() error {
	if z.r == nil {
		return nil
	}
	err := z.r.Close()
	z.r = nil
	return err
}

func (z *zipEntry) GetID(ctx context.Context, a *artifacts) string {
	if z.id != "" {
		return z.id
	}
	// we need to upload the content to stash to get an ID
	r, err := z.Open()
	if err != nil {
		return ""
	}
	defer r.Close()
	info := stash.Upload{
		Name:       []string{z.file.Name},
		Executable: z.file.Mode()&0111 != 0,
	}
	id, err := a.store.UploadStream(ctx, info, r)
	if err != nil {
		return ""
	}
	z.id = id
	return z.id
}

func (a *artifacts) get(ctx context.Context, id string, builderAbi *device.ABI) (*Artifact, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if entry := a.byID[id]; entry != nil {
		return entry, nil
	}
	data, err := a.store.Read(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, log.Err(ctx, nil, "Build not in the stash")
	}
	zipFile, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, log.Err(ctx, nil, "File is not a build artifact.")
	}
	toolSet := ToolSet{Abi: builderAbi, Host: new(HostToolSet)}
	// TODO(baldwinn): move these paths to layout
	toolSetIDByZipEntry := map[string]*string{
		"gapid/gapir":                          &toolSet.Host.Gapir,
		"gapid/gapis":                          &toolSet.Host.Gapis,
		"gapid/gapit":                          &toolSet.Host.Gapit,
		"gapid/libVkLayer_VirtualSwapchain.so": &toolSet.Host.VirtualSwapChainLib,
		"gapid/VirtualSwapchainLayer.json":     &toolSet.Host.VirtualSwapChainJson,
	}
	for _, f := range zipFile.File {
		f.Name = filepath.ToSlash(f.Name)
		z := &zipEntry{file: f}

		if dirs := strings.Split(f.Name, "/"); strings.HasPrefix(dirs[1], "android") {
			androidTool := &AndroidToolSet{Abi: device.ABIByName(layout.DirToBinABI(dirs[1])), GapidApk: z.GetID(ctx, a)}
			toolSet.Android = append(toolSet.Android, androidTool)
		} else if toolID, ok := toolSetIDByZipEntry[f.Name]; ok {
			*toolID = z.GetID(ctx, a)
		}
	}
	entry := &Artifact{Id: id, Tool: &toolSet}
	a.ledger.Add(ctx, entry)
	return entry, nil
}
