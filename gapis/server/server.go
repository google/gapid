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
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sync/atomic"
	"time"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/crash/reporting"
	"github.com/google/gapid/core/context/keys"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/analytics"
	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/devices"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
	"github.com/google/gapid/gapis/trace"
	"github.com/google/gapid/gapis/trace/tracer"

	"github.com/google/go-github/github"

	// Register all the apis
	_ "github.com/google/gapid/gapis/api/all"
)

// Config holds the server configuration settings.
type Config struct {
	Info             *service.ServerInfo
	StringTables     []*stringtable.StringTable
	EnableLocalFiles bool
	AuthToken        auth.Token
	DeviceScanDone   task.Signal
	LogBroadcaster   *log.Broadcaster
	IdleTimeout      time.Duration
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
		cfg.EnableLocalFiles,
		cfg.DeviceScanDone,
		cfg.LogBroadcaster,
	}
}

type server struct {
	info             *service.ServerInfo
	stbs             []*stringtable.StringTable
	enableLocalFiles bool
	deviceScanDone   task.Signal
	logBroadcaster   *log.Broadcaster
}

func (s *server) Ping(ctx context.Context) error {
	return nil
}

func (s *server) GetServerInfo(ctx context.Context) (*service.ServerInfo, error) {
	ctx = status.Start(ctx, "RPC GetServerInfo")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "GetServerInfo")
	return s.info, nil
}

func (s *server) CheckForUpdates(ctx context.Context, includePrereleases bool) (*service.Release, error) {
	const (
		githubOrg  = "google"
		githubRepo = "gapid"
	)
	ctx = status.Start(ctx, "RPC CheckForUpdates")
	defer status.Finish(ctx)
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
	ctx = status.Start(ctx, "RPC GetAvailableStringTables")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "GetAvailableStringTables")
	infos := make([]*stringtable.Info, len(s.stbs))
	for i, table := range s.stbs {
		infos[i] = table.Info
	}
	return infos, nil
}

func (s *server) GetStringTable(ctx context.Context, info *stringtable.Info) (*stringtable.StringTable, error) {
	ctx = status.Start(ctx, "RPC GetStringTable")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "GetStringTable")
	for _, table := range s.stbs {
		if table.Info.CultureCode == info.CultureCode {
			return table, nil
		}
	}
	return nil, fmt.Errorf("String table not found")
}

func (s *server) ImportCapture(ctx context.Context, name string, data []uint8) (*path.Capture, error) {
	ctx = status.Start(ctx, "RPC ImportCapture")
	defer status.Finish(ctx)
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
	ctx = status.Start(ctx, "RPC ExportCapture")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "ExportCapture")
	b := bytes.Buffer{}
	if err := capture.Export(ctx, c, &b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// ReadFile exists because ioutil.ReadFile is broken on Windows.
// https://github.com/golang/go/issues/26923
func ReadFile(f *os.File) ([]byte, error) {
	in := []byte{}
	buff := [1024 * 1024 * 1024]byte{}
	for {
		n, err := f.Read(buff[:])
		if err != nil && err != io.EOF {
			return nil, err
		}
		in = append(in, buff[:n]...)
		if err == io.EOF {
			break
		}
	}
	return in, nil
}

func (s *server) LoadCapture(ctx context.Context, path string) (*path.Capture, error) {
	ctx = status.Start(ctx, "RPC LoadCapture")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "LoadCapture")
	if !s.enableLocalFiles {
		return nil, fmt.Errorf("Server not configured to allow reading of local files")
	}
	name := filepath.Base(path)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	in, err := ReadFile(f)
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
	// Pre-resolve the dependency graph.
	crash.Go(func() {
		newCtx := keys.Clone(context.Background(), ctx)
		_, err = dependencygraph.GetFootprint(newCtx, p)
		if err != nil {
			log.E(newCtx, "Error resolve dependency graph: %v", err)
		}
	})
	return p, nil
}

func (s *server) SaveCapture(ctx context.Context, c *path.Capture, path string) error {
	ctx = status.Start(ctx, "RPC SaveCapture")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "SaveCapture")
	if !s.enableLocalFiles {
		return fmt.Errorf("Server not configured to allow writing of local files")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	return capture.Export(ctx, c, f)
}

func (s *server) ExportReplay(ctx context.Context, c *path.Capture, d *path.Device, path string) error {
	ctx = log.Enter(ctx, "ExportReplay")
	if !s.enableLocalFiles {
		return fmt.Errorf("Server not configured to allow writing of local files")
	}
	return replay.ExportReplay(ctx, c, d, path)
}

func (s *server) GetDevices(ctx context.Context) ([]*path.Device, error) {
	ctx = status.Start(ctx, "RPC GetDevices")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "GetDevices")
	s.deviceScanDone.Wait(ctx)
	devices := bind.GetRegistry(ctx).Devices()
	paths := make([]*path.Device, len(devices))
	for i, d := range devices {
		paths[i] = path.NewDevice(d.Instance().ID.ID())
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
	ctx = status.Start(ctx, "RPC GetDevicesForReplay")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "GetDevicesForReplay")
	s.deviceScanDone.Wait(ctx)
	return devices.ForReplay(ctx, p)
}

func (s *server) GetFramebufferAttachment(
	ctx context.Context,
	replaySettings *service.ReplaySettings,
	after *path.Command,
	attachment api.FramebufferAttachment,
	settings *service.RenderSettings,
	hints *service.UsageHints,
) (*path.ImageInfo, error) {

	ctx = status.Start(ctx, "RPC GetFramebufferAttachment")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "GetFramebufferAttachment")
	if err := replaySettings.Device.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", replaySettings.Device)
	}
	if err := after.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", after)
	}
	r := &path.ResolveConfig{
		ReplayDevice: replaySettings.Device,
	}
	return resolve.FramebufferAttachment(ctx, replaySettings, after, attachment, settings, hints, r)
}

func (s *server) Get(ctx context.Context, p *path.Any, c *path.ResolveConfig) (interface{}, error) {
	ctx = status.Start(ctx, "RPC Get<%v>", p)
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "Get")
	if err := p.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", p)
	}
	v, err := resolve.Get(ctx, p, c)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (s *server) Set(ctx context.Context, p *path.Any, v interface{}, r *path.ResolveConfig) (*path.Any, error) {
	ctx = status.Start(ctx, "RPC Set<%v>", p)
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "Set")
	if err := p.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", p)
	}
	return resolve.Set(ctx, p, v, r)
}

func (s *server) Follow(ctx context.Context, p *path.Any, r *path.ResolveConfig) (*path.Any, error) {
	ctx = status.Start(ctx, "RPC Follow")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "Follow")
	if err := p.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", p)
	}
	return resolve.Follow(ctx, p, r)
}

func (s *server) GetLogStream(ctx context.Context, handler log.Handler) error {
	ctx = status.Start(ctx, "RPC GetLogStream")
	defer status.Finish(ctx)
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
	ctx = status.Start(ctx, "RPC Find")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "Find")
	return resolve.Find(ctx, req, handler)
}

func (s *server) Profile(ctx context.Context, pprofW, traceW io.Writer, memorySnapshotInterval uint32) (stop func() error, err error) {
	ctx = status.Start(ctx, "RPC Profile")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "Profile")

	stop = task.Async(ctx, func(ctx context.Context) error {
		// flush is a list of functions that need to be called once the function
		// returns.
		flush := []func(){}
		defer func() {
			for _, f := range flush {
				f()
			}
		}()

		if pprofW != nil {
			// Pprof CPU profiling requested.
			if err := pprof.StartCPUProfile(pprofW); err != nil {
				return err
			}
			flush = append(flush, pprof.StopCPUProfile)
		}

		if traceW != nil {
			// Chrome trace data requested.
			stop := status.RegisterTracer(traceW)
			flush = append(flush, stop)
		}

		// Poll the memory. This will block until the context is cancelled.
		msi := time.Second * time.Duration(memorySnapshotInterval)
		return task.Poll(ctx, msi, func(ctx context.Context) error {
			status.SnapshotMemory(ctx)
			return nil
		})
	})

	return stop, nil
}

func (s *server) GetPerformanceCounters(ctx context.Context) (string, error) {
	ctx = status.Start(ctx, "RPC GetPerformanceCounters")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "GetPerformanceCounters")
	return fmt.Sprintf("%+v", benchmark.GlobalCounters.All()), nil
}

func (s *server) GetProfile(ctx context.Context, name string, debug int32) ([]byte, error) {
	ctx = status.Start(ctx, "RPC GetProfile")
	defer status.Finish(ctx)
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

func (s *server) EnableCrashReporting(ctx context.Context, enable bool) error {
	if enable {
		reporting.Enable(ctx, app.Name, app.Version.String())
	} else {
		reporting.Disable()
	}
	return nil
}

func (s *server) EnableAnalytics(ctx context.Context, enable bool, clientID string) error {
	if enable {
		analytics.Enable(ctx, clientID, analytics.AppVersion{
			Name: app.Name, Build: app.Version.Build,
			Major: app.Version.Major, Minor: app.Version.Minor, Point: app.Version.Point,
		})
	} else {
		analytics.Disable()
	}
	return nil
}

func (s *server) ClientEvent(ctx context.Context, req *service.ClientEventRequest) error {
	if i := req.GetInteraction(); i != nil {
		analytics.SendEvent("client", i.View, i.Action.String())
	}
	return nil
}

func (s *server) TraceTargetTreeNode(ctx context.Context, req *service.TraceTargetTreeNodeRequest) (*service.TraceTargetTreeNode, error) {
	tttn, err := trace.TraceTargetTreeNode(ctx, *req.Device, req.Uri, req.Density)
	if err != nil {
		return nil, err
	}

	return &service.TraceTargetTreeNode{
		Name:                tttn.Name,
		Icon:                tttn.Icon,
		Uri:                 tttn.URI,
		ParentUri:           tttn.Parent,
		ChildrenUris:        tttn.Children,
		TraceUri:            tttn.TraceURI,
		FriendlyApplication: tttn.ApplicationName,
		FriendlyExecutable:  tttn.ExecutableName,
	}, nil
}

func (s *server) FindTraceTargets(ctx context.Context, req *service.FindTraceTargetsRequest) ([]*service.TraceTargetTreeNode, error) {
	nodes, err := trace.FindTraceTargets(ctx, *req.Device, req.Uri)
	if err != nil {
		return nil, err
	}
	out := make([]*service.TraceTargetTreeNode, len(nodes))
	for i, n := range nodes {
		out[i] = &service.TraceTargetTreeNode{
			Name:                n.Name,
			Icon:                n.Icon,
			Uri:                 n.URI,
			ParentUri:           n.Parent,
			ChildrenUris:        n.Children,
			TraceUri:            n.TraceURI,
			FriendlyApplication: n.ApplicationName,
			FriendlyExecutable:  n.ExecutableName,
		}
	}
	return out, nil
}

func optionsToTraceOptions(opts *service.TraceOptions) tracer.TraceOptions {
	return tracer.TraceOptions{
		URI:                   opts.GetUri(),
		UploadApplication:     opts.GetUploadApplication(),
		Port:                  opts.GetPort(),
		ClearCache:            opts.ClearCache,
		APIs:                  opts.Apis,
		WriteFile:             opts.ServerLocalSavePath,
		AdditionalFlags:       opts.AdditionalCommandLineArgs,
		CWD:                   opts.Cwd,
		Environment:           opts.Environment,
		Duration:              opts.Duration,
		ObserveFrameFrequency: opts.ObserveFrameFrequency,
		ObserveDrawFrequency:  opts.ObserveDrawFrequency,
		StartFrame:            opts.StartFrame,
		FramesToCapture:       opts.FramesToCapture,
		DisablePCS:            opts.DisablePcs,
		RecordErrorState:      opts.RecordErrorState,
		DeferStart:            opts.DeferStart,
		NoBuffer:              opts.NoBuffer,
		HideUnknownExtensions: opts.HideUnknownExtensions,
		StoreTimestamps:       opts.RecordTraceTimes,
	}
}

// traceHandler implements the TraceHandler interface
// It wraps all of the state for a trace operation
type traceHandler struct {
	ctx            context.Context
	initialized    bool               // Has the trace been initialized yet
	started        bool               // Has the trace been started yet
	done           bool               // Has the trace been finished
	err            error              // Was there an error to report at next request
	bytesWritten   int64              // How many bytes have been written so far
	startSignal    task.Signal        // If we are in MEC this signal will start the trace
	startFunc      task.Task          // This is the function to go with the above signal.
	stopFunc       context.CancelFunc // stopFunc can be used to stop/clean up the trace
	doneSignal     task.Signal        // doneSignal can be waited on to make sure the trace is actually done
	doneSignalFunc task.Task          // doneSignalFunc is called when tracing finished normally
}

func (r *traceHandler) Initialize(opts *service.TraceOptions) (*service.StatusResponse, error) {
	if r.initialized {
		return nil, log.Errf(r.ctx, nil, "Error initialize a running trace")
	}
	r.initialized = true
	tracerOptions := optionsToTraceOptions(opts)
	go func() {
		r.err = trace.Trace(r.ctx, opts.Device, r.startSignal, &tracerOptions, &r.bytesWritten)
		r.done = true
		r.doneSignalFunc(r.ctx)
	}()

	stat := service.TraceStatus_Initializing
	if !opts.DeferStart {
		r.started = true
	}

	resp := &service.StatusResponse{
		BytesCaptured: 0,
		Status:        stat,
	}

	return resp, nil
}

func (r *traceHandler) Event(req service.TraceEvent) (*service.StatusResponse, error) {
	if !r.initialized {
		return nil, log.Errf(r.ctx, nil, "Cannot get/change status of an uninitialized trace")
	}
	if r.err != nil {
		return nil, log.Errf(r.ctx, r.err, "Tracing Failed")
	}

	switch req {
	case service.TraceEvent_Begin:
		if r.started {
			return nil, log.Errf(r.ctx, nil, "Invalid to start an already running trace")
		}
		r.startFunc(r.ctx)
		r.started = true
	case service.TraceEvent_Stop:
		if !r.started {
			return nil, log.Errf(r.ctx, nil, "Cannot end a trace that was not started")
		}
		r.stopFunc()
		r.doneSignal.Wait(r.ctx)
	case service.TraceEvent_Status:
		// intentionally empty
	}

	status := service.TraceStatus_Uninitialized
	bytes := atomic.LoadInt64(&r.bytesWritten)

	if r.initialized {
		if bytes == 0 {
			status = service.TraceStatus_Initializing
		} else {
			status = service.TraceStatus_WaitingToStart
		}
	}

	if r.started {
		if bytes == 0 {
			status = service.TraceStatus_Initializing
		} else {
			status = service.TraceStatus_Capturing
		}
	}
	if r.done {
		status = service.TraceStatus_Done
	}
	resp := &service.StatusResponse{
		BytesCaptured: atomic.LoadInt64(&r.bytesWritten),
		Status:        status,
	}
	return resp, nil
}

func (r *traceHandler) Dispose() {
	r.stopFunc()
	return

}

func (s *server) Trace(ctx context.Context) (service.TraceHandler, error) {
	startSignal, startFunc := task.NewSignal()
	ctx, stop := context.WithCancel(ctx)
	doneSignal, doneSigFunc := task.NewSignal()
	return &traceHandler{
		ctx,
		false,
		false,
		false,
		nil,
		0,
		startSignal,
		startFunc,
		stop,
		doneSignal,
		doneSigFunc,
	}, nil
}
