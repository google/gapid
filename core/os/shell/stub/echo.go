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
	"strings"
	"sync"

	"github.com/google/gapid/core/os/shell"
)

// Echo is an implementation of Target that acts like the echo command, printing the command arguments parameters to
// stdout.
type Echo struct{}

func (Echo) Start(cmd shell.Cmd) (shell.Process, error) {
	return &echo{cmd: cmd}, nil
}

type echo struct {
	once sync.Once
	cmd  shell.Cmd
}

func (p *echo) Wait(ctx context.Context) error {
	p.once.Do(func() {
		if p.cmd.Stdout != nil {
			io.WriteString(p.cmd.Stdout, strings.Join(p.cmd.Args, " "))
		}
	})
	return nil
}

func (p *echo) Kill() error {
	return nil
}
