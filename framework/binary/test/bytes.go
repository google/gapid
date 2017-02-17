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

type Bytes struct {
	Data []byte
}

func (b Bytes) Add(data ...byte) Bytes {
	return Bytes{Data: append(b.Data, data...)}
}

func (d *Bytes) Read(b []byte) (int, error) {
	if len(d.Data) == 0 {
		return 0, ReadError
	}
	n := copy(b, d.Data)
	d.Data = d.Data[n:]
	return n, nil
}

type LimitedWriter struct{ Limit int }

func (d *LimitedWriter) Write(b []byte) (int, error) {
	if d.Limit <= 0 {
		return 0, WriteError
	}
	if d.Limit < len(b) {
		result := d.Limit
		d.Limit = 0
		return result, nil
	}
	return len(b), nil
}
