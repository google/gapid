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
	"strconv"

	"github.com/google/gapid/test/robot/monitor"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	seenStr := r.URL.Query().Get("seen")
	seen, err := strconv.ParseUint(seenStr, 10, 64)
	if err != nil {
		seen = 0
	}

	var gen *monitor.Generation
	s.o.Read(func(data *monitor.Data) {
		gen = data.Gen
	})

	seq := gen.WaitForUpdate(seen)
	result := map[string]interface{}{
		"seq": seq,
	}
	json.NewEncoder(w).Encode(result)
}
