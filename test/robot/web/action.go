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

package web

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/gapid/test/robot/replay"
	"github.com/google/gapid/test/robot/report"
	"github.com/google/gapid/test/robot/trace"
)

func (s *Server) handleReplays(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result := []*replay.Action{}

	if query, err := query(w, r); err == nil {
		if err = s.Replay.Search(ctx, query, func(ctx context.Context, entry *replay.Action) error {
			result = append(result, entry)
			return nil
		}); err != nil {
			writeError(w, 500, err)
			return
		}

		json.NewEncoder(w).Encode(result)
	}
}

func (s *Server) handleTraces(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result := []*trace.Action{}

	if query, err := query(w, r); err == nil {
		if err = s.Trace.Search(ctx, query, func(ctx context.Context, entry *trace.Action) error {
			result = append(result, entry)
			return nil
		}); err != nil {
			writeError(w, 500, err)
			return
		}

		json.NewEncoder(w).Encode(result)
	}
}

func (s *Server) handleReports(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result := []*report.Action{}

	if query, err := query(w, r); err == nil {
		if err = s.Report.Search(ctx, query, func(ctx context.Context, entry *report.Action) error {
			result = append(result, entry)
			return nil
		}); err != nil {
			writeError(w, 500, err)
			return
		}

		json.NewEncoder(w).Encode(result)
	}
}
