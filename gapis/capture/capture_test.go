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

package capture_test

import (
	"bytes"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/testcmd"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
)

func TestCaptureExportImport(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	header := &capture.Header{ABI: device.WindowsX86_64}
	cmds := []api.Cmd{testcmd.P, testcmd.Q}
	p, err := capture.New(ctx, arena.New(), "test", header, cmds)
	if !assert.For(ctx, "capture.New").ThatError(err).Succeeded() {
		return
	}
	ctx = capture.Put(ctx, p)

	buf := &bytes.Buffer{}
	err = capture.Export(capture.Put(ctx, p), p, buf)
	if !assert.For(ctx, "capture.Export").ThatError(err).Succeeded() {
		return
	}

	ip, err := capture.Import(ctx, "imported", buf.Bytes())
	if !assert.For(ctx, "capture.Import").ThatError(err).Succeeded() {
		return
	}

	ic, err := capture.Resolve(capture.Put(ctx, ip))
	if !assert.For(ctx, "capture.Resolve").ThatError(err).Succeeded() {
		return
	}

	assert.For(ctx, "got").That(ic.Commands).DeepEquals(cmds)
}
