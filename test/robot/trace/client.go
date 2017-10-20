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

package trace

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/job/worker"
	"github.com/google/gapid/test/robot/stash"

	_ "github.com/google/gapid/gapidapk"
)

type client struct {
	store   *stash.Client
	manager Manager
	tempDir file.Path

	registry *bind.Registry
	runners  map[string]*runner
	l        sync.Mutex
}

type retryError struct {
}

func (r retryError) Error() string {
	return "try again"
}

type runner struct {
	*client
	device bind.Device
}

func (c *client) getRunner(d bind.Device) (r *runner, existing bool) {
	c.l.Lock()
	defer c.l.Unlock()
	serial := d.Instance().GetSerial()

	r, existing = c.runners[serial]
	if !existing {
		r = &runner{client: c, device: d}
		c.runners[serial] = r
	}
	return r, existing
}

// TODO: not assume all android devices are valid trace targets
func (c *client) OnDeviceAdded(ctx context.Context, d bind.Device) {
	host := host.Instance(ctx)
	inst := d.Instance()

	r, existing := c.getRunner(d)
	if !existing {
		log.I(ctx, "Device added: %s", inst.GetName())
		crash.Go(func() {
			if err := c.manager.Register(ctx, host, inst, r.trace); err != nil {
				log.E(ctx, "Error running trace client: %v", err)
			}
		})
	} else {
		log.I(ctx, "Device restored: %s", inst.GetName())
		r.device = d
	}
}

func (c *client) OnDeviceRemoved(ctx context.Context, d bind.Device) {
	log.I(ctx, "Device removed: %s", d.Instance().GetName())
	// TODO: find a more graceful way to handle this.
	// r, _ := c.getRunner(d)
	// r.device = offlineDevice{d}
}

// Run starts new trace client if any hardware is available.
func Run(ctx context.Context, store *stash.Client, manager Manager, tempDir file.Path) error {
	c := &client{
		store:    store,
		manager:  manager,
		tempDir:  tempDir,
		registry: bind.NewRegistry(),
		runners:  make(map[string]*runner),
	}
	c.registry.Listen(c)

	crash.Go(func() {
		ctx = bind.PutRegistry(ctx, c.registry)
		if err := adb.Monitor(ctx, c.registry, 15*time.Second); err != nil {
			log.E(ctx, "adb.Monitor failed: %v", err)
		}
	})

	return nil
}

func (r *runner) trace(ctx context.Context, t *Task) error {
	if r.device.Status() != bind.Status_Online {
		log.I(ctx, "Trying to trace %s on %s not started, device status %s", t.Input.Subject, r.device.Instance().GetSerial(), r.device.Status().String())
		return nil
	}

	if err := r.manager.Update(ctx, t.Action, job.Running, nil); err != nil {
		return err
	}
	var output *Output
	err := worker.RetryFunction(ctx, 4, time.Millisecond*100, func() (err error) {
		output, err = doTrace(ctx, t.Action, t.Input, r.store, r.device, r.tempDir)
		return
	})
	status := job.Succeeded
	if err != nil {
		status = job.Failed
		log.E(ctx, "Error running trace: %v", err)
	} else if output.CallError != "" {
		status = job.Failed
		log.E(ctx, "Error during trace: %v", output.CallError)
	}

	return r.manager.Update(ctx, t.Action, status, output)
}

// doTrace extracts input files and runs `gapit trace` on them, capturing the output. The output object will
// be partially filled in the event of an upload error from store in order to allow examination of the logs.
func doTrace(ctx context.Context, action string, in *Input, store *stash.Client, d bind.Device, tempDir file.Path) (*Output, error) {
	subject := tempDir.Join(action + ".apk")
	tracefile := tempDir.Join(action + ".gfxtrace")
	extractedDir := tempDir.Join(action + "_tools")
	extractedLayout := layout.BinLayout(extractedDir)

	gapidAPK, err := extractedLayout.GapidApk(ctx, in.GetLayout().GetGapidAbi())
	if err != nil {
		return nil, err
	}

	gapit, err := extractedLayout.Gapit(ctx)
	if err != nil {
		return nil, err
	}

	traceTime, err := ptypes.Duration(in.GetHints().GetTraceTime())
	if err != nil {
		traceTime = time.Minute // TODO: support Robot-wide override
	}

	defer func() {
		file.Remove(subject)
		file.Remove(tracefile)
		file.RemoveAll(extractedDir)
	}()
	if err := store.GetFile(ctx, in.Subject, subject); err != nil {
		return nil, err
	}
	if err := store.GetFile(ctx, in.Gapit, gapit); err != nil {
		return nil, err
	}
	if err := store.GetFile(ctx, in.GapidApk, gapidAPK); err != nil {
		return nil, err
	}
	params := []string{
		"trace",
		"-out", tracefile.System(),
		"-apk", subject.System(),
		"-for", traceTime.String(),
		"-disable-pcs",
		"-observe-frames", "5",
		"-record-errors",
		"-gapii-device", d.Instance().Serial,
		"-api", in.GetHints().GetAPI(),
	}
	cmd := shell.Command(gapit.System(), params...)
	output, callErr := cmd.Call(ctx)
	if err := worker.NeedsRetry(output, "Failed to connect to the GAPIS server"); err != nil {
		return nil, err
	}

	outputObj := &Output{}
	if callErr != nil {
		if err := worker.NeedsRetry(callErr.Error()); err != nil {
			return nil, err
		}
		outputObj.CallError = callErr.Error()
	}
	output = fmt.Sprintf("%s\n\n%s", cmd, output)
	log.I(ctx, output)
	logID, err := store.UploadString(ctx, stash.Upload{Name: []string{"trace.log"}, Type: []string{"text/plain"}}, output)
	if err != nil {
		return outputObj, err
	}
	outputObj.Log = logID
	traceID, err := store.UploadFile(ctx, tracefile)
	if err != nil {
		return outputObj, err
	}
	outputObj.Trace = traceID
	return outputObj, nil
}

type offlineDevice struct {
	bind.Device
}

func (offlineDevice) Status() bind.Status {
	return bind.Status_Offline
}
