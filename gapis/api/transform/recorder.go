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

package transform

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/gapid/gapis/api"
)

// Recorder is a Writer that record all commands that pass through it.
type Recorder struct {
	S          *api.GlobalState
	Cmds       []api.Cmd
	CmdsAndIDs []CmdAndID
}

// State returns the state object associated with this writer.
func (r *Recorder) State() *api.GlobalState {
	return r.S
}

// MutateAndWrite records the command and id into the Cmds and CmdsAndIDs lists
// and if the Recorder has a state, mutates this state object.
func (r *Recorder) MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
	if r.S != nil {
		if err := cmd.Mutate(ctx, id, r.S, nil, nil); err != nil {
			return err
		}
	}
	r.Cmds = append(r.Cmds, cmd)
	r.CmdsAndIDs = append(r.CmdsAndIDs, CmdAndID{cmd, id})
	return nil
}

// Required to implement Writer interface
func (r *Recorder) NotifyPreLoop(ctx context.Context) {
}

// Required to implement Writer interface
func (r *Recorder) NotifyPostLoop(ctx context.Context) {
}

// CmdAndID is a pair of api.Cmd and api.CmdID
type CmdAndID struct {
	Cmd api.Cmd
	ID  api.CmdID
}

// CmdAndIDList is a list of CmdAndIDs.
type CmdAndIDList []CmdAndID

// CmdWithID is a api.Cmd that embeds its own ID.
type CmdWithID interface {
	api.Cmd
	ID() api.CmdID
}

// NewCmdAndIDList takes a mix of CmdWithIDs and Cmds with an ID property and
// returns a CmdAndIDList.
// Cmds that have an ID property are transformed into CmdAndID by using the ID.
func NewCmdAndIDList(cmds ...interface{}) CmdAndIDList {
	l := CmdAndIDList{}
	for _, a := range cmds {
		switch a := a.(type) {
		case api.Cmd:
			p, err := api.GetParameter(a, "ID")
			if err != nil {
				panic(fmt.Errorf("Command %v does not have ID property: %v", a.CmdName(), err))
			}
			v := reflect.ValueOf(p)
			if v.Kind() != reflect.Uint64 {
				panic(fmt.Errorf("Command %v has unexpected ID property type: %T", a.CmdName(), p))
			}
			l = append(l, CmdAndID{a, api.CmdID(v.Uint())})
		case CmdAndID:
			l = append(l, a)
		default:
			panic(fmt.Errorf("list only accepts types CmdWithID or CmdAndID. Got %T", a))
		}
	}
	return l
}

func (l *CmdAndIDList) Write(ctx context.Context, id api.CmdID, c api.Cmd) {
	*l = append(*l, CmdAndID{c, id})
}
