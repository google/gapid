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
	"encoding/json"
	"net/http"

	"github.com/google/gapid/core/data/search/query"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/robot/master"
)

func (s *Server) handleSatellites(w http.ResponseWriter, r *http.Request) {
	ctx := log.Wrap(r.Context())
	result := []*master.Satellite(nil)
	s.Master.Search(ctx, query.Bool(true).Query(), func(ctx log.Context, entry *master.Satellite) error {
		result = append(result, entry)
		return nil
	})
	json.NewEncoder(w).Encode(result)
}
