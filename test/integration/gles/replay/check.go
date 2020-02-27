// Copyright (C) 2018 Google Inc.
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
	"sync"

	"github.com/google/gapid/core/os/device"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type intentCfg struct {
	intent replay.Intent
	config replay.Config
}

func (c intentCfg) String() string {
	return fmt.Sprintf("Context: %+v, Config: %+v", c.intent, c.config)
}

func checkImage(ctx context.Context, name string, got *image.Data, threshold float64) {
	if *generateReferenceImages != "" {
		storeReferenceImage(ctx, *generateReferenceImages, name, got)
	} else {
		quantized := quantizeImage(got)
		expected := loadReferenceImage(ctx, name)
		diff, err := image.Difference(quantized, expected)
		assert.For(ctx, "CheckImage").ThatError(err).Succeeded()
		assert.For(ctx, "CheckImage").ThatFloat(float64(diff)).IsAtMost(threshold)
	}
}

func checkIssues(ctx context.Context, c *path.Capture, d *device.Instance, expected []replay.Issue, done *sync.WaitGroup) {
	mgr := replay.GetManager(ctx)
	if done != nil {
		defer done.Done()
	}
	ctx, _ = task.WithTimeout(ctx, replayTimeout)
	intent := replay.Intent{
		Capture: c,
		Device:  path.NewDevice(d.ID.ID()),
	}
	issues, err := gles.API{}.QueryIssues(ctx, intent, mgr, 1, false, nil)
	if assert.For(ctx, "err").ThatError(err).Succeeded() {
		assert.For(ctx, "issues").ThatSlice(issues).DeepEquals(expected)
	}
}

func checkReport(ctx context.Context, c *path.Capture, d *device.Instance, cmds []api.Cmd, expected []string, done *sync.WaitGroup) {
	if done != nil {
		defer done.Done()
	}

	report, err := resolve.Report(ctx, c.Report(path.NewDevice(d.ID.ID()), nil, false), nil)
	assert.For(ctx, "err").ThatError(err).Succeeded()

	got := []string{}
	for _, e := range report.Items {
		if e.Command != nil {
			got = append(got, fmt.Sprintf("%s@%d: %s: %v", e.Severity.String(), e.Command.Indices, cmds[e.Command.Indices[0]], report.Msg(e.Message).Text(nil)))
		} else {
			got = append(got, fmt.Sprintf("%s /%v", e.Severity.String(), report.Msg(e.Message).Text(nil)))
		}
	}
	assert.For(ctx, "got").ThatSlice(got).Equals(expected)
}

func checkColorBuffer(ctx context.Context, c *path.Capture, d *device.Instance, w, h uint32, threshold float64, name string, after api.CmdID, done *sync.WaitGroup) {
	mgr := replay.GetManager(ctx)
	ctx = log.Enter(ctx, "ColorBuffer")
	ctx = log.V{"name": name, "after": after}.Bind(ctx)
	if done != nil {
		defer done.Done()
	}
	ctx, _ = task.WithTimeout(ctx, replayTimeout)
	intent := replay.Intent{
		Capture: c,
		Device:  path.NewDevice(d.ID.ID()),
	}
	img, err := gles.API{}.QueryFramebufferAttachment(
		ctx, intent, mgr, []uint64{uint64(after)}, w, h, api.FramebufferAttachment_Color0, 0, service.DrawMode_NORMAL, false, false, nil)
	if !assert.For(ctx, "err").ThatError(err).Succeeded() {
		return
	}
	checkImage(ctx, name, img, threshold)
}

func checkTextureBuffer(ctx context.Context, c *path.Capture, d *device.Instance, w, h uint32, threshold float64, name string, after api.CmdID, tex gles.TextureId, done *sync.WaitGroup) {
	ctx = log.Enter(ctx, "TextureBuffer")
	ctx = log.V{"name": name, "after": after}.Bind(ctx)
	if done != nil {
		defer done.Done()
	}
	ctx, _ = task.WithTimeout(ctx, replayTimeout)

	cmdPath := c.Command(uint64(after))

	cmd, err := resolve.Cmd(ctx, cmdPath, nil)
	if !assert.For(ctx, "resolve cmd").ThatError(err).Succeeded() {
		return
	}

	thread := cmd.Thread()

	globalState, err := resolve.GlobalState(ctx, cmdPath.GlobalStateAfter(), nil)
	if !assert.For(ctx, "resolve global state").ThatError(err).Succeeded() {
		return
	}

	state := gles.GetState(globalState)

	context, ok := state.Contexts().Lookup(thread)
	if !assert.For(ctx, "lookup context").That(ok).Equals(true) {
		return
	}

	t := context.Objects().Textures().Get(tex)
	if !assert.For(ctx, "texture found").That(!t.IsNil()).Equals(true) {
		return
	}

	res, err := t.ResourceData(ctx, globalState, cmdPath)
	if !assert.For(ctx, "resource data").ThatError(err).Succeeded() {
		return
	}

	imginfo := res.GetTexture().GetTexture_2D().GetLevels()[0]

	img, err := imginfo.Data(ctx)
	if !assert.For(ctx, "image data").ThatError(err).Succeeded() {
		return
	}

	checkImage(ctx, name, img, threshold)
}

func checkDepthBuffer(ctx context.Context, c *path.Capture, d *device.Instance, w, h uint32, threshold float64, name string, after api.CmdID, done *sync.WaitGroup) {
	mgr := replay.GetManager(ctx)
	ctx = log.Enter(ctx, "DepthBuffer")
	ctx = log.V{"name": name, "after": after}.Bind(ctx)
	if done != nil {
		defer done.Done()
	}
	ctx, _ = task.WithTimeout(ctx, replayTimeout)
	intent := replay.Intent{
		Capture: c,
		Device:  path.NewDevice(d.ID.ID()),
	}
	img, err := gles.API{}.QueryFramebufferAttachment(
		ctx, intent, mgr, []uint64{uint64(after)}, w, h, api.FramebufferAttachment_Depth, 0, service.DrawMode_NORMAL, false, false, nil)
	if !assert.For(ctx, "err").ThatError(err).Succeeded() {
		return
	}
	checkImage(ctx, name, img, threshold)
}

func checkReplay(ctx context.Context, c *path.Capture, d *device.Instance, expectedBatchCount int, doReplay func()) {
	expectedIntent := replay.Intent{
		Capture: c,
		Device:  path.NewDevice(d.ID.ID()),
	}

	batchCount := 0
	uniqueIntentConfigs := map[intentCfg]struct{}{}
	replay.Events.OnReplay = func(device bind.Device, intent replay.Intent, config replay.Config) {
		assert.For(ctx, "Replay intent").That(intent).DeepEquals(expectedIntent)
		batchCount++
		uniqueIntentConfigs[intentCfg{intent, config}] = struct{}{}
	}

	doReplay()

	replay.Events.OnReplay = nil // Avoid stale assertions in subsequent tests that don't use checkReplay.
	if assert.For(ctx, "Batch count").That(batchCount).Equals(expectedBatchCount) {
		log.I(ctx, "%d unique intent-config pairs:", len(uniqueIntentConfigs))
		for cc := range uniqueIntentConfigs {
			log.I(ctx, " • %v", cc)
		}
	}
}
