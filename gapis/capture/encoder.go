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

package capture

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/pack"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
)

type encoder struct {
	c      *GraphicsCapture
	w      *pack.Writer
	cmdIDs map[api.Cmd]uint64
	resIDs map[id.ID]int64
}

func newEncoder(c *GraphicsCapture, w *pack.Writer) *encoder {
	return &encoder{
		c:      c,
		w:      w,
		cmdIDs: map[api.Cmd]uint64{},
		resIDs: map[id.ID]int64{id.ID{}: 0},
	}
}

func (e *encoder) encode(ctx context.Context) error {

	// Write the capture header.
	if err := e.w.Object(ctx, e.c.Header); err != nil {
		return err
	}

	if e.c.InitialState != nil {
		if err := e.initialState(ctx); err != nil {
			return err
		}
	}

	for _, cmd := range e.c.Commands {
		cmdID, err := e.startCmd(ctx, cmd)
		if err != nil {
			return err
		}
		if err := e.extras(ctx, cmd, cmdID); err != nil {
			return err
		}
		if err := e.endCmd(ctx, cmd); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) initialState(ctx context.Context) (err error) {
	var msg proto.Message
	var initialStateID uint64
	if msg, err = protoconv.ToProto(ctx, e.c.InitialState); err != nil {
		return err
	}
	if initialStateID, err = e.w.BeginGroup(ctx, msg); err != nil {
		return err
	}
	for _, initialMemory := range e.c.InitialState.Memory {
		if msg, err = protoconv.ToProto(ctx, initialMemory); err != nil {
			return err
		}
		if err = e.w.ChildObject(ctx, msg, initialStateID); err != nil {
			return err
		}
	}
	for _, initialAPI := range e.c.InitialState.APIs {
		if msg, err = protoconv.ToProto(ctx, initialAPI); err != nil {
			return err
		}
		if err = e.w.ChildObject(ctx, msg, initialStateID); err != nil {
			return err
		}
	}
	if err = e.w.EndGroup(ctx, initialStateID); err != nil {
		return err
	}
	return nil
}

func (e *encoder) childObject(ctx context.Context, obj interface{}, parentID uint64) error {
	var err error
	msg, ok := obj.(proto.Message)
	if !ok {
		if msg, err = protoconv.ToProto(ctx, obj); err != nil {
			return err
		}
	}
	if err := e.w.ChildObject(ctx, msg, parentID); err != nil {
		return err
	}
	return nil
}

func (e *encoder) startCmd(ctx context.Context, cmd api.Cmd) (uint64, error) {
	if cmdID, ok := e.cmdIDs[cmd]; ok {
		return cmdID, nil
	}
	cmdProto, err := protoconv.ToProto(ctx, cmd)
	if err != nil {
		return 0, err
	}

	cmdID, err := e.w.BeginGroup(ctx, cmdProto)
	if err != nil {
		return 0, err
	}
	e.cmdIDs[cmd] = cmdID
	return cmdID, nil
}

func (e *encoder) endCmd(ctx context.Context, cmd api.Cmd) error {
	id, ok := e.cmdIDs[cmd]
	if !ok {
		panic("Attempting to end command that was not in cmdIDs")
	}
	if err := e.w.EndGroup(ctx, id); err != nil {
		return err
	}
	delete(e.cmdIDs, cmd)
	return nil
}

func (e *encoder) extras(ctx context.Context, cmd api.Cmd, cmdID uint64) error {
	handledCall := false
	for _, extra := range cmd.Extras().All() {
		switch extra := extra.(type) {
		case *api.CmdObservations:
			for _, o := range extra.Reads {
				if err := e.childObject(ctx, o, cmdID); err != nil {
					return err
				}
			}
			if err := e.w.ChildObject(ctx, api.CmdCallFor(cmd), cmdID); err != nil {
				return err
			}
			handledCall = true
			for _, o := range extra.Writes {
				if err := e.childObject(ctx, o, cmdID); err != nil {
					return err
				}
			}
		default:
			if err := e.childObject(ctx, extra, cmdID); err != nil {
				return err
			}
		}
	}

	if !handledCall {
		if err := e.w.ChildObject(ctx, api.CmdCallFor(cmd), cmdID); err != nil {
			return err
		}
	}
	return nil
}

// RemapIndex remaps resource index to ID.
func (e *encoder) RemapIndex(ctx context.Context, index int64) (id.ID, error) {
	panic("Not allowed in encoder")
}

// RemapID remaps resource ID to index.
// protoconv callbacks use this to handle resources.
func (e *encoder) RemapID(ctx context.Context, id id.ID) (int64, error) {
	index, found := e.resIDs[id]
	if !found {
		index = int64(len(e.resIDs))
		e.resIDs[id] = index
		data, err := database.Resolve(ctx, id)
		if err != nil {
			return 0, err
		}
		res := &Resource{Index: index, Data: data.([]uint8)}
		if err := e.w.Object(ctx, res); err != nil {
			return 0, err
		}
	}
	return index, nil
}
