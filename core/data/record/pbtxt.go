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
	"bufio"
	"io"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
)

// pbtxtFileType is an implementation of fileType that stores it's records in proto text format.
type pbtxtFileType struct{}

type pbtxtHandler struct {
	f    *os.File
	null proto.Message
}

type pbtxtReader struct {
	pending string
	s       *bufio.Scanner
	null    proto.Message
}

const (
	pbtxtSeparator     = "===================="
	pbtxtSeparatorLine = "\n " + pbtxtSeparator + "\n"
)

func (pbtxtFileType) Ext() string { return ".pbtxt" }

func (pbtxtFileType) Open(ctx log.Context, f *os.File, null interface{}) (LedgerInstance, error) {
	m, ok := null.(proto.Message)
	if !ok {
		return nil, cause.Explain(ctx, nil, "Cannot create proto text ledger with non proto type")
	}
	return &pbtxtHandler{f: f, null: m}, nil
}

func (h *pbtxtHandler) Write(ctx log.Context, record interface{}) error {
	if _, err := io.WriteString(h.f, proto.MarshalTextString(record.(proto.Message))); err != nil {
		return err
	}
	_, err := io.WriteString(h.f, pbtxtSeparatorLine)
	return err
}

func (h *pbtxtHandler) Reader(ctx log.Context) event.Source {
	return &pbtxtReader{s: bufio.NewScanner(&readAt{f: h.f}), null: h.null}
}

func (h *pbtxtHandler) Close(ctx log.Context) {
	h.f.Close()
}

func (h *pbtxtHandler) New(ctx log.Context) interface{} {
	return proto.Clone(h.null)
}

func (r *pbtxtReader) Next(ctx log.Context) interface{} {
	for r.s.Scan() {
		line := r.s.Text()
		if line == pbtxtSeparator {
			message := proto.Clone(r.null)
			err := proto.UnmarshalText(r.pending, message)
			if err != nil {
				jot.Fail(ctx, err, "Invalid text proto in ledger")
				return nil
			}
			r.pending = ""
			return message
		}
		r.pending += line
		r.pending += "\n"
	}
	return nil
}

func (h *pbtxtReader) Close(ctx log.Context) {}
