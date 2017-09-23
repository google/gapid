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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay/devices"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"

	"github.com/google/go-github/github"

	// Register all the apis
	_ "github.com/google/gapid/gapis/api/all"
)

// Config holds the server configuration settings.
type Config struct {
	Info           *service.ServerInfo
	StringTables   []*stringtable.StringTable
	AuthToken      auth.Token
	DeviceScanDone task.Signal
	LogBroadcaster *log.Broadcaster
	IdleTimeout    time.Duration
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

func (s *server) Ping(ctx context.Context) error {
	return nil
}

func (s *server) GetServerInfo(ctx context.Context) (*service.ServerInfo, error) {
	ctx = log.Enter(ctx, "GetServerInfo")
	return s.info, nil
}

func (s *server) CheckForUpdates(ctx context.Context, includePrereleases bool) (*service.Release, error) {
	const (
		githubOrg  = "google"
		githubRepo = "gapid"
	)
	ctx = log.Enter(ctx, "CheckForUpdates")
	client := github.NewClient(nil)
	options := &github.ListOptions{}
	releases, _, err := client.Repositories.ListReleases(ctx, githubOrg, githubRepo, options)
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to list releases")
	}
	var mostRecent *service.Release
	mostRecentVersion := app.Version
	for _, release := range releases {
		if !includePrereleases && release.GetPrerelease() {
			continue
		}
		var version app.VersionSpec
		fmt.Sscanf(release.GetTagName(), "v%d.%d.%d", &version.Major, &version.Minor, &version.Point)
		if version.GreaterThan(mostRecentVersion) {
			mostRecent = &service.Release{
				Name:         release.GetName(),
				VersionMajor: uint32(version.Major),
				VersionMinor: uint32(version.Minor),
				VersionPoint: uint32(version.Point),
				Prerelease:   release.GetPrerelease(),
				BrowserUrl:   release.GetHTMLURL(),
			}
			mostRecentVersion = version
		}
	}
	if mostRecent == nil {
		return nil, &service.ErrDataUnavailable{
			Reason:    messages.NoNewBuildsAvailable(),
			Transient: true,
		}
	}
	return mostRecent, nil
}

func (s *server) GetAvailableStringTables(ctx context.Context) ([]*stringtable.Info, error) {
	ctx = log.Enter(ctx, "GetAvailableStringTables")
	infos := make([]*stringtable.Info, len(s.stbs))
	for i, table := range s.stbs {
		infos[i] = table.Info
	}
	return infos, nil
}

func (s *server) GetStringTable(ctx context.Context, info *stringtable.Info) (*stringtable.StringTable, error) {
	ctx = log.Enter(ctx, "GetStringTable")
	for _, table := range s.stbs {
		if table.Info.CultureCode == info.CultureCode {
			return table, nil
		}
	}
	return nil, fmt.Errorf("String table not found")
}

func (s *server) ImportCapture(ctx context.Context, name string, data []uint8) (*path.Capture, error) {
	ctx = log.Enter(ctx, "ImportCapture")
	p, err := capture.Import(ctx, name, data)
	if err != nil {
		return nil, err
	}
	// Ensure the capture can be read by resolving it now.
	if _, err = capture.ResolveFromPath(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *server) ExportCapture(ctx context.Context, c *path.Capture) ([]byte, error) {
	ctx = log.Enter(ctx, "ExportCapture")
	b := bytes.Buffer{}
	if err := capture.Export(ctx, c, &b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (s *server) LoadCapture(ctx context.Context, path string) (*path.Capture, error) {
	ctx = log.Enter(ctx, "LoadCapture")
	name := filepath.Base(path)
	in, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	p, err := capture.Import(ctx, name, in)
	if err != nil {
		return nil, err
	}
	// Ensure the capture can be read by resolving it now.
	if _, err = capture.ResolveFromPath(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *server) GetDevices(ctx context.Context) ([]*path.Device, error) {
	ctx = log.Enter(ctx, "GetDevices")
	s.deviceScanDone.Wait(ctx)
	devices := bind.GetRegistry(ctx).Devices()
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
	ctx = log.Enter(ctx, "GetDevicesForReplay")
	s.deviceScanDone.Wait(ctx)
	return devices.ForReplay(ctx, p)
}

func (s *server) GetFramebufferAttachment(
	ctx context.Context,
	device *path.Device,
	after *path.Command,
	attachment api.FramebufferAttachment,
	settings *service.RenderSettings,
	hints *service.UsageHints) (*path.ImageInfo, error) {

	ctx = log.Enter(ctx, "GetFramebufferAttachment")
	if err := device.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", device)
	}
	if err := after.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", after)
	}
	return resolve.FramebufferAttachment(ctx, device, after, attachment, settings, hints)
}

func (s *server) Get(ctx context.Context, p *path.Any) (interface{}, error) {
	ctx = log.Enter(ctx, "Get")
	if err := p.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", p)
	}
	v, err := resolve.Get(ctx, p)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (s *server) Set(ctx context.Context, p *path.Any, v interface{}) (*path.Any, error) {
	ctx = log.Enter(ctx, "Set")
	if err := p.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", p)
	}
	return resolve.Set(ctx, p, v)
}

func (s *server) Follow(ctx context.Context, p *path.Any) (*path.Any, error) {
	ctx = log.Enter(ctx, "Follow")
	if err := p.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", p)
	}
	return resolve.Follow(ctx, p)
}

func (s *server) GetLogStream(ctx context.Context, handler log.Handler) error {
	ctx = log.Enter(ctx, "GetLogStream")
	closed := make(chan struct{})
	handler = log.OnClosed(handler, func() { close(closed) })
	handler = log.Channel(handler, 64)
	unregister := s.logBroadcaster.Listen(handler)
	defer unregister()
	select {
	case <-closed:
		// Logs were closed - likely server is shutting down.
	case <-task.ShouldStop(ctx):
		// Context was stopped - likely client has disconnected.
	}
	return task.StopReason(ctx)
}

func (s *server) Find(ctx context.Context, req *service.FindRequest, handler service.FindHandler) error {
	ctx = log.Enter(ctx, "Find")
	return resolve.Find(ctx, req, handler)
}

func (s *server) BeginCPUProfile(ctx context.Context) error {
	ctx = log.Enter(ctx, "BeginCPUProfile")
	s.profile.Reset()
	return pprof.StartCPUProfile(&s.profile)
}

func (s *server) EndCPUProfile(ctx context.Context) ([]byte, error) {
	ctx = log.Enter(ctx, "EndCPUProfile")
	pprof.StopCPUProfile()
	return s.profile.Bytes(), nil
}

func (s *server) GetPerformanceCounters(ctx context.Context) (string, error) {
	ctx = log.Enter(ctx, "GetPerformanceCounters")
	return fmt.Sprintf("%+v", benchmark.GlobalCounters.All()), nil
}

func (s *server) GetProfile(ctx context.Context, name string, debug int32) ([]byte, error) {
	ctx = log.Enter(ctx, "GetProfile")
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
