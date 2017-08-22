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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/data/pack"
	"github.com/google/gapid/core/log"
)

type unpackVerb struct{ UnpackFlags }

func init() {
	verb := &unpackVerb{}
	app.AddVerb(&app.Verb{
		Name:      "unpack",
		ShortHelp: "Displays the raw protos in a protopack file",
		Action:    verb,
	})
}

func (verb *unpackVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one protopack file expected, got %d", flags.NArg())
		return nil
	}

	filepath, err := filepath.Abs(flags.Arg(0))
	ctx = log.V{"filepath": filepath}.Bind(ctx)
	if err != nil {
		return log.Err(ctx, err, "Could not find capture file")
	}

	r, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer r.Close()

	return pack.Read(ctx, r, unpacker{})
}

type unpacker struct{}

func (unpacker) BeginGroup(ctx context.Context, msg proto.Message, id uint64) error {
	log.I(ctx, "BeginGroup(msg: %v, id: %v)", msgString(msg), id)
	return nil
}
func (unpacker) BeginChildGroup(ctx context.Context, msg proto.Message, id, parentID uint64) error {
	log.I(ctx, "BeginChildGroup(msg: %v, id: %v, parentID: %v)", msgString(msg), id, parentID)
	return nil
}
func (unpacker) EndGroup(ctx context.Context, id uint64) error {
	log.I(ctx, "EndGroup(id: %v)", id)
	return nil
}
func (unpacker) Object(ctx context.Context, msg proto.Message) error {
	log.I(ctx, "Object(msg: %v)", msgString(msg))
	return nil
}
func (unpacker) ChildObject(ctx context.Context, msg proto.Message, parentID uint64) error {
	log.I(ctx, "ChildObject(msg: %v, parentID: %v)", msgString(msg), parentID)
	return nil
}

func msgString(msg proto.Message) string {
	if msg, ok := msg.(*pack.Dynamic); ok {
		return fmt.Sprintf("%+v", msg)
	}
	return fmt.Sprintf("%T{%+v}", msg, msg)
}
