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

package stub

import (
	"context"
	"io"
	"sync"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/os/shell"
)

// Response is an implementation of Target that always gives exactly the same response.
type Response struct {
	// WaitSignal if set is waited on inside the Wait method of the process
	WaitSignal task.Signal
	// KillTask is invoked if it is non nil and the Kill method is called.
	KillTask task.Task
	// StartErr is returned by the target Start method if set.
	StartErr error
	// WaitErr is returned from the Wait method of the Process if set.
	WaitErr error
	// KillErr is returned from the Kill method of the Process if set.
	KillErr error
	// Stdout is the string to write as the standard output of the process.
	Stdout string
	// Stderr is the string to write as the standard error of the process.
	Stderr string
}

func (t *Response) Start(cmd shell.Cmd) (shell.Process, error) {
	if t.StartErr != nil {
		return nil, t.StartErr
	}
	return &responseProcess{cmd: cmd, response: t}, nil
}

type responseProcess struct {
	once     sync.Once
	cmd      shell.Cmd
	response *Response
}

func (p *responseProcess) Wait(ctx context.Context) error {
	if p.response.WaitSignal != nil {
		p.response.WaitSignal.Wait(ctx)
	}
	p.once.Do(func() {
		if p.cmd.Stdout != nil {
			io.WriteString(p.cmd.Stdout, p.response.Stdout)
		}
		if p.cmd.Stderr != nil {
			io.WriteString(p.cmd.Stdout, p.response.Stderr)
		}
	})
	return p.response.WaitErr
}

func (p *responseProcess) Kill() error {
	if p.response.KillTask != nil {
		p.response.KillTask(context.Background())
	}
	return p.response.KillErr
}
