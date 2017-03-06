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

package test

import (
	"bytes"

	_ "github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
	_ "github.com/google/gapid/framework/binary/registry"
	_ "github.com/google/gapid/framework/binary/schema"
)

type TypeA struct {
	binary.Generate
	Data string
}

type TypeB struct {
	binary.Generate
	Data string
}

type TypeC struct {
	binary.Generate
	Data Simple
}

type BadType struct {
	binary.Generate `disable:"true"`
	Data            string
}

type Simple int8

func (s *Simple) ReadSimple(r pod.Reader) { *s = Simple(r.Uint8()) }
func (s Simple) WriteSimple(w pod.Writer) { w.Uint8(byte(s)) }

type Error string

func (d Error) Error() string { return string(d) }

var (
	ReadError   = Error("ReadError")
	WriteError  = Error("WriteError")
	SecondError = Error("SecondError")
)

var (
	ObjectA = &TypeA{Data: "ObjectA"}
	EntityA = []byte{
		0x04, 't', 'e', 's', 't', // Package
		0x05, 'T', 'y', 'p', 'e', 'A', // Identity
		0x00, // Version
		0x01, // field count
		0xb0, // primitive string
	}
	EntityAFull = []byte{
		0x04, 't', 'e', 's', 't', // Package
		0x05, 'T', 'y', 'p', 'e', 'A', // Identity
		0x00,                               // Version
		0x00,                               // Display
		0x01,                               // field count
		0xb0,                               // primitive string
		0x06, 's', 't', 'r', 'i', 'n', 'g', // Name
		0x04, 'D', 'a', 't', 'a', // Declared
		0x00, // metadata count
	}
	ObjectB = &TypeB{Data: "ObjectB"}
	EntityB = []byte{
		0x04, 't', 'e', 's', 't', // Package
		0x05, 'T', 'y', 'p', 'e', 'B', // Identity
		0x00, // Version
		0x01, // field count
		0xb0, // primitive string
	}
	ObjectC = &TypeC{Data: Simple(3)}
	EntityC = []byte{
		0x04, 't', 'e', 's', 't', // Package
		0x05, 'T', 'y', 'p', 'e', 'C', // Identity
		0x00, // Version
		0x01, // field count
		0x10, // primitive string
	}
	BadObject = &BadType{Data: "BadObject"}
)

type Entry struct {
	Name   string
	Values []binary.Object
	Data   []byte
}

func VerifyData(ctx log.Context, entry Entry, got *bytes.Buffer) {
	if !bytes.Equal(entry.Data, got.Bytes()) {
		ctx.Error().Logf(`%v gave unexpected bytes.
Expected: %# x
Got:      %# x`, entry.Name, entry.Data, got.Bytes())
	}
}
