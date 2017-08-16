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

package shell

import (
	"context"
	"os/exec"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
)

var (
	// LocalTarget is an implementation of Target that runs the command on the local machine using the exec package.
	LocalTarget Target = localTarget{}
)

type localTarget struct{}

type localProcess struct {
	exec *exec.Cmd
}

func (localTarget) Start(cmd Cmd) (Process, error) {
	p := &localProcess{
		exec: exec.Command(cmd.Name, cmd.Args...),
	}
	p.exec.Dir = cmd.Dir
	p.exec.Stdout = cmd.Stdout
	p.exec.Stderr = cmd.Stderr
	p.exec.Stdin = cmd.Stdin
	p.exec.Env = cmd.Environment.Vars()
	return p, p.exec.Start()
}

func (p *localProcess) Wait(ctx context.Context) error {
	res := make(chan error, 1)
	go func() { res <- p.exec.Wait() }()
	select {
	case err := <-res:
		return err
	case <-task.ShouldStop(ctx):
		log.W(ctx, "Killing %v (context cancelled)", p.exec.Path)
		p.exec.Process.Kill()
		return task.StopReason(ctx)
	}
}

func (p *localProcess) Kill() error {
	return p.exec.Process.Kill()
}
