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
	"reflect"
	"strings"
	"sync"

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

var toolPatterns = []*ToolSet{{
	Abi:                 device.AndroidARMv7a,
	GapidApk:            "android/armeabi-v7a/gapid.apk",
	VirtualSwapChainLib: "android/armeabi-v7a/libVkLayer_VirtualSwapchain.so",
}, {
	Abi:                 device.AndroidARM64v8a,
	GapidApk:            "android/arm64-v8a/gapid.apk",
	VirtualSwapChainLib: "android/arm64-v8a/libVkLayer_VirtualSwapchain.so",
}, {
	Abi:                 device.AndroidX86_64,
	GapidApk:            "android/x86/gapid.apk",
	VirtualSwapChainLib: "android/x86/libVkLayer_VirtualSwapchain.so",
}, {
	Abi:                  device.LinuxX86_64,
	Gapir:                "linux/x86_64/gapir",
	Gapis:                "linux/x86_64/gapis",
	Gapit:                "linux/x86_64/gapit",
	VirtualSwapChainLib:  "linux/x86_64/libVkLayer_VirtualSwapchain.so",
	VirtualSwapChainJson: "linux/x86_64/VirtualSwapchainLayer.json",
}, {
	Abi:   device.OSXX86_64,
	Gapir: "osx/x86_64/gapir",
	Gapis: "osx/x86_64/gapis",
	Gapit: "osx/x86_64/gapit",
	// TODO(baldwinn): These should not be required, https://github.com/google/gapid/issues/570
	VirtualSwapChainLib:  "osx/x86_64/libVkLayer_VirtualSwapchain.so",
	VirtualSwapChainJson: "osx/x86_64/VirtualSwapchainLayer.json",
}, {
	Abi:                  device.WindowsX86_64,
	Gapir:                "windows/x86_64/gapir",
	Gapis:                "windows/x86_64/gapis",
	Gapit:                "windows/x86_64/gapit",
	VirtualSwapChainLib:  "windows/x86_64/libVkLayer_VirtualSwapchain.so",
	VirtualSwapChainJson: "windows/x86_64/VirtualSwapchainLayer.json",
}}

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

func toolIsEmpty(t *ToolSet) bool {
	return t.GapidApk == "" &&
		t.Gapii == "" &&
		t.Gapir == "" &&
		t.Gapis == "" &&
		t.Gapit == "" &&
		t.VirtualSwapChainLib == "" &&
		t.VirtualSwapChainJson == ""
}

func (a *artifacts) matchTool(ctx context.Context, z *zipEntry, pattern string, target *string) {
	if pattern == "" {
		return
	}
	if !strings.HasSuffix(z.file.Name, pattern) {
		return
	}
	*target = z.GetID(ctx, a)
}

func (a *artifacts) matchTools(ctx context.Context, z *zipEntry, pattern *ToolSet, target *ToolSet) {
	a.matchTool(ctx, z, pattern.Interceptor, &target.Interceptor)
	a.matchTool(ctx, z, pattern.Gapii, &target.Gapii)
	a.matchTool(ctx, z, pattern.Gapir, &target.Gapir)
	a.matchTool(ctx, z, pattern.Gapis, &target.Gapis)
	a.matchTool(ctx, z, pattern.Gapit, &target.Gapit)
	a.matchTool(ctx, z, pattern.GapidApk, &target.GapidApk)
	a.matchTool(ctx, z, pattern.VirtualSwapChainLib, &target.VirtualSwapChainLib)
	a.matchTool(ctx, z, pattern.VirtualSwapChainJson, &target.VirtualSwapChainJson)
}

func (a *artifacts) get(ctx context.Context, id string) (*Artifact, error) {
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
	tools := make([]*ToolSet, len(toolPatterns))
	for i := range tools {
		tools[i] = &ToolSet{
			Abi: toolPatterns[i].Abi,
		}
	}
	for _, f := range zipFile.File {
		z := &zipEntry{file: f}
		for i, p := range toolPatterns {
			a.matchTools(ctx, z, p, tools[i])
		}
	}
	entry := &Artifact{Id: id}
	for _, t := range tools {
		if !toolIsEmpty(t) {
			entry.Tool = append(entry.Tool, t)
		}
	}
	a.ledger.Add(ctx, entry)
	return entry, nil
}
