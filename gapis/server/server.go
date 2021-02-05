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
	"runtime"
	"runtime/pprof"
	"sync"
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
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/messages"
	perfetto "github.com/google/gapid/gapis/perfetto/service"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/devices"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/resolve/dependencygraph2"
	"github.com/google/gapid/gapis/resolve/dependencygraph2/graph_visualization"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
	"github.com/google/gapid/gapis/trace"

	// Register all the apis
	_ "github.com/google/gapid/gapis/api/all"
)

const (
	FILE_SIZE_LIMIT_IN_BYTES = 2147483647
)

// Config holds the server configuration settings.
type Config struct {
	Info             *service.ServerInfo
	StringTables     []*stringtable.StringTable
	EnableLocalFiles bool
	PreloadDepGraph  bool
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
		cfg.PreloadDepGraph,
		cfg.DeviceScanDone,
		cfg.LogBroadcaster,
	}
}

type server struct {
	info             *service.ServerInfo
	stbs             []*stringtable.StringTable
	enableLocalFiles bool
	preloadDepGraph  bool
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

func (s *server) CheckForUpdates(ctx context.Context, includeDevReleases bool) (*service.Releases, error) {
	ctx = status.Start(ctx, "RPC CheckForUpdates")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "CheckForUpdates")
	return checkForUpdates(ctx, includeDevReleases)
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
	src := &capture.Blob{Data: data}
	p, err := capture.Import(ctx, name, name, src)
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

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	name := fileInfo.Name()
	// Create a key to prevent name collusion between traces.
	key := fmt.Sprintf("%v%v%v", name, fileInfo.Size(), fileInfo.ModTime().Unix())
	src := &capture.File{Path: path}

	p, err := capture.Import(ctx, key, name, src)
	if err != nil {
		return nil, err
	}
	// Ensure the capture can be read by resolving it now.
	c, err := capture.ResolveFromPath(ctx, p)
	if err != nil {
		return nil, err
	}

	// For graphics capture, pre-resolve the dependency graph.
	if c.Service(ctx, p).Type == service.TraceType_Graphics && !config.DisableDeadCodeElimination && s.preloadDepGraph {
		newCtx := keys.Clone(context.Background(), ctx)
		crash.Go(func() {
			cctx := status.PutTask(newCtx, nil)
			cctx = status.StartBackground(cctx, "Precaching Dependency Graph")
			defer status.Finish(cctx)
			var err error
			cfg := dependencygraph2.DependencyGraphConfig{
				MergeSubCmdNodes:       true,
				IncludeInitialCommands: false,
			}
			_, err = dependencygraph2.GetDependencyGraph(cctx, p, cfg)
			if err != nil {
				log.E(newCtx, "Error resolve dependency graph: %v", err)
			}
		})
	}
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
func (s *server) ExportReplay(ctx context.Context, c *path.Capture, d *path.Device, out string, opts *service.ExportReplayOptions) error {
	ctx = status.Start(ctx, "RPC ExportReplay")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "ExportReplay")
	if !s.enableLocalFiles {
		return fmt.Errorf("Server not configured to allow writing of local files")
	}
	return exportReplay(ctx, c, d, out, opts)
}

func (s *server) DCECapture(ctx context.Context, p *path.Capture, requested []*path.Command) (*path.Capture, error) {
	ctx = log.Enter(ctx, "DCECapture")
	c, err := capture.ResolveFromPath(ctx, p)
	if err != nil {
		return nil, err
	}
	trimmed, err := dependencygraph2.DCECapture(ctx, c.Name()+"_dce", p, requested)
	if err != nil {
		return nil, err
	}
	return trimmed, nil
}

func (s *server) SplitCapture(ctx context.Context, rng *path.Commands) (*path.Capture, error) {
	ctx = log.Enter(ctx, "SplitCapture")
	c, err := capture.ResolveGraphicsFromPath(ctx, rng.Capture)
	if err != nil {
		return nil, err
	}

	var from, to int
	if len(rng.From) > 0 {
		from = int(rng.From[0])
	}
	if len(rng.To) > 0 {
		to = int(rng.To[0])
	}

	if from >= len(c.Commands) || to > len(c.Commands) || (to != 0 && to <= from) {
		return nil, fmt.Errorf("Invalid range: [%d, %d) not in [%d, %d)", from, to, 0, len(c.Commands))
	} else if to == 0 {
		to = len(c.Commands)
	}

	name := fmt.Sprintf("%s [%d,%d)", c.Name(), from, to)
	if gc, err := capture.NewGraphicsCapture(ctx, name, c.Header, c.InitialState, c.Commands[from:to]); err != nil {
		return nil, err
	} else {
		return capture.New(ctx, gc)
	}
}

func (s *server) GetGraphVisualization(ctx context.Context, p *path.Capture, format service.GraphFormat) ([]byte, error) {
	ctx = status.Start(ctx, "RPC GetGraphVisualization")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "GetGraphVisualization")

	graphVisualization, err := graph_visualization.GetGraphVisualizationFromCapture(ctx, p, format)
	if err != nil {
		return []byte{}, err
	}
	if len(graphVisualization) > FILE_SIZE_LIMIT_IN_BYTES {
		return []byte{}, log.Errf(ctx, err, "The file size for graph visualization exceeds %d bytes", FILE_SIZE_LIMIT_IN_BYTES)
	}
	return graphVisualization, nil
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

func (s *server) GetDevicesForReplay(ctx context.Context, p *path.Capture) ([]*path.Device, []bool, []*stringtable.Msg, error) {
	ctx = status.Start(ctx, "RPC GetDevicesForReplay")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "GetDevicesForReplay")
	s.deviceScanDone.Wait(ctx)
	return devices.ForReplay(ctx, p)
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

func (s *server) Delete(ctx context.Context, p *path.Any, r *path.ResolveConfig) (*path.Any, error) {
	ctx = status.Start(ctx, "RPC Delete<%v>", p)
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "Delete")
	if err := p.Validate(); err != nil {
		return nil, log.Errf(ctx, err, "Invalid path: %v", p)
	}
	return resolve.Delete(ctx, p, r)
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
	ctx = status.StartBackground(ctx, "RPC GetLogStream")
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

type statusListener struct {
	f                  func(*service.TaskUpdate)
	m                  func(*service.MemoryStatus)
	r                  func(*service.ReplayUpdate)
	lastProgressUpdate map[*status.Task]time.Time // guarded by progressMutex
	progressUpdateFreq time.Duration              // guarded by progressMutex
	progressMutex      sync.Mutex
}

func (l *statusListener) OnTaskStart(ctx context.Context, task *status.Task) {
	l.progressMutex.Lock()
	defer l.progressMutex.Unlock()

	l.f(&service.TaskUpdate{
		Status:          service.TaskStatus_STARTING,
		Id:              task.ID(),
		Parent:          task.ParentID(),
		Name:            task.Name(),
		CompletePercent: 0,
		Background:      task.Background(),
	})
	l.lastProgressUpdate[task] = time.Now()
}

func (l *statusListener) OnTaskProgress(ctx context.Context, task *status.Task) {
	l.progressMutex.Lock()
	defer l.progressMutex.Unlock()
	if t, ok := l.lastProgressUpdate[task]; ok {
		if time.Since(t) > l.progressUpdateFreq || int32(task.Completion()) == 100 {
			l.f(&service.TaskUpdate{
				Status:          service.TaskStatus_PROGRESS,
				Id:              task.ID(),
				Parent:          task.ParentID(),
				Name:            task.Name(),
				CompletePercent: int32(task.Completion()),
				Background:      task.Background(),
			})
			l.lastProgressUpdate[task] = time.Now()
		}
	}
}

func (l *statusListener) OnTaskBlock(ctx context.Context, task *status.Task) {
	l.progressMutex.Lock()
	defer l.progressMutex.Unlock()

	l.f(&service.TaskUpdate{
		Status:          service.TaskStatus_BLOCKED,
		Id:              task.ID(),
		Parent:          task.ParentID(),
		Name:            task.Name(),
		CompletePercent: int32(task.Completion()),
		Background:      task.Background(),
	})
}

func (l *statusListener) OnTaskUnblock(ctx context.Context, task *status.Task) {
	l.progressMutex.Lock()
	defer l.progressMutex.Unlock()

	l.f(&service.TaskUpdate{
		Status:          service.TaskStatus_UNBLOCKED,
		Id:              task.ID(),
		Parent:          task.ParentID(),
		Name:            task.Name(),
		CompletePercent: int32(task.Completion()),
		Background:      task.Background(),
	})
}

func (l *statusListener) OnTaskFinish(ctx context.Context, task *status.Task) {
	l.progressMutex.Lock()
	defer l.progressMutex.Unlock()

	l.f(&service.TaskUpdate{
		Status:          service.TaskStatus_FINISHED,
		Id:              task.ID(),
		Parent:          task.ParentID(),
		Name:            task.Name(),
		CompletePercent: 100,
		Background:      task.Background(),
	})
	delete(l.lastProgressUpdate, task)

}

func (l *statusListener) OnEvent(ctx context.Context, task *status.Task, event string, scope status.EventScope) {
	l.progressMutex.Lock()
	defer l.progressMutex.Unlock()

	if task != nil {
		l.f(&service.TaskUpdate{
			Status:          service.TaskStatus_EVENT,
			Id:              task.ID(),
			Parent:          task.ParentID(),
			Name:            task.Name(),
			CompletePercent: int32(task.Completion()),
			Background:      task.Background(),
			Event:           event,
		})
	} else {
		l.f(&service.TaskUpdate{
			Status:          service.TaskStatus_EVENT,
			Id:              0,
			Parent:          0,
			Name:            "Global",
			CompletePercent: 0,
			Background:      false,
			Event:           event,
		})
	}
}

func (l *statusListener) OnMemorySnapshot(ctx context.Context, stats runtime.MemStats) {
	l.progressMutex.Lock()
	defer l.progressMutex.Unlock()

	l.m(&service.MemoryStatus{
		TotalHeap: stats.Alloc,
	})
}

func (l *statusListener) OnReplayStatusUpdate(ctx context.Context, r *status.Replay, label uint64, totalInstrs, finishedInstrs uint32) {
	// Not using lock here for efficiency. Because replay status update is of high frequency.
	// l.progressMutex.Lock()
	// defer l.progressMutex.Unlock()

	device := path.NewDevice(r.Device)
	started, finished := r.Started(), r.Finished()

	var status service.ReplayStatus
	switch {
	case finished:
		status = service.ReplayStatus_REPLAY_FINISHED
	case totalInstrs > 0:
		status = service.ReplayStatus_REPLAY_EXECUTING
	case started:
		status = service.ReplayStatus_REPLAY_STARTED
	default:
		status = service.ReplayStatus_REPLAY_QUEUED
	}

	l.r(&service.ReplayUpdate{
		ReplayId:       r.ID,
		Device:         device,
		Status:         status,
		Label:          label,
		TotalInstrs:    totalInstrs,
		FinishedInstrs: finishedInstrs,
	})
}

func (s *server) Status(ctx context.Context, snapshotInterval, statusInterval time.Duration, f func(*service.TaskUpdate), m func(*service.MemoryStatus), r func(*service.ReplayUpdate)) error {
	ctx = status.StartBackground(ctx, "RPC Status")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "Status")
	l := &statusListener{f, m, r, make(map[*status.Task]time.Time), statusInterval, sync.Mutex{}}
	unregister := status.RegisterListener(l)
	defer unregister()

	if snapshotInterval > 0 {
		// Poll the memory. This will block until the context is cancelled.
		stopSnapshot := task.Async(ctx, func(ctx context.Context) error {
			return task.Poll(ctx, snapshotInterval, func(ctx context.Context) error {
				status.SnapshotMemory(ctx)
				return nil
			})
		})
		defer stopSnapshot()
	}

	select {
	case <-task.ShouldStop(ctx):
	}
	return task.StopReason(ctx)
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

// traceHandler implements the TraceHandler interface
// It wraps all of the state for a trace operation
type traceHandler struct {
	initialized    bool        // Has the trace been initialized yet
	started        bool        // Has the trace been started yet
	done           bool        // Has the trace been finished
	err            error       // Was there an error to report at next request
	bytesWritten   int64       // How many bytes have been written so far
	startSignal    task.Signal // If we are in MEC this signal will start the trace
	startFunc      task.Task   // This is the function to go with the above signal.
	stopFunc       task.Task   // stopFunc can be used to stop/clean up the trace
	doneSignal     task.Signal // doneSignal can be waited on to make sure the trace is actually done
	doneSignalFunc task.Task   // doneSignalFunc is called when tracing finished normally
}

func (r *traceHandler) Initialize(ctx context.Context, opts *service.TraceOptions) (*service.StatusResponse, error) {
	if r.initialized {
		return nil, log.Errf(ctx, nil, "Error initialize a running trace")
	}
	r.initialized = true
	stopSignal, stopFunc := task.NewSignal()
	readyFunc := task.Noop()
	r.stopFunc = stopFunc
	go func() {
		r.err = trace.Trace(ctx, opts.Device, r.startSignal, stopSignal, readyFunc, opts, &r.bytesWritten)
		r.done = true
		r.doneSignalFunc(ctx)
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

func (r *traceHandler) Event(ctx context.Context, req service.TraceEvent) (*service.StatusResponse, error) {
	if !r.initialized {
		return nil, log.Errf(ctx, nil, "Cannot get/change status of an uninitialized trace")
	}
	if r.err != nil {
		return nil, log.Errf(ctx, r.err, "Tracing Failed")
	}

	switch req {
	case service.TraceEvent_Begin:
		if r.started {
			return nil, log.Errf(ctx, nil, "Invalid to start an already running trace")
		}
		r.startFunc(ctx)
		r.started = true
	case service.TraceEvent_Stop:
		if !r.started {
			return nil, log.Errf(ctx, nil, "Cannot end a trace that was not started")
		}
		r.stopFunc(ctx)
		r.doneSignal.Wait(ctx)
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
		// TODO: this is not a good way to signal that we are ready to capture.
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

func (r *traceHandler) Dispose(ctx context.Context) {
	if !r.done {
		r.stopFunc(ctx)
	}
	return

}

func (s *server) Trace(ctx context.Context) (service.TraceHandler, error) {
	startSignal, startFunc := task.NewSignal()
	doneSignal, doneSigFunc := task.NewSignal()
	return &traceHandler{
		startSignal:    startSignal,
		startFunc:      startFunc,
		doneSignal:     doneSignal,
		doneSignalFunc: doneSigFunc,
		stopFunc:       doneSigFunc,
	}, nil
}

func (s *server) UpdateSettings(ctx context.Context, settings *service.UpdateSettingsRequest) error {
	if settings.EnableAnalytics {
		analytics.Enable(ctx, settings.ClientId, analytics.AppVersion{
			Name: app.Name, Build: app.Version.Build,
			Major: app.Version.Major, Minor: app.Version.Minor, Point: app.Version.Point,
		})
	} else {
		analytics.Disable()
	}

	if settings.EnableCrashReporting {
		reporting.Enable(ctx, app.Name, app.Version.String())
	} else {
		reporting.Disable()
	}

	if settings.Adb != "" {
		adb.ADB = file.Abs(settings.Adb)
	}
	return nil
}

func (s *server) GetTimestamps(ctx context.Context, req *service.GetTimestampsRequest, h service.TimeStampsHandler) error {
	ctx = status.Start(ctx, "RPC GetTimestamps")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "GetTimestamps")
	return replay.GetTimestamps(ctx, req.Capture, req.Device, req.LoopCount, h)
}

func (s *server) GpuProfile(ctx context.Context, req *service.GpuProfileRequest) (*service.ProfilingData, error) {
	ctx = status.Start(ctx, "RPC GpuProfile")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "GpuProfile")
	res, err := replay.GpuProfile(ctx, req.Capture, req.Device, req.Experiments)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *server) PerfettoQuery(ctx context.Context, c *path.Capture, query string) (*perfetto.QueryResult, error) {
	ctx = status.Start(ctx, "RPC PerfettoQuery")
	defer status.Finish(ctx)

	ctx = log.Enter(ctx, "PerfettoQuery")
	p, err := capture.ResolvePerfettoFromPath(ctx, c)
	if err != nil {
		return nil, err
	}

	res, err := p.Processor.Query(query)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *server) ValidateDevice(ctx context.Context, d *path.Device) error {
	ctx = status.Start(ctx, "RPC ValidateDevice")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "ValidateDevice")
	return trace.Validate(ctx, d)
}

func (s *server) InstallApp(ctx context.Context, d *path.Device, app string) error {
	ctx = status.Start(ctx, "RPC Install App")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "InstallApp")

	if !s.enableLocalFiles {
		return fmt.Errorf("Server not configured to allow reading of local files")
	}

	device := bind.GetRegistry(ctx).Device(d.GetID().ID())
	if device == nil {
		return &service.ErrDataUnavailable{Reason: messages.ErrUnknownDevice()}
	}
	return device.InstallApp(ctx, app)
}
