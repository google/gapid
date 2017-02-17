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

// +build linux darwin

package shell_test

import (
	"bytes"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/shell"
)

func TestCommand(t *testing.T) {
	ctx := log.Testing(t)
	err := shell.Command("echo", "test").Run(ctx)
	assert.With(ctx).ThatError(err).Succeeded()
}

func TestCommandFailed(t *testing.T) {
	const expect = `Error:Failed:Wait<exit status 1>`
	ctx := log.Testing(t)
	err := shell.Command("false").Run(ctx)
	assert.With(ctx).ThatError(err).HasMessage(expect)
}

func TestCommandBadProgram(t *testing.T) {
	const expect = `Error:Failed:Start<exec: "not#a#program": executable file not found in $PATH>`
	ctx := log.Testing(t)
	err := shell.Command("not#a#program", "test").Run(ctx)
	assert.With(ctx).ThatError(err).HasMessage(expect)
}

func TestCommandBadDir(t *testing.T) {
	const expect = `Error:Failed:Start<chdir not#a#dir: no such file or directory>`
	ctx := log.Testing(t)
	err := shell.Command("echo", "test").In("not#a#dir").Run(ctx)
	assert.With(ctx).ThatError(err).HasMessage(expect)
}

func TestCommandCancel(t *testing.T) {
	const expect = `Error:Failed:Wait<context canceled>`
	ctx := log.Testing(t)
	child, cancel := task.WithCancel(ctx)
	cancel()
	err := shell.Command("sleep", "1").Run(child)
	assert.With(ctx).ThatError(err).HasMessage(expect)
}

func TestCommandCaptureStdout(t *testing.T) {
	const expect = "echo to stdout\n"
	ctx := log.Testing(t)
	buf := &bytes.Buffer{}
	err := shell.Command("echo", "echo to stdout").Capture(buf, nil).Run(ctx)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).ThatString(buf).Equals(expect)
}

func TestCommandCaptureStderr(t *testing.T) {
	const expect = "print to stderr\n"
	ctx := log.Testing(t)
	buf := &bytes.Buffer{}
	shell.Command("awk", `END {print "print to stderr" > "/dev/stderr"}`).Capture(nil, buf).Run(ctx)
	assert.With(ctx).ThatString(buf).Equals(expect)
}

func TestCommandStdin(t *testing.T) {
	const expect = "Hello you\nGoodbye me"
	ctx := log.Testing(t)
	stdin := bytes.NewBufferString("1 Hello to you\n2 Goodbye from me\n")
	output, _ := shell.Command("awk", `{ print $2 " " $4 }`).Read(stdin).Call(ctx)
	assert.With(ctx).ThatString(output).Equals(expect)
}

func TestCommandCall(t *testing.T) {
	const expect = "echo to stdout"
	ctx := log.Testing(t)
	output, err := shell.Command("echo", "echo to stdout").Call(ctx)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).ThatString(output).Equals(expect)
}

func TestCommandEnvironment(t *testing.T) {
	const expect = "message from environment\n"
	ctx := log.Testing(t)
	buf := &bytes.Buffer{}
	env := shell.NewEnv().Set("MESSAGE", "message from environment")
	err := shell.Command("printenv", "MESSAGE").
		Capture(buf, nil).
		Env(env).
		Run(ctx)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).ThatString(buf).Equals(expect)
}

type errorTarget struct{}

func (t errorTarget) Start(cmd shell.Cmd) (shell.Process, error) {
	return nil, fault.Const("AlwaysFail")
}

func TestCommandOn(t *testing.T) {
	const expect = "echo to stdout"
	ctx := log.Testing(t)
	_, err := shell.Command("echo", "echo to stdout").On(errorTarget{}).Call(ctx)
	assert.With(ctx).ThatError(err).HasMessage("Error:Failed:Start<AlwaysFail>")
}
