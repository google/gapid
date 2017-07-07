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

package api

import (
	"context"

	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api/core/core_pb"
	"github.com/google/gapid/gapis/atom/atom_pb"
)

// ProtoToCmd returns a function that converts all the storage commands it is
// handed, passing the generated live atoms to the handler.
// You must call this with a nil to flush the final atom.
func ProtoToCmd(handler func(Cmd)) func(context.Context, atom_pb.Atom) error {
	var (
		last         Cmd
		observations *CmdObservations
		invoked      bool
		count        int
	)
	var threadID uint64
	return func(ctx context.Context, in atom_pb.Atom) error {
		count++
		if in == nil {
			if last != nil {
				handler(last)
			}
			last = nil
			return nil
		}

		if in, ok := in.(*core_pb.SwitchThread); ok {
			threadID = in.ThreadID
			return nil
		}

		out, err := protoconv.ToObject(ctx, in)
		if err != nil {
			return err
		}
		switch out := out.(type) {
		case Cmd:
			if last != nil {
				handler(last)
			}
			last = out
			invoked = false
			observations = nil
			out.SetThread(threadID)

		case CmdObservation:
			if observations == nil {
				observations = &CmdObservations{}
				e := last.Extras()
				if e == nil {
					return log.Errf(ctx, nil, "Not allowed extras %T:%v", last, last)
				}
				*e = append(*e, observations)
			}
			if !invoked {
				observations.Reads = append(observations.Reads, out)
			} else {
				observations.Writes = append(observations.Writes, out)
			}
		case *invokeMarker:
			invoked = true
		case CmdExtra:
			e := last.Extras()
			if e == nil {
				return log.Errf(ctx, nil, "Not allowed extras %T:%v", last, last)
			}
			*e = append(*e, out)
		default:
			return log.Errf(ctx, nil, "Unhandled type during conversion %T:%v", out, out)
		}
		return nil
	}
}

// CmdToProto returns a function that converts all the commands it is handed,
// passing the generated protos to the handler.
func CmdToProto(handler func(a atom_pb.Atom)) func(context.Context, Cmd) error {
	var threadID uint64
	return func(ctx context.Context, in Cmd) error {
		if in.Thread() != threadID {
			threadID = in.Thread()
			handler(&core_pb.SwitchThread{ThreadID: threadID})
		}
		out, err := protoconv.ToProto(ctx, in)
		if err != nil {
			return err
		}
		handler(out)

		for _, e := range in.Extras().All() {
			switch e := e.(type) {
			case *CmdObservations:
				for _, o := range e.Reads {
					p, err := protoconv.ToProto(ctx, o)
					if err != nil {
						return err
					}
					handler(p)
				}
				handler(atom_pb.InvokeMarker)
				for _, o := range e.Writes {
					p, err := protoconv.ToProto(ctx, o)
					if err != nil {
						return err
					}
					handler(p)
				}
			default:
				p, err := protoconv.ToProto(ctx, e)
				if err != nil {
					return err
				}
				handler(p)
			}
		}

		return nil
	}
}

type invokeMarker struct{}

func init() {
	protoconv.Register(
		func(ctx context.Context, a *invokeMarker) (*atom_pb.Invoke, error) {
			return &atom_pb.Invoke{}, nil
		},
		func(ctx context.Context, a *atom_pb.Invoke) (*invokeMarker, error) {
			return &invokeMarker{}, nil
		},
	)
}
