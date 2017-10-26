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
	"strings"

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

	return pack.Read(ctx, r, unpacker{verb.Verbose, map[uint64]int{}}, true)
}

type unpacker struct {
	Verbose bool
	DepthOf map[uint64]int
}

func (u unpacker) BeginGroup(ctx context.Context, msg proto.Message, id uint64) error {
	depth := 0
	u.DepthOf[id] = depth
	log.I(ctx, "%sBeginGroup(msg: %v, id: %v)", indent(depth), u.msgString(msg), id)
	return nil
}
func (u unpacker) BeginChildGroup(ctx context.Context, msg proto.Message, id, parentID uint64) error {
	depth := u.DepthOf[parentID] + 1
	u.DepthOf[id] = depth
	log.I(ctx, "%sBeginChildGroup(msg: %v, id: %v, parentID: %v)", indent(depth), u.msgString(msg), id, parentID)
	return nil
}
func (u unpacker) EndGroup(ctx context.Context, id uint64) error {
	depth := u.DepthOf[id]
	delete(u.DepthOf, id)
	log.I(ctx, "%sEndGroup(id: %v)", indent(depth), id)
	return nil
}
func (u unpacker) Object(ctx context.Context, msg proto.Message) error {
	depth := 0
	log.I(ctx, "%sObject(msg: %v)", indent(depth), u.msgString(msg))
	return nil
}
func (u unpacker) ChildObject(ctx context.Context, msg proto.Message, parentID uint64) error {
	depth := u.DepthOf[parentID] + 1
	log.I(ctx, "%sChildObject(msg: %v, parentID: %v)", indent(depth), u.msgString(msg), parentID)
	return nil
}

func indent(depth int) string {
	return strings.Repeat("  ", depth)
}

func (u *unpacker) msgString(msg proto.Message) string {
	var str string
	switch msg := msg.(type) {
	case *pack.Dynamic:
		str = fmt.Sprintf("%+v", msg)
	default:
		str = fmt.Sprintf("%T{%+v}", msg, msg)
	}
	if len(str) > 100 && !u.Verbose {
		str = str[:97] + "..." // TODO: Consider unicode.
	}
	return str
}
