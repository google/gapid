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

func (e ErrCmdAborted) Error() string {
	return fmt.Sprintf("aborted(%s)", e.Reason)
}

// Abort retuns a new ErrCmdAborted with the given error message.
func Abort(msg string) *ErrCmdAborted {
	return &ErrCmdAborted{Reason: msg}
}

// IsErrCmdAborted returns true if the cause of the error err was due to an
// abort() in the command.
func IsErrCmdAborted(err error) bool {
	_, ok := errors.Cause(err).(*ErrCmdAborted)
	return ok
}
