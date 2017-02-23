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

package process

import (
	"fmt"
	"net"
	"regexp"

	"strconv"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/shell"
)

var portPattern = regexp.MustCompile(`^Bound on port '(\d+)'$`)

type portWatcher struct {
	portChan chan<- string
	fragment string
	done     bool
}

func (w *portWatcher) Write(b []byte) (n int, err error) {
	if w.done {
		return len(b), nil
	}
	s := w.fragment + string(b)
	start := 0
	for i, c := range s {
		if c == '\n' || c == '\r' {
			line := s[start:i]
			start = i + 1
			match := portPattern.FindStringSubmatch(line)
			if match != nil {
				w.portChan <- match[1]
				close(w.portChan)
				w.done = true
				return len(b), nil
			}
		}
	}
	w.fragment = s[start:]
	return len(b), nil
}

func Start(ctx log.Context, name string, extraEnv map[string][]string, args ...string) (int, error) {

	// Append extra environment variable values
	env := shell.CloneEnv()
	for key, vals := range extraEnv {
		env.AddPathStart(key, vals...)
	}
	return StartWithEnv(ctx, name, env, args...)
}

func StartWithEnv(ctx log.Context, name string, env *shell.Env, args ...string) (int, error) {
	errChan := make(chan error, 1)
	portChan := make(chan string, 1)
	stdout := &portWatcher{portChan: portChan}

	go func() {
		errChan <- shell.Command(name, args...).Verbose().Env(env).Capture(stdout, nil).Run(ctx)
	}()
	select {
	case port := <-portChan:
		return strconv.Atoi(port)
	case err := <-errChan:
		return 0, err
	}
}

func Connect(port int, authToken auth.Token) (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return nil, err
	}
	if err := auth.Write(conn, authToken); err != nil {
		return nil, err
	}
	return conn, err
}
