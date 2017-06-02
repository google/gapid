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
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/gapid/test/robot/stash"
)

var entitiesPathPattern = regexp.MustCompile("^/entities/([a-fA-F0-9]+)/?$")

func simplifyName(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' {
			return r
		} else {
			return '_'
		}
	}, s)
}

func entityId(path string) (string, error) {
	m := entitiesPathPattern.FindStringSubmatch(path)
	if len(m) != 2 {
		return "", fmt.Errorf("Request not supported: %s", path)
	}
	return m[1], nil
}

func (s *Server) handleEntities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var (
		err     error
		code    int
		id      string
		e       *stash.Entity
		rs      io.ReadSeeker
		name    string
		modtime time.Time
	)

	defer func() {
		if err != nil {
			w.WriteHeader(code)
			fmt.Fprintf(w, "error: %s", err.Error())
		}
	}()

	if id, err = entityId(r.URL.Path); err != nil {
		code = 400
		return
	}
	if e, err = s.Stash.Lookup(ctx, id); err != nil {
		code = 404
		return
	}
	if e.GetStatus() != stash.Status_Present {
		code, err = 404, fmt.Errorf("Entity status %s: %s", e.GetStatus().String(), id)
		return
	}
	if rs, err = s.Stash.Open(ctx, id); err != nil {
		code = 500
		return
	}

	modtime = time.Unix(e.GetTimestamp().Seconds, int64(e.GetTimestamp().Nanos))

	types := e.GetUpload().GetType()
	if len(types) == 0 || types[0] == "" {
		types = []string{"application/octet-stream"}
	}
	w.Header().Set("Content-Type", types[0])

	names := e.GetUpload().GetName()
	if len(names) > 0 {
		name = simplifyName(names[0])
		w.Header().Set("Content-Disposition", "; filename="+name)
	}

	http.ServeContent(w, r, name, modtime, rs)
}
