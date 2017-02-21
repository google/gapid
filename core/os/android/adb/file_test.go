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

package adb_test

import (
	"testing"

	"github.com/google/gapid/core/log"
)

func TestFilePush(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "push_device")
	err := d.Push(ctx, "local_file", "remote_file")
	expectedCommand(ctx, adbPath.System()+` -s push_device push local_file remote_file`, err)
}

func TestFilePull(t_ *testing.T) {
	ctx := log.Testing(t_)
	d := mustConnect(ctx, "pull_device")
	err := d.Pull(ctx, "remote_file", "local_file")
	expectedCommand(ctx, adbPath.System()+` -s pull_device pull remote_file local_file`, err)
}
