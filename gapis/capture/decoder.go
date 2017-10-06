package capture

import (
	"context"
	fmt "fmt"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory/memory_pb"
)

type cmdGroup struct {
	cmd      api.Cmd
	invoked  bool
	children []api.Cmd
}

type Reference struct {
	ID uint64
}

type InitialAPIState struct {
	State        api.State
	MemoryWrites []*memory_pb.MemoryWrite
}

type initialStateGroup struct {
	apis map[api.ID]*InitialAPIState
}

type decoder struct {
	header               *Header
	builder              *builder
	groups               map[uint64]interface{}
	unresolvedReferences map[uint64][]interface{}
	resolveReferences    map[uint64]interface{}
}

func newDecoder() *decoder {
	return &decoder{
		builder:              newBuilder(),
		groups:               map[uint64]interface{}{},
		unresolvedReferences: map[uint64][]interface{}{},
		resolveReferences:    map[uint64]interface{}{},
	}
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

func (d *decoder) EndGroup(ctx context.Context, id uint64) error {
	obj := d.groups[id]
	delete(d.groups, id)

	switch obj := obj.(type) {
	case *cmdGroup:
		id := d.builder.addCmd(ctx, obj.cmd)
		for _, c := range obj.children {
			c.SetCaller(id)
		}
	}

	return nil
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
				if cmd.SetResult(res) == nil {
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
	if parent, ok := parent.(*initialStateGroup); ok {
		res, ok := child.(api.State)
		if !ok {
			return fmt.Errorf("We only expect an initial state inside of a State group, got %T", child)
		}
		if _, ok := parent.apis[res.GetID()]; ok {
			return fmt.Errorf("We have more than one set of initial state for API %d", res.GetID())
		}
		parent.apis[res.GetID()] = &InitialAPIState{State: res}
		d.builder.addInitialState(ctx, res.GetID(), parent.apis[res.GetID()])
	}
	if parent, ok := parent.(api.State); ok {
		switch obj := child.(type) {
		case *memory_pb.MemoryWrite:
			d.builder.addInitialMemory(parent, obj)
		case *Reference:
			// Intentionally empty
		default:
			return fmt.Errorf("We do not expect a %T, %T, as a child of an API state", obj, child)
		}
	}
	if parent, ok := parent.(*Reference); ok {
		d.resolveReferences[parent.ID] = child
	}

	return nil
}

func (d *decoder) unmarshal(ctx context.Context, in proto.Message) (interface{}, error) {
	obj, err := protoconv.ToObject(ctx, func(idx uint64, i interface{}) {
		if _, ok := d.unresolvedReferences[idx]; !ok {
			d.unresolvedReferences[idx] = []interface{}{}
		}
		d.unresolvedReferences[idx] = append(d.unresolvedReferences[idx], i)
	}, in)
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

	if r, ok := obj.(api.ResourceReference); ok {
		obj = r.RemapResourceIDs(d.builder.idmap)
	}

	switch obj := obj.(type) {
	case *Header:
		d.header = obj
		return in, nil

	case *Resource:
		var rID id.ID
		copy(rID[:], obj.Id)
		if err := d.builder.addRes(ctx, rID, obj.Data); err != nil {
			return nil, err
		}
		return in, nil

	case api.Cmd:
		return &cmdGroup{cmd: obj}, nil

	case *State:
		return &initialStateGroup{map[api.ID]*InitialAPIState{}}, nil
	case *memory_pb.Reference:
		return &Reference{ID: obj.GetIdentifier()}, nil
	}

	return obj, nil
}
