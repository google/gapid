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
	"io/ioutil"
	"path/filepath"
	"runtime/pprof"
	"sort"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"

	// Register all the gfxapis
	_ "github.com/google/gapid/gapis/gfxapi/all"
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
	return capture.Import(ctx, name, data)
}

func (s *server) ExportCapture(ctx context.Context, c *path.Capture) ([]byte, error) {
	b := bytes.Buffer{}
	if err := capture.Export(ctx, c, &b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (s *server) LoadCapture(ctx context.Context, path string) (*path.Capture, error) {
	name := filepath.Base(path)
	in, err := ioutil.ReadFile(path)
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

type prioritizedDevice struct {
	device   bind.Device
	priority uint32
}

type prioritizedDevices []prioritizedDevice

func (p prioritizedDevices) Len() int {
	return len(p)
}

func (p prioritizedDevices) Less(i, j int) bool {
	return p[i].priority < p[j].priority
}

func (p prioritizedDevices) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (s *server) GetDevicesForReplay(ctx context.Context, p *path.Capture) ([]*path.Device, error) {
	s.deviceScanDone.Wait(ctx)
	c, err := capture.ResolveFromPath(ctx, p)
	if err != nil {
		return nil, err
	}

	state := c.NewState()

	apis := make([]replay.Support, 0, len(c.APIs))
	for _, i := range c.APIs {
		api := gfxapi.Find(gfxapi.ID(i.ID()))
		if f, ok := api.(replay.Support); ok {
			apis = append(apis, f)
		}
	}

	all := getSortedDevices(ctx)
	filtered := make([]prioritizedDevice, 0, len(all))
	for _, device := range all {
		instance := device.Instance()
		p := uint32(1)
		for _, api := range apis {
			// TODO: Check if device is a LAD, and if so filter by supportsLAD.
			ctx := log.V{
				"api":    fmt.Sprintf("%T", api),
				"device": instance.Name,
			}.Bind(ctx)
			priority := api.GetReplayPriority(ctx, instance, state.MemoryLayout)
			p = p * priority
			if priority != 0 {
				log.I(ctx, "Compatible %d", priority)
			} else {
				log.I(ctx, "Incompatible")
			}
		}
		if p > 0 {
			ctx := log.V{
				"device": instance,
			}.Bind(ctx)
			log.I(ctx, "Priority %d", p)
			filtered = append(filtered, prioritizedDevice{device, p})
		}
	}

	sort.Sort(prioritizedDevices(filtered))

	paths := make([]*path.Device, len(filtered))
	for i, d := range filtered {
		paths[i] = path.NewDevice(d.device.Instance().Id.ID())
	}
	return paths, nil
}

func (s *server) GetFramebufferAttachment(
	ctx context.Context,
	device *path.Device,
	after *path.Command,
	attachment gfxapi.FramebufferAttachment,
	settings *service.RenderSettings,
	hints *service.UsageHints) (*path.ImageInfo, error) {

	if err := device.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", device.Text())
	}
	if err := after.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", after.Text())
	}
	return resolve.FramebufferAttachment(ctx, device, after, attachment, settings, hints)
}

func (s *server) Get(ctx context.Context, p *path.Any) (interface{}, error) {
	if err := p.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", p.Text())
	}
	v, err := resolve.Get(ctx, p)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (s *server) Set(ctx context.Context, p *path.Any, v interface{}) (*path.Any, error) {
	if err := p.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", p.Text())
	}
	return resolve.Set(ctx, p, v)
}

func (s *server) Follow(ctx context.Context, p *path.Any) (*path.Any, error) {
	if err := p.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", p.Text())
	}
	return resolve.Follow(ctx, p)
}

func (s *server) GetLogStream(ctx context.Context, handler log.Handler) error {
	handler = log.Channel(handler, 64)
	unregister := s.logBroadcaster.Listen(handler)
	defer func() {
		unregister()
		handler.Close()
	}()
	<-task.ShouldStop(ctx)
	return task.StopReason(ctx)
}

func (s *server) Find(ctx context.Context, req *service.FindRequest, handler service.FindHandler) error {
	return resolve.Find(ctx, req, handler)
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
