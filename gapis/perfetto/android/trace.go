// Copyright (C) 2019 Google Inc.
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

package android

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	perfetto_pb "protos/perfetto/config"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/text"
	"github.com/google/gapid/gapidapk"
	"github.com/google/gapid/gapis/perfetto"
	"github.com/google/gapid/gapis/service"
)

const (
	ftraceDataSourceName          = "linux.ftrace"
	gpuRenderStagesDataSourceName = "gpu.renderstages"

	// perfettoTraceFile is the location on the device where we'll ask Perfetto
	// to store the trace data while tracing.
	perfettoTraceFile          = "/data/misc/perfetto-traces/gapis-trace"
	renderStageVulkanLayerName = "VkRenderStagesProducer"
	readInterval               = 100 * time.Millisecond
)

// Process represents a running Perfetto capture.
type Process struct {
	device         adb.Device
	config         *perfetto_pb.TraceConfig
	deferred       bool
	perfettoClient *perfetto.Client
}

func hasDataSourceEnabled(perfettoConfig *perfetto_pb.TraceConfig, ds string) bool {
	for _, dataSource := range perfettoConfig.GetDataSources() {
		if dataSource.Config.GetName() == ds {
			return true
		}
	}
	return false
}

// SetupProfileLayersSource configures the device to allow packages to be used as layer sources for profiling
func SetupProfileLayersSource(ctx context.Context, d adb.Device, apk *android.InstalledPackage, abi *device.ABI) (app.Cleanup, error) {
	supported, packageName, cleanup, err := d.PrepareGpuProfiling(ctx, apk)
	if err != nil || !supported {
		return cleanup.Invoke(ctx), err
	}

	packages := []string{}
	if packageName != "" {
		packages = append(packages, packageName)
	}
	if abi != nil {
		packages = append(packages, gapidapk.PackageName(abi))
	}

	nextCleanup, err := android.SetupLayers(ctx, d, apk.Name, packages, []string{})
	cleanup = cleanup.Then(nextCleanup)
	if err != nil {
		return cleanup.Invoke(ctx), log.Err(ctx, err, "Failed to setup GPU activity producer environment.")
	}
	return cleanup, nil
}

// setupProfileLayers configures the device to allow the app being traced to load the layers required for render stage profiling
func setupProfileLayers(ctx context.Context, d adb.Device, packageName string, hasRenderStages bool, abi *device.ABI, profileLayerPackages []string, layers []string) (app.Cleanup, error) {
	packages := []string{}
	enabledLayers := []string{}

	if abi != nil {
		packages = append(packages, gapidapk.PackageName(abi))
		enabledLayers = append(enabledLayers, layers...)
	}

	// Setup render stage layer. Render stage layer should be at the bottom (end of list).
	if hasRenderStages && d.Instance().GetConfiguration().GetPerfettoCapability().GetGpuProfiling().GetHasRenderStageProducerLayer() {
		packages = append(packages, profileLayerPackages...)
		enabledLayers = append(enabledLayers, renderStageVulkanLayerName)
	}

	cleanup, err := android.SetupLayers(ctx, d, packageName, packages, enabledLayers)
	if err != nil {
		return cleanup.Invoke(ctx), log.Err(ctx, err, "Failed to setup GPU activity producer environment.")
	}
	return cleanup, nil
}

// Start optional starts an app and sets up a Perfetto trace
func Start(ctx context.Context, d adb.Device, a *android.ActivityAction, opts *service.TraceOptions, abi *device.ABI, layers []string) (*Process, app.Cleanup, error) {
	ctx = log.Enter(ctx, "start")

	if abi != nil {
		_, err := gapidapk.EnsureInstalled(ctx, d, abi)
		if err != nil {
			return nil, nil, err
		}
	}

	var cleanup app.Cleanup

	if a != nil {
		ctx = log.V{
			"package":  a.Package.Name,
			"activity": a.Activity,
		}.Bind(ctx)

		packages := []string{}
		_, packageName, nextCleanup, err := d.PrepareGpuProfiling(ctx, a.Package)
		cleanup = cleanup.Then(nextCleanup)
		if err != nil {
			return nil, cleanup.Invoke(ctx), log.Err(ctx, err, "Failed to prepare GPU profiling")
		}
		if packageName != "" {
			packages = append(packages, packageName)
		}

		// Setup the profiling layers.
		hasRenderStages := hasDataSourceEnabled(opts.PerfettoConfig, gpuRenderStagesDataSourceName)
		nextCleanup, err = setupProfileLayers(ctx, d, a.Package.Name, hasRenderStages, abi, packages, layers)
		cleanup = cleanup.Then(nextCleanup)
		if err != nil {
			return nil, cleanup.Invoke(ctx), err
		}
	}

	log.I(ctx, "Unlocking device screen")
	unlocked, err := d.UnlockScreen(ctx)
	if err != nil {
		log.W(ctx, "Failed to determine lock state: %s", err)
	} else if !unlocked {
		return nil, cleanup.Invoke(ctx), log.Err(ctx, nil, "Please unlock your device screen: AGI can automatically unlock the screen only when no PIN/password/pattern is needed")
	}

	if a != nil {
		var additionalArgs []android.ActionExtra
		if opts.AdditionalCommandLineArgs != "" {
			additionalArgs = append(additionalArgs, android.CustomExtras(text.Quote(text.SplitArgs(opts.AdditionalCommandLineArgs))))
		}
		if err := d.StartActivity(ctx, *a, additionalArgs...); err != nil {
			return nil, cleanup.Invoke(ctx), log.Err(ctx, err, "Starting the activity")
		}
	}

	// Forcefully re-launch the Perfetto data producer in case it has gotten into
	// a bad state. This will kill any running producer and re-launch it.
	// Note that this could lead to undefined behavior due to driver behavior
	// changes or counters set differences (both are rare), if the driver was updated
	// since the producer was last started (e.g. at device discovery time). This should
	// be a very rare circumstance and can be fixed by restarting AGI.
	if err := gapidapk.EnsurePerfettoProducerLaunched(ctx, d); err != nil {
		return nil, cleanup.Invoke(ctx), err
	}

	// Use the direct client if we are not writting into a file on the device.
	var c *perfetto.Client
	if !opts.PerfettoConfig.GetWriteIntoFile() {
		c, err = d.ConnectPerfetto(ctx)
		if err != nil {
			log.W(ctx, "Failed to connect Perfetto through client API: %v, fall back to CLI.", err)
			c = nil
		}
	}

	return &Process{
		device:         d,
		config:         opts.PerfettoConfig,
		deferred:       opts.DeferStart,
		perfettoClient: c,
	}, cleanup, nil
}

// Capture starts the perfetto capture.
func (p *Process) Capture(ctx context.Context, start task.Signal, stop task.Signal, ready task.Task, w io.Writer, written *int64) (int64, error) {
	if p.perfettoClient != nil {
		return p.captureWithClientApi(ctx, start, stop, ready, w, written, p.device.Instance().GetConfiguration().GetPerfettoCapability().GetCanDownloadWhileTracing())
	}

	tmp, err := file.Temp()
	if err != nil {
		return 0, log.Err(ctx, err, "Failed to create a temp file")
	}

	// Signal that we are ready to start.
	atomic.StoreInt64(written, 1)

	if p.deferred && !start.Wait(ctx) {
		return 0, log.Err(ctx, nil, "Cancelled")
	}

	if err := p.device.StartPerfettoTrace(ctx, p.config, perfettoTraceFile, stop, ready); err != nil {
		return 0, err
	}

	if err := p.device.Pull(ctx, perfettoTraceFile, tmp.System()); err != nil {
		return 0, err
	}

	if err := p.device.RemoveFile(ctx, perfettoTraceFile); err != nil {
		log.E(ctx, "Failed to delete perfetto trace file %v", err)
	}

	size := tmp.Info().Size()
	atomic.StoreInt64(written, size)
	fh, err := os.Open(tmp.System())
	if err != nil {
		return 0, log.Err(ctx, err, fmt.Sprintf("Failed to open %s", tmp))
	}
	return io.Copy(w, fh)
}

type trackingWriter struct {
	out     io.Writer
	done    int64
	written *int64
}

// Write implements the io.Writer interface.
func (w *trackingWriter) Write(p []byte) (int, error) {
	n, err := w.out.Write(p)
	w.done += int64(n)
	atomic.StoreInt64(w.written, w.done)
	return n, err
}

func (p *Process) captureWithClientApi(ctx context.Context, start task.Signal, stop task.Signal, ready task.Task, w io.Writer, written *int64, downloadWhileTracing bool) (int64, error) {
	defer p.perfettoClient.Close(ctx)

	p.config.DeferredStart = proto.Bool(true)
	var buf bytes.Buffer
	var out io.Writer
	if downloadWhileTracing {
		out = &trackingWriter{w, 0, written}
	} else {
		out = &buf
	}
	ts, err := p.perfettoClient.Trace(ctx, p.config, out)
	if err != nil {
		return 0, log.Err(ctx, err, "Failed to setup Perfetto trace")
	}

	// Setting up ftrace trace points takes time, hence sometimes there's
	// a gap at the beginning of the trace. In order to avoid this issue, we
	// need to know when ftrace trace points are ready. This checks the node
	// of the ftrace event task_newtask that is always enabled regardless of
	// the user selection. Skip if ftrace is not enabled.
	if hasDataSourceEnabled(p.config, ftraceDataSourceName) {
		sessionReadySignal, sessionReadyFunc := task.NewSignal()
		timeout := false
		crash.Go(func() {
			cnt := 0
			fTraceFilePath := "/sys/kernel/tracing/events/task/task_newtask/enable"
			// debugfs was deprecated in Android 11, and hence AGI must use the path
			// to debugfs before Android 11.
			if p.device.Instance().GetConfiguration().GetOS().GetAPIVersion() < 30 {
				fTraceFilePath = "/sys/kernel/debug/tracing/events/task/task_newtask/enable"
			}
			for {
				res, _ := p.device.Shell("cat", fTraceFilePath).Call(ctx)
				if res == "1" {
					sessionReadyFunc(ctx)
					break
				}
				cnt += 1

				// Give up after 1000 tries
				if cnt == 1000 {
					timeout = true
					sessionReadyFunc(ctx)
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
		})

		// Signal that we are ready to start.
		if !sessionReadySignal.Wait(ctx) {
			return 0, log.Err(ctx, nil, "Cancelled")
		}
		if timeout {
			return 0, log.Err(ctx, nil, "Timed out in waiting for trace session ready")
		}
	}

	// TODO(b/147388497): Find a way to reliably know when Perfetto/producers are ready.
	delayedReady := task.Delay(ready, 250*time.Millisecond)

	atomic.StoreInt64(written, 1)
	if p.deferred && !start.Wait(ctx) {
		ts.Stop(ctx)
		return 0, log.Err(ctx, nil, "Cancelled")
	}
	delayedReady(ctx)
	ts.Start(ctx)

	wait := make(chan error, 1)
	crash.Go(func() {
		wait <- ts.Wait(ctx)
	})

	if downloadWhileTracing {
		crash.Go(func() {
			ticker := time.NewTicker(readInterval)
			for {
				<-ticker.C
				if err := ts.Read(ctx); err != nil {
					if err != perfetto.ErrDone {
						// This error will likely also be returned by ts.Wait, so just log it
						// here, in case there's also another error.
						log.W(ctx, "Perfetto client read error: %v", err)
					}
					ticker.Stop()
					break
				}
			}
		})
	}

	select {
	case err = <-wait:
	case <-stop:
		ts.Stop(ctx)
		err = <-wait
	}

	if err != nil {
		return 0, log.Err(ctx, err, "Failed during tracing session")
	}

	var numWritten int64
	if downloadWhileTracing {
		numWritten = atomic.LoadInt64(written)
	} else {
		numWritten, err = io.Copy(w, &buf)
		atomic.StoreInt64(written, numWritten)
	}
	return numWritten, err
}
