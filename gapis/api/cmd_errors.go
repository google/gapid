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

package api

import (
	"fmt"

	"github.com/pkg/errors"
)

// ErrCmdAborted is the error returned by Cmd.Mutate() when the execution was
// terminated by the abort() intrinsic.
type ErrCmdAborted string

func (e ErrCmdAborted) Error() string {
	return fmt.Sprintf("aborted(%s)", string(e))
}

// IsErrCmdAborted returns true if the cause of the error err was due to an
// abort() in the command.
func IsErrCmdAborted(err error) bool {
	_, ok := errors.Cause(err).(ErrCmdAborted)
	return ok
}
