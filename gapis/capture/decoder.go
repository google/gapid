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
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/api"
)

type cmdGroup struct {
	cmd      api.Cmd
	invoked  bool
	children []api.Cmd
}

type decoder struct {
	header  *Header
	builder *builder
	groups  map[uint64]interface{}
}

func newDecoder() *decoder {
	return &decoder{
		builder: newBuilder(),
		groups:  map[uint64]interface{}{},
	}
}

// RemapIndex remaps resource index to ID.
// protoconv callbacks use this to handle resources.
func (d *decoder) RemapIndex(ctx context.Context, index int64) (id.ID, error) {
	if index < 0 {
		// Negative values encode index from the end of the array.
		// This is currently unused as it is difficult to encode.
		index = int64(len(d.builder.resIDs)) + index
	}
	if !(0 <= index && index < int64(len(d.builder.resIDs))) {
		return id.ID{}, fmt.Errorf("Can not remap resource %v", index)
	}
	return d.builder.resIDs[index], nil
}

// RemapID remaps resource ID to index.
func (d *decoder) RemapID(ctx context.Context, id id.ID) (int64, error) {
	panic("Not allowed in decoder")
}

func (d *decoder) BeginGroup(ctx context.Context, msg proto.Message, id uint64) error {
	obj, err := d.decode(ctx, msg)
	if err != nil {
		return err
	}
	d.groups[id] = obj
	return nil
}

func (d *decoder) BeginChildGroup(ctx context.Context, msg proto.Message, id, parentID uint64) error {
	obj, err := d.decode(ctx, msg)
	if err != nil {
		return err
	}
	d.groups[id] = obj
	return d.add(ctx, obj, d.groups[parentID])
}

func (d *decoder) endGroupImpl(ctx context.Context, id uint64, terminated bool) error {
	obj := d.groups[id]
	delete(d.groups, id)

	switch obj := obj.(type) {
	case *cmdGroup:
		obj.invoked = true
		obj.cmd.SetTerminated(terminated)

		if len(obj.children) > 0 {
			return fmt.Errorf("Nested commands not supported, but command %v has children", id)
		}

		d.builder.addCmd(ctx, obj.cmd)
	}

	return nil
}

func (d *decoder) EndGroup(ctx context.Context, id uint64) error {
	return d.endGroupImpl(ctx, id, true)
}

func (d *decoder) EndGroupNonTerminated(ctx context.Context, id uint64) error {
	return d.endGroupImpl(ctx, id, false)
}

func (d *decoder) Object(ctx context.Context, msg proto.Message) error {
	_, err := d.decode(ctx, msg)
	return err
}

func (d *decoder) ChildObject(ctx context.Context, msg proto.Message, parentID uint64) error {
	obj, err := d.decode(ctx, msg)
	if err != nil {
		return err
	}
	return d.add(ctx, obj, d.groups[parentID])
}

func (d *decoder) add(ctx context.Context, child, parent interface{}) error {
	if parent, ok := parent.(*cmdGroup); ok {
		// adding something to a command

		// is the child the result?
		if res, ok := child.(proto.Message); ok {
			if cmd, ok := parent.cmd.(api.CmdWithResult); ok {
				if cmd.SetCallResult(ctx, res) == nil {
					parent.invoked = true
					return nil
				}
			}
		}

		switch obj := child.(type) {
		case api.Cmd:
			parent.children = append(parent.children, obj)

		case *cmdGroup:
			parent.children = append(parent.children, obj.cmd)

		case api.CmdObservation:
			d.builder.addObservation(ctx, &obj)
			observations := parent.cmd.Extras().GetOrAppendObservations()
			if !parent.invoked {
				observations.Reads = append(observations.Reads, obj)
			} else {
				observations.Writes = append(observations.Writes, obj)
			}

		case *api.CmdCall:
			parent.invoked = true

		default:
			parent.cmd.Extras().Add(obj)
		}
	}
	if _, ok := parent.(*InitialState); ok {
		switch obj := child.(type) {
		case api.CmdObservation:
			return d.builder.addInitialMemory(ctx, obj)
		case api.State:
			return d.builder.addInitialState(ctx, obj)
		default:
			return fmt.Errorf("We do not expect a %T as a child of an initial state: %+200v", obj, obj)
		}
	}
	return nil
}

func (d *decoder) unmarshal(ctx context.Context, in proto.Message) (interface{}, error) {
	obj, err := protoconv.ToObject(ctx, in)
	if err != nil {
		if e, ok := err.(protoconv.ErrNoConverterRegistered); ok && e.Object == in {
			return in, nil // No registered converter. Treat proto as the object.
		}
		return nil, err
	}
	return obj, nil
}

func (d *decoder) decode(ctx context.Context, in proto.Message) (interface{}, error) {
	obj, err := d.unmarshal(ctx, in)
	if err != nil {
		return nil, err
	}

	switch obj := obj.(type) {
	case *Header:
		d.header = obj
		if d.header.Version != CurrentCaptureVersion {
			return nil, ErrUnsupportedVersion{Version: d.header.Version}
		}
		return in, nil

	case *Resource:
		if err := d.builder.addRes(ctx, obj.Index, obj.Data); err != nil {
			return nil, err
		}
		return in, nil

	case *TraceMessage:
		d.builder.addMessage(ctx, obj)
		return in, nil

	case api.Cmd:
		return &cmdGroup{cmd: obj}, nil

	case *InitialState:
		d.builder.initialState = obj
	}

	return obj, nil
}

func (d *decoder) flush(ctx context.Context) {
	for k := range d.groups {
		d.EndGroupNonTerminated(ctx, k)
	}
}
