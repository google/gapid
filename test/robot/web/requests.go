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
	"fmt"
	"net/http"

	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/search/script"
)


func query(w http.ResponseWriter, r *http.Request) (*search.Query, error) {
	builder, err := script.Parse(r.Context(), r.FormValue("q"))
	if err != nil {
		return nil, writeError(w, 400, err)
	}
	return builder.Query(), nil
}

func writeError(w http.ResponseWriter, code int, err error) error {
	w.WriteHeader(code)
	fmt.Fprintf(w, "Error processing request: %v", err)
	return err
}
