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

package replay

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/data/stash"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/test/robot/job"
)

type client struct {
	store   *stash.Client
	manager Manager
	tempDir file.Path
}

// Run starts new replay client if any hardware is available.
func Run(ctx context.Context, store *stash.Client, manager Manager, tempDir file.Path) error {
	c := &client{store: store, manager: manager, tempDir: tempDir}
	host := host.Instance(ctx)
	return manager.Register(ctx, host, host, c.replay)
}

func (c *client) replay(ctx context.Context, t *Task) error {
	if err := c.manager.Update(ctx, t.Action, job.Running, nil); err != nil {
		return err
	}
	output, err := doReplay(ctx, t.Action, t.Input, c.store, c.tempDir)
	status := job.Succeeded
	if err != nil {
		status = job.Failed
		log.E(ctx, "Error running replay: %v", err)
	}
	return c.manager.Update(ctx, t.Action, status, output)
}

func doReplay(ctx context.Context, action string, in *Input, store *stash.Client, tempDir file.Path) (*Output, error) {
	tracefile := tempDir.Join(action + ".gfxtrace")
	videofile := tempDir.Join(action + "_replay.mp4")

	extractedDir := tempDir.Join(action + "_tools")
	extractedLayout := layout.BinLayout(extractedDir)
	gapit, err := extractedLayout.Gapit(ctx)
	if err != nil {
		return nil, err
	}
	gapir, err := extractedLayout.Gapir(ctx)
	if err != nil {
		return nil, err
	}
	gapis, err := extractedLayout.Gapis(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		file.Remove(tracefile)
		file.Remove(videofile)
		file.RemoveAll(extractedDir)
	}()

	if err := store.GetFile(ctx, in.Trace, tracefile); err != nil {
		return nil, err
	}
	if err := store.GetFile(ctx, in.Gapit, gapit); err != nil {
		return nil, err
	}
	if err := store.GetFile(ctx, in.Gapis, gapis); err != nil {
		return nil, err
	}
	if err := store.GetFile(ctx, in.Gapir, gapir); err != nil {
		return nil, err
	}
	params := []string{
		"video",
		"-out", videofile.System(),
		tracefile.System(),
	}
	cmd := shell.Command(gapit.System(), params...)
	output, callErr := cmd.Call(ctx)
	output = fmt.Sprintf("%s\n\n%s", cmd, output)
	log.I(ctx, output)

	outputObj := &Output{}
	logID, err := store.UploadString(ctx, stash.Upload{Name: []string{"replay.log"}}, output)
	if err != nil {
		return outputObj, err
	}
	outputObj.Log = logID
	videoID, err := store.UploadFile(ctx, videofile)
	if err != nil {
		return outputObj, err
	}
	outputObj.Video = videoID
	return outputObj, callErr
}
