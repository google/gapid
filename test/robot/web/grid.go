// Copyright (C) 2018 Google Inc.
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
	"errors"
	"net/http"

	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/replay"
	"github.com/google/gapid/test/robot/report"
	q "github.com/google/gapid/test/robot/search/query"
	"github.com/google/gapid/test/robot/trace"
)

type cell struct {
	Scheduled int
	Running   int
	Succeeded int
	Failed    int
}

type row struct {
	ID    string
	Cells []cell
}

type grid struct {
	Columns []string
	Rows    []row
}

func (s *Server) handleGrid(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pkg := r.FormValue("pkg")
	if pkg == "" {
		writeError(w, 404, errors.New("The pkg parameter is required"))
		return
	}
	pkgQuery := q.Name("Input").Member("Package").Equal(q.String(pkg)).Query()

	rows := make([]row, 0)
	rowByID := make(map[string]*row)
	update := func(rid string, cidx int, status job.Status) {
		r := rowByID[rid]
		if r == nil {
			rows = append(rows, row{
				ID:    rid,
				Cells: make([]cell, 3),
			})
			r = &rows[len(rows)-1]
			rowByID[rid] = r
		}

		switch status {
		case job.Status_Running:
			r.Cells[cidx].Running++
		case job.Status_Succeeded:
			r.Cells[cidx].Succeeded++
		case job.Status_Failed:
			r.Cells[cidx].Failed++
		default:
			r.Cells[cidx].Scheduled++
		}
	}

	traceToSubj := make(map[string]string)
	if err := s.Trace.Search(ctx, pkgQuery, func(ctx context.Context, entry *trace.Action) error {
		update(entry.Input.Subject, 0, entry.Status)
		if entry.Output != nil {
			traceToSubj[entry.Output.Trace] = entry.Input.Subject
		}
		return nil
	}); err != nil {
		writeError(w, 500, err)
		return
	}

	if err := s.Report.Search(ctx, pkgQuery, func(ctx context.Context, entry *report.Action) error {
		if subj, ok := traceToSubj[entry.Input.Trace]; ok {
			update(subj, 1, entry.Status)
		}
		return nil
	}); err != nil {
		writeError(w, 500, err)
		return
	}

	if err := s.Replay.Search(ctx, pkgQuery, func(ctx context.Context, entry *replay.Action) error {
		if subj, ok := traceToSubj[entry.Input.Trace]; ok {
			update(subj, 2, entry.Status)
		}
		return nil
	}); err != nil {
		writeError(w, 500, err)
		return
	}

	result := grid{
		Columns: []string{"trace", "report", "replay"},
		Rows:    rows,
	}

	json.NewEncoder(w).Encode(result)
}
