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
	"io/ioutil"
	"reflect"
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

func (a *artifacts) getID(ctx context.Context, getter func(ctx context.Context) (*zip.File, error), name string) (string, error) {
	f, err := getter(ctx)
	if err != nil {
		return "", log.Err(ctx, err, name+" not in zip file")
	}

	// we need to upload the content to stash to get an ID
	r, err := f.Open()
	if err != nil {
		return "", log.Err(ctx, err, "failed to open zip entry for "+name)
	}
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", log.Err(ctx, err, "failed to read zip entry for "+name)
	}

	info := stash.Upload{
		Name:       []string{f.Name},
		Executable: f.Mode()&0111 != 0,
	}
	id, err := a.store.UploadBytes(ctx, info, b)
	if err != nil {
		return "", log.Err(ctx, err, "failed to upload "+name)
	}
	return id, nil
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
	l := layout.NewZipLayout(zipFile, builderAbi.OS)

	toolSet := ToolSet{Abi: builderAbi, Host: new(HostToolSet)}
	if toolSet.Host.Gapir, err = a.getID(ctx, func(ctx context.Context) (*zip.File, error) {
		return l.Gapir(ctx, nil)
	}, "gapir"); err != nil {
		return nil, err
	}
	if toolSet.Host.Gapis, err = a.getID(ctx, l.Gapis, "gapis"); err != nil {
		return nil, err
	}
	if toolSet.Host.Gapit, err = a.getID(ctx, l.Gapit, "gapit"); err != nil {
		return nil, err
	}
	if toolSet.Host.VirtualSwapChainLib, err = a.getID(ctx, func(ctx context.Context) (*zip.File, error) {
		return l.Library(ctx, layout.LibVirtualSwapChain, nil)
	}, "libVirtualSwapChain"); err != nil {
		return nil, err
	}
	if toolSet.Host.VirtualSwapChainJson, err = a.getID(ctx, func(ctx context.Context) (*zip.File, error) {
		return l.Json(ctx, layout.LibVirtualSwapChain)
	}, "libVirtualSwapChain JSON"); err != nil {
		return nil, err
	}
	for _, abi := range []*device.ABI{device.AndroidARMv7a, device.AndroidARM64v8a, device.AndroidX86} {
		id, err := a.getID(ctx, func(ctx context.Context) (*zip.File, error) {
			return l.GapidApk(ctx, abi)
		}, abi.Name+"gapid APK")
		if err != nil {
			return nil, err
		}
		toolSet.Android = append(toolSet.Android, &AndroidToolSet{
			Abi:      abi,
			GapidApk: id,
		})
	}

	entry := &Artifact{Id: id, Tool: &toolSet}
	a.ledger.Add(ctx, entry)
	return entry, nil
}
