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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
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

	inpath, err := filepath.Abs(flags.Arg(0))
	ctx = log.V{"filepath": inpath}.Bind(ctx)
	if err != nil {
		return log.Err(ctx, err, "Could not find capture file")
	}

	r, err := os.Open(inpath)
	if err != nil {
		return err
	}
	defer r.Close()

	var ev pack.Events
	if len(verb.Dump) > 0 {
		ids := make(map[uint64]bool, len(verb.Dump))
		for _, id := range verb.Dump {
			ids[id] = true
		}

		out := verb.Out
		if out == "" {
			out = "."
		}
		out, err := filepath.Abs(out)
		ctx = log.V{"out": out}.Bind(ctx)
		if err != nil {
			return log.Err(ctx, err, "Could not find output directory")
		}

		ev = &dumper{ids, verb.Children, out, 0, 0}
	} else {
		ev = &printer{verb.Verbose, map[uint64]int{}}
	}
	return pack.Read(ctx, r, ev, true)
}

type printer struct {
	Verbose bool
	DepthOf map[uint64]int
}

func (u *printer) BeginGroup(ctx context.Context, msg proto.Message, id uint64) error {
	depth := 0
	u.DepthOf[id] = depth
	log.I(ctx, "%sBeginGroup(msg: %v, id: %v)", indent(depth), u.msgString(msg), id)
	return nil
}
func (u *printer) BeginChildGroup(ctx context.Context, msg proto.Message, id, parentID uint64) error {
	depth := u.DepthOf[parentID] + 1
	u.DepthOf[id] = depth
	log.I(ctx, "%sBeginChildGroup(msg: %v, id: %v, parentID: %v)", indent(depth), u.msgString(msg), id, parentID)
	return nil
}
func (u *printer) EndGroup(ctx context.Context, id uint64) error {
	depth := u.DepthOf[id]
	delete(u.DepthOf, id)
	log.I(ctx, "%sEndGroup(id: %v)", indent(depth), id)
	return nil
}
func (u *printer) Object(ctx context.Context, msg proto.Message) error {
	depth := 0
	log.I(ctx, "%sObject(msg: %v)", indent(depth), u.msgString(msg))
	return nil
}
func (u *printer) ChildObject(ctx context.Context, msg proto.Message, parentID uint64) error {
	depth := u.DepthOf[parentID] + 1
	log.I(ctx, "%sChildObject(msg: %v, parentID: %v)", indent(depth), u.msgString(msg), parentID)
	return nil
}

func indent(depth int) string {
	return strings.Repeat("  ", depth)
}

func (u *printer) msgString(msg proto.Message) string {
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

type dumper struct {
	ids      map[uint64]bool
	children bool
	dir      string
	found    int
	count    int
}

func (u *dumper) BeginGroup(ctx context.Context, msg proto.Message, id uint64) error {
	if _, ok := u.ids[id]; ok {
		u.found = 1
		path, err := u.dump(msg)
		log.I(ctx, "BeginGroup(id: %v) -> %s", id, path)
		return err
	}
	return nil
}
func (u *dumper) BeginChildGroup(ctx context.Context, msg proto.Message, id, parentID uint64) error {
	include := u.found > 0 && u.children
	if _, ok := u.ids[id]; ok {
		u.found++
		include = true
	}

	if include {
		path, err := u.dump(msg)
		log.I(ctx, "BeginChildGroup(id: %v, parentID: %v) -> %s", id, parentID, path)
		return err
	}
	return nil
}
func (u *dumper) EndGroup(ctx context.Context, id uint64) error {
	include := u.found > 0 && u.children
	if _, ok := u.ids[id]; ok {
		u.found--
		include = true
	}

	if include {
		log.I(ctx, "EndGroup(id: %v)", id)
	}
	return nil
}
func (u *dumper) Object(ctx context.Context, msg proto.Message) error {
	if u.found > 0 {
		path, err := u.dump(msg)
		log.I(ctx, "Object -> %s", path)
		return err
	}
	return nil
}
func (u *dumper) ChildObject(ctx context.Context, msg proto.Message, parentID uint64) error {
	if u.found > 0 && u.children {
		path, err := u.dump(msg)
		log.I(ctx, "ChildObject(parentID: %v) -> %s", parentID, path)
		return err
	}
	return nil
}

func (u *dumper) dump(msg proto.Message) (path string, err error) {
	u.count++
	path = filepath.Join(u.dir, fmt.Sprintf("unpack_%03d.json", u.count))

	data, err := json.Marshal(msg)
	if err == nil {
		err = ioutil.WriteFile(path, data, 0664)
	}
	return
}
