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

package cyclic

import (
	"fmt"

	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/registry"
	"github.com/google/gapid/framework/binary/schema"
)

// Encoder creates a binary.Encoder that writes to the supplied pod.Writer.
func Encoder(writer pod.Writer) binary.Encoder {
	return &encoder{
		Writer:   writer,
		entities: map[*binary.Entity]uint32{nil: 0},
		objects:  map[binary.Object]uint32{nil: 0},
	}
}

// Decoder creates a binary.Decoder that reads from the provided pod.Reader.
func Decoder(reader pod.Reader) *decoder {
	return &decoder{
		Reader:       reader,
		Namespace:    registry.Global,
		AllowDynamic: true,
		entities:     map[uint32]*binary.Entity{0: nil},
		objects:      map[uint32]binary.Object{0: nil},
		substack:     substack{},
	}
}

type encoder struct {
	pod.Writer
	entities      map[*binary.Entity]uint32
	objects       map[binary.Object]uint32
	controlNeeded bool
	control       Control
}

type decoder struct {
	pod.Reader
	Namespace    *registry.Namespace
	AllowDynamic bool
	entities     map[uint32]*binary.Entity
	objects      map[uint32]binary.Object
	substack     substack
	control      Control
}

// writeSid write out the stream id and whether the data follows
// It may also trigger a control block to be written
// You are not allowed to call this function with a sid of 0 and encoded set to true
func (e *encoder) writeSid(sid uint32, encoded bool) {
	if e.controlNeeded {
		e.controlNeeded = false // set early to prevent recursive triggering
		e.Uint32(1)             // encoded sid 0 is a special marker
		e.control.write(e)      // and write the control block itself
	}
	v := sid << 1
	if encoded {
		v |= 1
	}
	e.Uint32(v)
}

// readSid returns the stream id and whether the data follows
// If it returns a sid of zero, encoded will always be false.
func (d *decoder) readSid() (uint32, bool) {
	v := d.Uint32()
	if v == 1 { // encoded sis 0 is a special marker
		// read control block.
		d.control.read(d)
		// read the real sid
		v = d.Uint32()
	}
	encoded := (v & 1) != 0
	sid := v >> 1
	return sid, encoded
}

func (e *encoder) Entity(s *binary.Entity) {
	if sid, found := e.entities[s]; found {
		e.writeSid(sid, false)
	} else {
		sid = uint32(len(e.entities))
		e.entities[s] = sid
		e.writeSid(sid, true)
		schema.EncodeEntity(e, s)
	}
}

func (d *decoder) Entity() *binary.Entity {
	sid, encoded := d.readSid()
	if encoded {
		s := &binary.Entity{}
		d.entities[sid] = s
		schema.DecodeEntity(d, s)
		return s
	}
	s, found := d.entities[sid]
	if !found {
		d.SetError(fmt.Errorf("Unknown entity sid %v", sid))
	}
	return s
}

func (e *encoder) Simple(o pod.Writable) {
	if obj, ok := o.(binary.Object); ok {
		e.Struct(obj)
	} else {
		o.WriteSimple(e)
	}
}

func (e *encoder) Struct(obj binary.Object) {
	obj.Class().Encode(e, obj)
}

func (d *decoder) doStruct(t binary.SubspaceType, obj binary.Object) {
	entity := d.substack.pushExpectStruct(t)
	if entity == nil {
		d.SetError(
			fmt.Errorf("Struct() decoder expected %T %v got %s %v. Substack:\n%v",
				obj, obj.Class().Schema().Identity, t, t, d.substack.String()))
	} else {
		if u := d.Lookup(entity); u == nil {
			d.SetError(fmt.Errorf("Unknown type %v signature %q", t, entity.Signature()))
		} else {
			u.DecodeTo(d, obj)
		}
	}
}

func (d *decoder) Simple(o pod.Readable) {
	if obj, ok := o.(binary.Object); ok {
		d.Struct(obj)
	} else {
		o.ReadSimple(d)
	}
}

func (d *decoder) Struct(obj binary.Object) {
	if t, err := d.substack.popType(); err != nil {
		d.SetError(err)
	} else {
		d.doStruct(t, obj)
	}
}

func (e *encoder) Variant(obj binary.Object) {
	if obj == nil {
		e.Entity(nil)
		return
	}
	class := obj.Class()
	e.Entity(class.Schema())
	class.Encode(e, obj)
}

func (d *decoder) Variant() binary.Object {
	entity := d.Entity()
	if entity == nil {
		return nil
	}
	if u := d.Lookup(entity); u == nil {
		d.SetError(fmt.Errorf("Unknown type %q", entity.Signature()))
		return nil
	} else {
		d.substack.pushStruct(entity)
		o := u.New()
		u.DecodeTo(d, o)
		return o
	}
}

func (e *encoder) Object(obj binary.Object) {
	if sid, found := e.objects[obj]; found {
		e.writeSid(sid, false)
	} else {
		sid = uint32(len(e.objects))
		e.objects[obj] = sid
		e.writeSid(sid, true)
		e.Variant(obj)
	}
}

func (d *decoder) Object() binary.Object {
	sid, decode := d.readSid()
	o, found := d.objects[sid]
	if decode {
		if found {
			d.SetError(fmt.Errorf("Object sid %v occured more than once", sid))
		}
		o = d.Variant()
		d.objects[sid] = o
	} else if !found {
		d.SetError(fmt.Errorf("Unknown object sid %v", sid))
	}
	return o
}

func (d *decoder) Lookup(entity *binary.Entity) binary.UpgradeDecoder {
	u := d.Namespace.LookupUpgrader(entity.Signature())
	if u == nil && d.AllowDynamic {
		u = (*schema.ObjectClass)(entity)
	}
	return u
}

func (d *decoder) Count() uint32 {
	count := d.Uint32()
	if err := d.substack.pushCount(count); err != nil {
		d.SetError(err)
	}
	return count
}

func (e *encoder) GetMode() binary.Mode {
	return e.control.Mode
}

func (e *encoder) SetMode(mode binary.Mode) {
	if e.control.Mode != mode {
		e.controlNeeded = true
		e.control.Mode = mode
	}
}

func (d *decoder) GetMode() binary.Mode {
	return d.control.Mode
}
