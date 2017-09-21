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
	"context"
	"reflect"
	"sync"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/test/robot/record"
	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/search/eval"
)

var packageClass = reflect.TypeOf(&Package{})

type packages struct {
	mu       sync.Mutex
	ledger   record.Ledger
	entries  []*Package
	byID     map[string]*Package
	byName   map[string]*Package
	onChange event.Broadcast
}

func (p *packages) init(ctx context.Context, library record.Library) error {
	ledger, err := library.Open(ctx, "packages", &Package{})
	if err != nil {
		return err
	}
	p.ledger = ledger
	p.byID = map[string]*Package{}
	apply := event.AsHandler(ctx, p.apply)
	if err := ledger.Read(ctx, apply); err != nil {
		return err
	}
	ledger.Watch(ctx, apply)
	return nil
}

// apply is called with items coming out of the ledger
// it should be called with the mutation lock already held.
func (p *packages) apply(ctx context.Context, pkg *Package) error {
	old := p.byID[pkg.Id]
	if old == nil {
		p.entries = append(p.entries, pkg)
		p.byID[pkg.Id] = pkg
		p.onChange.Send(ctx, pkg)
		return nil
	}
	if pkg.Parent != "" {
		old.Parent = pkg.Parent
	}
	if pkg.Information != nil {
		// description is the only thing we allow to be edited
		if pkg.Information.Description != "" {
			old.Information.Description = pkg.Information.Description
		}
	}
	if len(pkg.Artifact) > 0 {
		old.Artifact = append(old.Artifact, pkg.Artifact...)
	}
	for _, t := range pkg.Tool {
		old.mergeTool(ctx, t)
	}
	p.onChange.Send(ctx, old)
	return nil
}

func (p *packages) search(ctx context.Context, query *search.Query, handler PackageHandler) error {
	filter := eval.Filter(ctx, query, packageClass, event.AsHandler(ctx, handler))
	initial := event.AsProducer(ctx, p.entries)
	if query.Monitor {
		return event.Monitor(ctx, &p.mu, p.onChange.Listen, initial, filter)
	}
	return event.Feed(ctx, filter, initial)
}

func (p *packages) update(ctx context.Context, pkg *Package) (*Package, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, found := p.byID[pkg.Id]; !found {
		return nil, log.Err(ctx, nil, "Package not found")
	}
	if err := p.ledger.Add(ctx, pkg); err != nil {
		return nil, err
	}
	return p.byID[pkg.Id], nil
}

func (p *packages) addArtifact(ctx context.Context, a *Artifact, info *Information) (*Package, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	old := p.findArtifactPackage(ctx, a, info)
	pkg := &Package{
		Information: info,
		Artifact:    []string{a.Id},
		Tool:        []*ToolSet{a.Tool},
	}
	merged := false
	if old != nil {
		pkg.Id = old.Id
		merged = true
	} else {
		pkg.Id = id.Unique().String()
	}
	if err := p.ledger.Add(ctx, pkg); err != nil {
		return nil, false, err
	}
	return p.byID[pkg.Id], merged, nil
}

func (p *packages) findArtifactPackage(ctx context.Context, a *Artifact, info *Information) *Package {
	// Search for a set to merge into
	if info.Tag != "" {
		// if we have a tag, that's the only thing that matters
		for _, pkg := range p.entries {
			if pkg.Information.Tag == info.Tag {
				return pkg
			}
		}
	}
	// if we don't have a tag, local builds never merge
	if info.Type == Local {
		return nil
	}
	if info.Cl != "" {
		// non local builds with a matching cl can be merged
		for _, pkg := range p.entries {
			if pkg.Information.Cl == info.Cl {
				return pkg
			}
		}
	}
	// No match, cannot merge
	return nil
}

func (pkg *Package) mergeTool(ctx context.Context, tool *ToolSet) {
	for _, t := range pkg.Tool {
		if t.Abi.SameAs(tool.Abi) {
			// merge into existing tool
			if tool.Host.Gapir != "" {
				t.Host.Gapir = tool.Host.Gapir
			}
			if tool.Host.Gapis != "" {
				t.Host.Gapis = tool.Host.Gapis
			}
			if tool.Host.Gapit != "" {
				t.Host.Gapit = tool.Host.Gapit
			}
			for _, android := range tool.Android {
				if android.GapidApk != "" {
					if a := t.FindAndroidToolSet(android.Abi); a != nil {
						a.GapidApk = android.GapidApk
					} else {
						t.Android = append(t.Android, android)
					}
				}
			}
			return
		}
	}
	// no matching tool, so just append it
	pkg.Tool = append(pkg.Tool, tool)
}

// GetHostTools will return the toolset that match the abi, if there is one.
func (pkg *Package) GetHostTools(abi *device.ABI) *ToolSet {
	for _, t := range pkg.Tool {
		if t.Abi.SameAs(abi) {
			return t
		}
	}
	return nil
}

// FindAndroidToolSet will return the toolset for a target ABI, if there is one.
func (t *ToolSet) FindAndroidToolSet(abi *device.ABI) *AndroidToolSet {
	for _, a := range t.Android {
		if a.Abi.SameAs(abi) {
			return a
		}
	}
	return nil
}

// GetTargetTools will return the toolset for a target ABI that was build by a host ABI, if there is one.
func (pkg *Package) GetTargetTools(hostAbi *device.ABI, targetAbi *device.ABI) *AndroidToolSet {
	hostTools := pkg.GetHostTools(hostAbi)
	if hostTools == nil {
		return nil
	}

	return hostTools.FindAndroidToolSet(targetAbi)
}
