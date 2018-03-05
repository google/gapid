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

package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/google/gapid/core/os/shell"
)

var exe string

// Git is a go-wrapper for the git version control system.
type Git struct {
	exe string
	wd  string
}

// New returns a new Git instance targeting the working directory wd.
func New(wd string) (Git, error) {
	if exe == "" {
		path, err := exec.LookPath("git")
		if err != nil {
			return Git{}, fmt.Errorf("Couldn't find git: %v", err)
		}
		exe = path
	}

	return Git{
		wd:  wd,
		exe: exe,
	}, nil
}

func (g Git) run(ctx context.Context, args ...interface{}) (string, string, error) {
	return g.runWithStdin(ctx, nil, args...)
}

func (g Git) runWithStdin(ctx context.Context, stdin io.Reader, args ...interface{}) (string, string, error) {
	argstrs := make([]string, len(args))
	for i := range args {
		argstrs[i] = fmt.Sprint(args[i])
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := shell.Command("git", argstrs...).In(g.wd).Read(stdin).Capture(stdout, stderr)
	err := cmd.Run(ctx)
	if err != nil {
		err = fmt.Errorf("%v\n%v", err, stderr.String())
	}
	return stdout.String(), stderr.String(), err
}
