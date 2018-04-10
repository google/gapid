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

package master

import (
	"context"
	"sync"

	"github.com/google/gapid/core/event/task"
)

type satellite struct {
	lock   sync.Mutex
	info   *Satellite
	issues chan issue
}

// issue is used to package a command with a channel to feed the result of
// handling the command.
type issue struct {
	command *Command
	result  chan error
}

func newSatellite(ctx context.Context, name string, services ServiceList) *satellite {
	return &satellite{
		info: &Satellite{
			Name:     name,
			Services: &services,
		},
		issues: make(chan issue),
	}
}

// processCommands reads the issues from the channel and hands them to the command handler, sending
// the result back through the issue channel.
func (sat *satellite) processCommands(ctx context.Context, handler CommandHandler) {
	for {
		select {
		case <-task.ShouldStop(ctx):
			return
		case i := <-sat.issues:
			i.result <- handler(ctx, i.command)
			close(i.result)
		}
	}
}

// sendCommand posts an issue for the command into the channel, then blocks until it gets a result.
func (sat *satellite) sendCommand(ctx context.Context, command *Command) {
	result := make(chan error)
	sat.lock.Lock()
	if sat.issues != nil {
		sat.issues <- issue{command: command, result: result}
	}
	sat.lock.Unlock()
	err := <-result
	if err != nil {
		sat.lock.Lock()
		if sat.issues != nil {
			close(sat.issues)
			sat.issues = nil
		}
		sat.lock.Unlock()
	}
}
