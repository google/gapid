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
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api/core/core_pb"
)

// CmdWithResult is the optional interface implemented by commands that have
// a result value.
type CmdWithResult interface {
	Cmd

	// GetResult returns the result value for this command.
	GetResult() proto.Message

	// SetResult changes the result value.
	SetResult(proto.Message) error
}

func cmdCallFor(cmd Cmd) proto.Message {
	if cmd, ok := cmd.(CmdWithResult); ok {
		return cmd.GetResult()
	}
	return &CmdCall{}
}

// ProtoToCmd returns a function that converts all the storage commands it is
// handed, passing the generated live commands to the handler.
// You must call this with a nil to flush the final command.
func ProtoToCmd(handler func(Cmd) error) func(context.Context, proto.Message) error {
	var (
		last    Cmd
		invoked bool
	)
	var threadID uint64
	return func(ctx context.Context, in proto.Message) error {
		if in == nil {
			if last != nil {
				if err := handler(last); err != nil {
					return err
				}
			}
			last = nil
			return nil
		}

		if cwr, ok := last.(CmdWithResult); ok {
			if cwr.SetResult(in) == nil {
				invoked = true
				return nil
			}
		}

		if in, ok := in.(*core_pb.SwitchThread); ok {
			threadID = in.ThreadID
			return nil
		}

		out, err := protoconv.ToObject(ctx, in)
		if e, ok := err.(protoconv.ErrNoConverterRegistered); ok && e.Object == in {
			out, err = in, nil // No registered converter. Treat proto as the object.
		}
		if err != nil {
			return err
		}
		switch out := out.(type) {
		case Cmd:
			if last != nil {
				if err := handler(last); err != nil {
					return err
				}
			}
			last = out
			invoked = false
			out.SetThread(threadID)

		case *CmdCall:
			invoked = true
			return nil

		case CmdObservation:
			if last == nil {
				return fmt.Errorf("Got observation without a command")
			}
			observations := last.Extras().GetOrAppendObservations()
			if !invoked {
				observations.Reads = append(observations.Reads, out)
			} else {
				observations.Writes = append(observations.Writes, out)
			}
		case CmdExtra:
			if last == nil {
				log.W(ctx, "Got %T before first command. Ignoring", out)
				return nil
			}
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
func CmdToProto(handler func(a proto.Message) error) func(context.Context, Cmd) error {
	var threadID uint64
	return func(ctx context.Context, in Cmd) error {
		if in.Thread() != threadID {
			threadID = in.Thread()
			if err := handler(&core_pb.SwitchThread{ThreadID: threadID}); err != nil {
				return err
			}
		}
		out, err := protoconv.ToProto(ctx, in)
		if err != nil {
			return err
		}
		if err := handler(out); err != nil {
			return err
		}

		handledCall := false

		for _, e := range in.Extras().All() {
			switch e := e.(type) {
			case *CmdObservations:
				for _, o := range e.Reads {
					p, err := protoconv.ToProto(ctx, o)
					if err != nil {
						return err
					}
					if err := handler(p); err != nil {
						return err
					}
				}
				if err := handler(cmdCallFor(in)); err != nil {
					return err
				}
				handledCall = true
				for _, o := range e.Writes {
					p, err := protoconv.ToProto(ctx, o)
					if err != nil {
						return err
					}
					if err := handler(p); err != nil {
						return err
					}
				}
			default:
				p, err := protoconv.ToProto(ctx, e)
				if err != nil {
					return err
				}
				if err := handler(p); err != nil {
					return err
				}
			}
		}

		if !handledCall {
			if err := handler(cmdCallFor(in)); err != nil {
				return err
			}
		}

		return nil
	}
}
