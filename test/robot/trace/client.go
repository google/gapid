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
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/data/stash"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/test/robot/job"
)

type client struct {
	runners []*runner
}

type runner struct {
	store   *stash.Client
	manager Manager
	device  adb.Device
	tempDir file.Path
}

// Run starts new trace client if any hardware is available.
func Run(ctx log.Context, store *stash.Client, manager Manager, tempDir file.Path) error {
	c := &client{}
	host := device.Host(ctx)
	// TODO: make the list of adb devices dynamic
	devices, err := adb.Devices(ctx)
	if err != nil {
		ctx.Notice().Logf("adb devices failed: %v", err)
		return err
	}
	ctx.Notice().Logf("adb devices gave: %v", devices)
	for _, d := range devices {
		d := d
		// TODO: not assume all android devices are valid trace targets
		r := &runner{store: store, manager: manager, device: d, tempDir: tempDir}
		c.runners = append(c.runners, r)
		go func() {
			err := manager.Register(ctx, host, d.Instance(), r.trace)
			if err != nil {
				jot.Fail(ctx, err, "Running trace client")
			}
		}()
	}
	return nil
}

func (r *runner) trace(ctx log.Context, t *Task) (err error) {
	if err := r.manager.Update(ctx, t.Action, job.Running, nil); err != nil {
		return err
	}
	output, err := doTrace(ctx, t.Action, t.Input, r.store, r.device, r.tempDir)
	status := job.Succeeded
	if err != nil {
		status = job.Failed
		jot.Fail(ctx, err, "Running trace")
	}
	return r.manager.Update(ctx, t.Action, status, output)
}

func doTrace(ctx log.Context, action string, in *Input, store *stash.Client, d adb.Device, tempDir file.Path) (*Output, error) {
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
		traceTime = 10 * time.Second // TODO: support Robot-wide override
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
	}
	cmd := shell.Command(gapit.System(), params...)
	output, callErr := cmd.Call(ctx)
	output = fmt.Sprintf("%s\n\n%s", cmd, output)

	outputObj := &Output{}
	logID, err := store.UploadString(ctx, stash.Upload{Name: []string{"trace.log"}}, output)
	if err != nil {
		return outputObj, err
	}
	outputObj.Log = logID
	traceID, err := store.UploadFile(ctx, tracefile)
	if err != nil {
		return outputObj, err
	}
	outputObj.Trace = traceID
	return outputObj, callErr
}
