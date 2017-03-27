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

// Package server implements the rpc gpu debugger service, queriable by the
// clients, along with some helpers exposed via an http listener.
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/registry"
	"github.com/google/gapid/framework/binary/schema"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/gfxapi/all"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
)

// Config holds the server configuration settings.
type Config struct {
	Info           *service.ServerInfo
	StringTables   []*stringtable.StringTable
	AuthToken      auth.Token
	DeviceScanDone task.Signal
	LogBroadcaster *log.Broadcaster
}

// Server is the server interface to GAPIS.
type Server interface {
	service.Service
}

// New constructs and returns a new Server.
func New(ctx context.Context, cfg Config) Server {
	return &server{
		cfg.Info,
		cfg.StringTables,
		cfg.DeviceScanDone,
		cfg.LogBroadcaster,
		bytes.Buffer{},
	}
}

type server struct {
	info           *service.ServerInfo
	stbs           []*stringtable.StringTable
	deviceScanDone task.Signal
	logBroadcaster *log.Broadcaster
	profile        bytes.Buffer
}

func (s *server) GetServerInfo(ctx context.Context) (*service.ServerInfo, error) {
	return s.info, nil
}

func (s *server) GetSchema(ctx context.Context) (*schema.Message, error) {
	result := &schema.Message{}
	result.Entities = make([]*binary.Entity, 0, registry.Global.Count())
	all.GraphicsNamespace.Visit(func(c binary.Class) {
		entity := c.Schema()
		if entity != nil {
			result.Entities = append(result.Entities, entity)
		}
	})
	all.VisitConstantSets(func(c schema.ConstantSet) {
		result.Constants = append(result.Constants, c)
	})
	return result, nil
}

func (s *server) GetAvailableStringTables(ctx context.Context) ([]*stringtable.Info, error) {
	infos := make([]*stringtable.Info, len(s.stbs))
	for i, table := range s.stbs {
		infos[i] = table.Info
	}
	return infos, nil
}

func (s *server) GetStringTable(ctx context.Context, info *stringtable.Info) (*stringtable.StringTable, error) {
	for _, table := range s.stbs {
		if table.Info.CultureCode == info.CultureCode {
			return table, nil
		}
	}
	return nil, fmt.Errorf("String table not found")
}

func (s *server) ImportCapture(ctx context.Context, name string, data []uint8) (*path.Capture, error) {
	return capture.Import(ctx, name, bytes.NewReader(data))
}

func (s *server) LoadCapture(ctx context.Context, path string) (*path.Capture, error) {
	name := filepath.Base(path)
	in, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return capture.Import(ctx, name, in)
}

// Returns all devices, sorted by Android first, and then Host
func getSortedDevices(ctx context.Context) []bind.Device {
	all := bind.GetRegistry(ctx).Devices()
	androidDevices := make([]bind.Device, 0, len(all))
	nonAndroidDevices := make([]bind.Device, 0, len(all))
	for _, dev := range all {
		instance := dev.Instance()
		if instance.GetConfiguration().GetOS().GetKind() == device.Android {
			androidDevices = append(androidDevices, dev)
		} else {
			nonAndroidDevices = append(nonAndroidDevices, dev)
		}
	}
	return append(androidDevices, nonAndroidDevices...)
}

func (s *server) GetDevices(ctx context.Context) ([]*path.Device, error) {
	s.deviceScanDone.Wait(ctx)
	devices := getSortedDevices(ctx)
	paths := make([]*path.Device, len(devices))
	for i, d := range devices {
		paths[i] = path.NewDevice(d.Instance().Id.ID())
	}
	return paths, nil
}

func (s *server) GetDevicesForReplay(ctx context.Context, p *path.Capture) ([]*path.Device, error) {
	s.deviceScanDone.Wait(ctx)
	c, err := capture.ResolveFromPath(ctx, p)
	if err != nil {
		return nil, err
	}
	list, err := c.Atoms(ctx)
	if err != nil {
		return nil, err
	}

	state := c.NewState()

	// We expect the architecture atom to always be within the first 10
	// TODO(awoloszyn): Remove this once we have a proper file-header
	for i, a := range list.Atoms {
		if i > 10 {
			break
		}
		a.Mutate(ctx, state, nil /* no builder, just mutate */)
	}
	layout := state.MemoryLayout

	apis := make([]replay.Support, 0, len(c.Apis))
	for _, i := range c.Apis {
		api := gfxapi.Find(gfxapi.ID(i.ID()))
		if f, ok := api.(replay.Support); ok {
			apis = append(apis, f)
		}
	}

	all := getSortedDevices(ctx)
	filtered := make([]bind.Device, 0, len(all))
nextDevice:
	for _, device := range all {
		instance := device.Instance()
		for _, api := range apis {
			// TODO: Check if device is a LAD, and if so filter by supportsLAD.
			ctx := log.V{
				"api":    fmt.Sprintf("%T", api),
				"device": instance,
			}.Bind(ctx)
			if api.CanReplayOn(ctx, instance, layout) {
				log.I(ctx, "Compatible")
			} else {
				log.I(ctx, "Incompatible")
				continue nextDevice
			}
		}
		filtered = append(filtered, device)
	}

	paths := make([]*path.Device, len(filtered))
	for i, d := range filtered {
		paths[i] = path.NewDevice(d.Instance().Id.ID())
	}
	return paths, nil
}

func (s *server) GetFramebufferAttachment(
	ctx context.Context,
	device *path.Device,
	after *path.Command,
	attachment gfxapi.FramebufferAttachment,
	settings *service.RenderSettings) (*path.ImageInfo, error) {

	// TODO: Path validation
	// if err := device.Validate(); err != nil {
	// 	return nil, err
	// }
	// if err := after.Validate(); err != nil {
	// 	return nil, err
	// }
	return resolve.FramebufferAttachment(ctx, device, after, attachment, settings)
}

func (s *server) Get(ctx context.Context, p *path.Any) (interface{}, error) {
	// TODO: Path validation
	// if err := p.Validate(); err != nil {
	// 	return nil, err
	// }
	v, err := resolve.Get(ctx, p)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (s *server) Set(ctx context.Context, p *path.Any, v interface{}) (*path.Any, error) {
	// TODO: Path validation
	// if err := p.Validate(); err != nil {
	// 	return nil, err
	// }
	return resolve.Set(ctx, p, v)
}

func (s *server) Follow(ctx context.Context, p *path.Any) (*path.Any, error) {
	// TODO: Path validation
	// if err := p.Validate(); err != nil {
	// 	return nil, err
	// }
	return resolve.Follow(ctx, p)
}

func (s *server) BeginCPUProfile(ctx context.Context) error {
	s.profile.Reset()
	return pprof.StartCPUProfile(&s.profile)
}

func (s *server) EndCPUProfile(ctx context.Context) ([]byte, error) {
	pprof.StopCPUProfile()
	return s.profile.Bytes(), nil
}

func (s *server) GetPerformanceCounters(ctx context.Context) ([]byte, error) {
	return json.Marshal(benchmark.GlobalCounters)
}

func (s *server) GetProfile(ctx context.Context, name string, debug int32) ([]byte, error) {
	p := pprof.Lookup(name)
	if p == nil {
		return []byte{}, fmt.Errorf("Profile not found: %s", name)
	}
	var b bytes.Buffer
	if err := p.WriteTo(&b, int(debug)); err != nil {
		return []byte{}, err
	}
	return b.Bytes(), nil
}

func (s *server) GetLogStream(ctx context.Context, handler log.Handler) error {
	unregister := s.logBroadcaster.Listen(handler)
	defer unregister()
	<-task.ShouldStop(ctx)
	return task.StopReason(ctx)
}
