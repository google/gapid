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

package testcmd

import (
	"context"

	"github.com/google/gapid/gapis/api"
)

// Writer is a transform.Writer that record all commands that pass through it.
type Writer struct {
	S          *api.GlobalState
	Cmds       []api.Cmd
	CmdsAndIDs []CmdAndID
}

func (w *Writer) State() *api.GlobalState {
	return w.S
}

func (w *Writer) MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) {
	if w.S != nil {
		cmd.Mutate(ctx, id, w.S, nil)
	}
	w.Cmds = append(w.Cmds, cmd)
	w.CmdsAndIDs = append(w.CmdsAndIDs, CmdAndID{cmd, id})
}

type CmdAndID struct {
	Cmd api.Cmd
	Id  api.CmdID
}

type CmdAndIDList []CmdAndID

// List takes a mix of Cmds and CmdIDsAndCmd and returns a CmdIDListAndCmd.
// Cmds are transformed into CmdIDsAndCmd by using the ID field as the command
// id.
func List(cmds ...interface{}) CmdAndIDList {
	l := CmdAndIDList{}
	for _, a := range cmds {
		switch a := a.(type) {
		case *A:
			l = append(l, CmdAndID{a, a.ID})
		case *B:
			l = append(l, CmdAndID{a, a.ID})
		case CmdAndID:
			l = append(l, a)
		default:
			panic("list only accepts types A, B or CmdAndID")
		}
	}
	return l
}

func (l *CmdAndIDList) Write(ctx context.Context, id api.CmdID, c api.Cmd) {
	*l = append(*l, CmdAndID{c, id})
}
