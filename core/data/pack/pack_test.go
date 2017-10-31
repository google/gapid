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

package pack_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/pack"
	"github.com/google/gapid/core/data/protoutil/testprotos"
	"github.com/google/gapid/core/log"
)

type (
	eventBeginGroup struct {
		Msg proto.Message
		ID  *uint64
	}
	eventBeginChildGroup struct {
		Msg      proto.Message
		ID       *uint64
		ParentID *uint64
	}
	eventEndGroup struct {
		ParentID *uint64
	}
	eventObject struct {
		Msg proto.Message
	}
	eventChildObject struct {
		Msg      proto.Message
		ParentID *uint64
	}
)

func (e eventBeginGroup) write(ctx context.Context, w *pack.Writer) {
	id, err := w.BeginGroup(ctx, e.Msg)
	if assert.For(ctx, "BeginGroup").ThatError(err).Succeeded() {
		assert.For(ctx, "Set id").That(*e.ID).Equals(uint64(0)) // Isn't already set
		*e.ID = id
	}
}
func (e eventBeginChildGroup) write(ctx context.Context, w *pack.Writer) {
	assert.For(ctx, "Get parent id").That(*e.ParentID).NotEquals(0) // Was set
	id, err := w.BeginChildGroup(ctx, e.Msg, *e.ParentID)
	if assert.For(ctx, "BeginChildGroup").ThatError(err).Succeeded() {
		assert.For(ctx, "Set id").That(*e.ID).Equals(uint64(0)) // Isn't already set
		*e.ID = id
	}
}
func (e eventEndGroup) write(ctx context.Context, w *pack.Writer) {
	assert.For(ctx, "Get parent id").That(*e.ParentID).NotEquals(0) // Was set
	err := w.EndGroup(ctx, *e.ParentID)
	assert.For(ctx, "EndGroup").ThatError(err).Succeeded()
}
func (e eventObject) write(ctx context.Context, w *pack.Writer) {
	err := w.Object(ctx, e.Msg)
	assert.For(ctx, "Object").ThatError(err).Succeeded()
}
func (e eventChildObject) write(ctx context.Context, w *pack.Writer) {
	assert.For(ctx, "Get parent id").That(*e.ParentID).NotEquals(0) // Was set
	err := w.ChildObject(ctx, e.Msg, *e.ParentID)
	assert.For(ctx, "ChildObject").ThatError(err).Succeeded()
}

type event interface {
	write(ctx context.Context, w *pack.Writer)
}

type events []event

func (e *events) add(ev event) { *e = append(*e, ev) }

func (e *events) BeginGroup(ctx context.Context, msg proto.Message, id uint64) error {
	e.add(eventBeginGroup{msg, &id})
	return nil
}
func (e *events) BeginChildGroup(ctx context.Context, msg proto.Message, id, parentID uint64) error {
	e.add(eventBeginChildGroup{msg, &id, &parentID})
	return nil
}
func (e *events) EndGroup(ctx context.Context, id uint64) error {
	e.add(eventEndGroup{&id})
	return nil
}
func (e *events) Object(ctx context.Context, msg proto.Message) error {
	e.add(eventObject{msg})
	return nil
}
func (e *events) ChildObject(ctx context.Context, msg proto.Message, parentID uint64) error {
	e.add(eventChildObject{msg, &parentID})
	return nil
}

func TestReaderWriter(t *testing.T) {
	ctx := log.Testing(t)
	buf := &bytes.Buffer{}

	// Serialization Begin* methods return IDs for the written chunks.
	// Store them here so that *Child* methods can read them.
	var id0, id1, id2, id3 uint64

	expected := events{
		eventObject{&testprotos.MsgA{F32: 1, U32: 2, S32: 3, Str: "four"}},
		eventObject{&testprotos.MsgB{F64: 2, U64: 3, S64: 4, Bool: false}},
		eventObject{&testprotos.MsgA{F32: 3, U32: 4, S32: 5, Str: "six"}},
		eventObject{&testprotos.MsgB{F64: 4, U64: 5, S64: 6, Bool: true}},
		eventObject{&testprotos.MsgC{Entries: []*testprotos.MsgC_Entry{
			&testprotos.MsgC_Entry{Value: 1},
		}}},

		eventBeginGroup{&testprotos.MsgA{F32: 5, U32: 6, S32: 10, Str: "eleven"}, &id0},
		eventBeginGroup{&testprotos.MsgB{F64: 6, U64: 7, S64: 11, Bool: false}, &id1},
		eventChildObject{&testprotos.MsgA{F32: 7, U32: 8, S32: 12, Str: "thirteen"}, &id0},
		eventBeginChildGroup{&testprotos.MsgB{F64: 8, U64: 9, S64: 13, Bool: true}, &id2, &id0},
		eventEndGroup{&id0},
		eventBeginChildGroup{&testprotos.MsgA{F32: 9, U32: 10, S32: 11, Str: "twelve"}, &id3, &id1},
		eventEndGroup{&id1},
	}

	w, err := pack.NewWriter(buf)
	assert.For(ctx, "NewWriter").ThatError(err).Succeeded()
	for _, e := range expected {
		e.write(ctx, w)
	}

	got := events{}
	err = pack.Read(ctx, bytes.NewBuffer(buf.Bytes()), &got, false)
	assert.For(ctx, "Read").ThatError(err).Succeeded()

	assert.For(ctx, "events").ThatSlice(got).DeepEquals(expected)

	err = pack.Read(ctx, bytes.NewBuffer(buf.Bytes()), &got, true)
	assert.For(ctx, "Read (force-dynamic)").ThatError(err).Succeeded()
}
