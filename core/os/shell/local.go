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
	"strings"
	"time"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
)

var (
	// LocalTarget is an implementation of Target that runs the command on the local machine using the exec package.
	LocalTarget Target = localTarget{}
)

type localTarget struct{}

type localProcess struct {
	exec  *exec.Cmd
	nbusy int
}

func (localTarget) Start(cmd Cmd) (Process, error) {
	p := &localProcess{
		exec:  exec.Command(cmd.Name, cmd.Args...),
		nbusy: 0,
	}
	p.exec.Dir = cmd.Dir
	p.exec.Stdout = cmd.Stdout
	p.exec.Stderr = cmd.Stderr
	p.exec.Stdin = cmd.Stdin
	p.exec.Env = cmd.Environment.Vars()
	// HACK:(baldwinn860)
	// On Unix systems the fork/exec model can hold a file descriptor open for a
	// very short amount of time (between the fork and the exec) in the child
	// process. If another thread has a executable being written before the fork
	// happens (to avoid the syscall.ForkLock), and before the exec happens it
	// finishes the write and executes itself, it can give an ETXTBUSY. This is
	// extremely short lived, but causes bugs in robot where we download and run
	// many executables in separate threads. Unfortunately the accepted fix for
	// this sort of issue is to do retry attempts with a wait in order to give
	// first child time to close-on-exec all of it's fds.
	// fix source: https://groups.google.com/forum/#!topic/golang-dev/4efaTJ5uA8Y
	// context: https://golang.org/src/syscall/exec_unix.go?h=StartProcess#L17
	for {
		err := p.exec.Start()

		// wait times are 100+200+400ms == 0.7s which seems reasonable.
		if err != nil && p.nbusy < 3 && strings.Contains(err.Error(), "text file busy") {
			time.Sleep(100 * time.Millisecond << uint(p.nbusy))
			p.nbusy++
		} else {
			return p, err
		}
	}
}

func (p *localProcess) Wait(ctx context.Context) error {
	res := make(chan error, 1)
	go func() { res <- p.exec.Wait() }()
	select {
	case err := <-res:
		if err == nil && p.nbusy > 0 {
			log.I(ctx, "Busy process completed successfully after %d retries", p.nbusy)
		}
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
