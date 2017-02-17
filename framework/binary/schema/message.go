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

package schema

import (
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/registry"
)

// Message is used to send schema data to a client.
// It is not a normal binary object because it requires custom encoding for the Entity objects.
type Message struct {
	Entities  []*binary.Entity // The schema entities
	Constants []ConstantSet    // The constant values
}

var Namespace = registry.NewNamespace()

func init() {
	registry.Global.AddFallbacks(Namespace)
	Namespace.Add((*Message)(nil).Class())
}

var (
	schemaMessage = &binary.Entity{
		Package:  "schema",
		Identity: "Message",
		Fields:   []binary.Field{},
	}
)

type binaryClassMessage struct{}

func (*Message) Class() binary.Class               { return (*binaryClassMessage)(nil) }
func (*binaryClassMessage) New() binary.Object     { return &Message{} }
func (*binaryClassMessage) Schema() *binary.Entity { return schemaMessage }
func (*binaryClassMessage) Encode(e binary.Encoder, obj binary.Object) {
	oldMode := e.GetMode()
	e.SetMode(binary.Full)
	defer e.SetMode(oldMode)
	m := obj.(*Message)
	e.Uint32(uint32(len(m.Entities)))
	for _, entity := range m.Entities {
		e.Entity(entity)
	}
	e.Uint32(uint32(len(m.Constants)))
	for i := range m.Constants {
		EncodeConstants(e, &m.Constants[i])
	}
}

func doDecodeMessage(d binary.Decoder, m *Message) {
	m.Entities = make([]*binary.Entity, d.Uint32())
	for i := range m.Entities {
		m.Entities[i] = d.Entity()
	}
	m.Constants = make([]ConstantSet, d.Uint32())
	for i := range m.Constants {
		DecodeConstants(d, &m.Constants[i])
	}
}

func (b *binaryClassMessage) Decode(d binary.Decoder) binary.Object {
	m := &Message{}
	doDecodeMessage(d, m)
	return m
}
func (*binaryClassMessage) DecodeTo(d binary.Decoder, obj binary.Object) {
	doDecodeMessage(d, obj.(*Message))
}
