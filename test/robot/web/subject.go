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

	"github.com/google/gapid/test/robot/search/query"
	"github.com/google/gapid/test/robot/subject"
)

func (s *Server) handleSubjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result := []*subject.Subject(nil)
	s.Subject.Search(ctx, query.Bool(true).Query(), func(ctx context.Context, entry *subject.Subject) error {
		result = append(result, entry)
		return nil
	})
	json.NewEncoder(w).Encode(result)
}
