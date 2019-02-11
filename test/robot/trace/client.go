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
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/job/worker"
	"github.com/google/gapid/test/robot/stash"

	_ "github.com/google/gapid/gapidapk"
)

const (
	traceTimeout = time.Hour
	// this string is returned when GAPIT fails to connect to the GAPIS, particularly due to ETXTBSY
	// look at https://github.com/google/gapid/pull/933 for more information
	retryString = "Failed to connect to the GAPIS server"
	// The value to use for gapit's --observe-frames flag.
	observeEveryNthFrame = 5
)

type client struct {
	store   *stash.Client
	manager Manager
	tempDir file.Path
}

// Run starts new trace client if any hardware is available.
func Run(ctx context.Context, store *stash.Client, manager Manager, tempDir file.Path) error {
	c := &client{store: store, manager: manager, tempDir: tempDir}
	job.OnDeviceAdded(ctx, c.onDeviceAdded)
	return nil
}

// TODO: not assume all android devices are valid trace targets
func (c *client) onDeviceAdded(ctx context.Context, host *device.Instance, target bind.Device) {
	traceOnTarget := func(ctx context.Context, t *Task) error {
		job.LockDevice(ctx, target)
		defer job.UnlockDevice(ctx, target)
		if target.Status(ctx) != bind.Status_Online {
			log.I(ctx, "Trying to trace %s on %s not started, device status %s",
				t.Input.Subject, target.Instance().GetSerial(), target.Status(ctx).String())
			return nil
		}
		return c.trace(ctx, target, t)
	}
	crash.Go(func() {
		if err := c.manager.Register(ctx, host, target.Instance(), traceOnTarget); err != nil {
			log.E(ctx, "Error running trace client: %v", err)
		}
	})
}

func (c *client) trace(ctx context.Context, d bind.Device, t *Task) error {
	if err := c.manager.Update(ctx, t.Action, job.Running, nil); err != nil {
		return err
	}
	var output *Output
	err := worker.RetryFunction(ctx, 4, time.Millisecond*100, func() (err error) {
		ctx, cancel := task.WithTimeout(ctx, traceTimeout)
		defer cancel()
		output, err = doTrace(ctx, t.Action, t.Input, c.store, d, c.tempDir)
		return
	})
	status := job.Succeeded
	if err != nil {
		status = job.Failed
		log.E(ctx, "Error running trace: %v", err)
	} else if output.Err != "" {
		status = job.Failed
		log.E(ctx, "Error during trace: %v", output.Err)
	}

	return c.manager.Update(ctx, t.Action, status, output)
}

// doTrace extracts input files and runs `gapit trace` on them, capturing the output. The output object will
// be partially filled in the event of an upload error from store in order to allow examination of the logs.
func doTrace(ctx context.Context, action string, in *Input, store *stash.Client, d bind.Device, tempDir file.Path) (*Output, error) {
	subject := tempDir.Join(action + ".apk")
	obb := tempDir.Join(action + ".obb")
	tracefile := tempDir.Join(action + ".gfxtrace")
	extractedDir := tempDir.Join(action + "_tools")
	extractedLayout, err := layout.NewPkgLayout(extractedDir, true)
	if err != nil {
		return nil, err
	}

	gapidAPK, err := extractedLayout.GapidApk(ctx, in.GetToolingLayout().GetGapidAbi())
	if err != nil {
		return nil, err
	}

	gapis, err := extractedLayout.Gapis(ctx)
	if err != nil {
		return nil, err
	}

	gapit, err := extractedLayout.Gapit(ctx)
	if err != nil {
		return nil, err
	}

	traceTime, err := ptypes.Duration(in.GetHints().GetTraceTime())
	if err != nil {
		traceTime = 10 * time.Minute // TODO: support Robot-wide override
	}

	defer func() {
		file.Remove(subject)
		file.Remove(tracefile)
		file.RemoveAll(extractedDir)
	}()
	if err := store.GetFile(ctx, in.Subject, subject); err != nil {
		return nil, err
	}
	if err := store.GetFile(ctx, in.Gapis, gapis); err != nil {
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
		"-for", traceTime.String(),
		"-disable-pcs",
		"-observe-frames", strconv.Itoa(observeEveryNthFrame),
		"-record-errors",
		"-serial", d.Instance().Serial,
		"-api", in.GetHints().GetAPI(),
	}

	if frames := in.GetHints().GetObserveFrames(); frames > 0 {
		params = append(params, "-capture-frames", strconv.Itoa(int(frames*observeEveryNthFrame+1)))
	}
	if in.Obb != "" {
		if err := store.GetFile(ctx, in.Obb, obb); err != nil {
			return nil, err
		}
		defer func() {
			file.Remove(obb)
		}()
		// TODO fix this
		// params = append(params, "-obb", obb.System())
		return log.Errf(ctx, nil, "OBBs are currently not supported")
	}
	cmd := shell.Command(gapit.System(), append(params, "apk:"+subject.System())...)
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	outputObj := &Output{}
	errs := []string{}
	log.I(ctx, "Running trace action %s", cmd)
	if err := cmd.Capture(outBuf, errBuf).Run(ctx); err != nil {
		if err := worker.NeedsRetry(err.Error()); err != nil {
			return nil, err
		}
		errs = append(errs, err.Error())
	}
	errs = append(errs, strings.TrimSpace(errBuf.String()))
	outputObj.Err = strings.Join(errs, "\n")
	if err := worker.NeedsRetry(outputObj.Err, retryString); err != nil {
		return nil, err
	}

	output := fmt.Sprintf("%s\n\n%s", cmd, strings.TrimSpace(outBuf.String()))
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
