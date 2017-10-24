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

	if r, ok := obj.(api.ResourceReference); ok {
		var err error
		obj, err = r.RemapResourceIDs(func(id *id.ID) error {
			newID, found := d.builder.idmap[*id]
			if !found {
				return fmt.Errorf("Can not remap resource. %v not found.", id)
			}
			*id = newID
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	switch obj := obj.(type) {
	case *Header:
		d.header = obj
		if d.header.Version != CurrentCaptureVersion {
			return nil, ErrUnsupportedVersion{Version: d.header.Version}
		}
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
	}

	return obj, nil
}
