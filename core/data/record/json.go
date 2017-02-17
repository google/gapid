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

package record

import (
	"encoding/json"
	"os"
	"reflect"

	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
)

// jsonFileType is an implementation of fileType that stores it's records in json format.
type jsonFileType struct{}

type jsonHandler struct {
	f       *os.File
	encoder *json.Encoder
	null    reflect.Type
}

type jsonReader struct {
	decoder *json.Decoder
	null    reflect.Type
}

func (jsonFileType) Ext() string { return ".json" }

func (jsonFileType) Open(ctx log.Context, f *os.File, null interface{}) (LedgerInstance, error) {
	t := reflect.TypeOf(null).Elem()
	if _, err := json.Marshal(reflect.New(t)); err != nil {
		return nil, cause.Explain(ctx, nil, "Cannot create json ledger with non marshalable type")
	}
	return &jsonHandler{f: f, encoder: json.NewEncoder(f), null: t}, nil
}

func (h *jsonHandler) Write(ctx log.Context, record interface{}) error {
	return h.encoder.Encode(record)
}

func (h *jsonHandler) Reader(ctx log.Context) event.Source {
	return &jsonReader{decoder: json.NewDecoder(&readAt{f: h.f}), null: h.null}
}

func (h *jsonHandler) Close(ctx log.Context) {
	h.f.Close()
}

func (h *jsonHandler) New(ctx log.Context) interface{} {
	return reflect.New(h.null)
}

func (r *jsonReader) Next(ctx log.Context) interface{} {
	message := reflect.New(r.null)
	err := r.decoder.Decode(message)
	jot.Fail(ctx, err, "Corrupt record file")
	return message
}

func (h *jsonReader) Close(ctx log.Context) {}
