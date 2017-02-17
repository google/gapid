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

package service

import (
	"bytes"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/cyclic"
	"github.com/google/gapid/framework/binary/vle"
)

// Encode encodes a binary object to the Object's buffer.
func (o *Object) Encode(obj binary.Object) error {
	buf := bytes.Buffer{}
	e := cyclic.Encoder(vle.Writer(&buf))
	e.Object(obj)
	o.Data = buf.Bytes()
	return e.Error()
}

// Decode decodes a binary object from the Object's buffer.
func (o *Object) Decode() (binary.Object, error) {
	d := cyclic.Decoder(vle.Reader(bytes.NewReader(o.Data)))
	obj := d.Object()
	if err := d.Error(); err != nil {
		return nil, err
	}
	return obj, nil
}
