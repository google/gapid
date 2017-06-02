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
	"context"
	"encoding/json"
	"os"
	"reflect"

	"github.com/google/gapid/core/event"
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

func (jsonFileType) Open(ctx context.Context, f *os.File, null interface{}) (LedgerInstance, error) {
	t := reflect.TypeOf(null).Elem()
	if _, err := json.Marshal(reflect.New(t)); err != nil {
		return nil, log.Err(ctx, nil, "Cannot create json ledger with non marshalable type")
	}
	return &jsonHandler{f: f, encoder: json.NewEncoder(f), null: t}, nil
}

func (h *jsonHandler) Write(ctx context.Context, record interface{}) error {
	return h.encoder.Encode(record)
}

func (h *jsonHandler) Reader(ctx context.Context) event.Source {
	return &jsonReader{decoder: json.NewDecoder(&readAt{f: h.f}), null: h.null}
}

func (h *jsonHandler) Close(ctx context.Context) {
	h.f.Close()
}

func (h *jsonHandler) New(ctx context.Context) interface{} {
	return reflect.New(h.null)
}

func (r *jsonReader) Next(ctx context.Context) interface{} {
	message := reflect.New(r.null)
	err := r.decoder.Decode(message)
	log.E(ctx, "Corrupt record file. Error: %v", err)
	return message
}

func (h *jsonReader) Close(ctx context.Context) {}
