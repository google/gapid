package capture

import (
	"context"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/pack"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
)

type encoder struct {
	c      *Capture
	w      *pack.Writer
	cmdIDs map[api.Cmd]uint64
	seen   map[id.ID]bool
}

func newEncoder(c *Capture, w *pack.Writer) *encoder {
	return &encoder{
		c:      c,
		w:      w,
		cmdIDs: map[api.Cmd]uint64{},
		seen:   map[id.ID]bool{},
	}
}

func (e *encoder) encode(ctx context.Context) error {
	// Write the capture header.
	if err := e.w.Object(ctx, e.c.Header); err != nil {
		return err
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

func (e *encoder) childObject(ctx context.Context, obj interface{}, parentID uint64) error {
	msg, err := protoconv.ToProto(ctx, obj)
	if err != nil {
		return err
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
	if id := cmd.Caller(); id != api.CmdNoID {
		// TODO: Instead of just starting a caller as soon as the first child
		// is seen, perhaps we should store the start points in the Capture?
		if _, err := e.startCmd(ctx, e.c.Commands[id]); err != nil {
			return 0, err
		}
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
				if err := e.observation(ctx, o, cmdID); err != nil {
					return err
				}
			}
			if err := e.w.ChildObject(ctx, api.CmdCallFor(cmd), cmdID); err != nil {
				return err
			}
			handledCall = true
			for _, o := range extra.Writes {
				if err := e.observation(ctx, o, cmdID); err != nil {
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

func (e *encoder) observation(ctx context.Context, o api.CmdObservation, cmdID uint64) error {
	if !e.seen[o.ID] {
		data, err := database.Resolve(ctx, o.ID)
		if err != nil {
			return err
		}
		res := &Resource{Id: o.ID[:], Data: data.([]uint8)}
		if err := e.w.Object(ctx, res); err != nil {
			return err
		}
		e.seen[o.ID] = true
	}
	if err := e.childObject(ctx, o, cmdID); err != nil {
		return err
	}
	return nil
}
