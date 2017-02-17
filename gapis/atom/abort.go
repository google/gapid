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

package atom

import "fmt"

// ErrAborted is the error returned by Atom.Mutate() when the execution of the
// state mutator was terminated by an API call to the abort() intrinsic.
type ErrAborted string

func (e ErrAborted) Error() string {
	return fmt.Sprintf("aborted(%s)", string(e))
}

func IsAbortedError(err error) bool {
	_, ok := err.(ErrAborted)
	return ok
}
