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

package jdbg

import "fmt"

// failure is the error type that can be thrown as a panic by JDbg.fail() and
// JDbg.err() and is caught by catchFailures(). It is used to immediately
// terminate execution of Do(), and return the error.
type failure struct {
	error
}

// fail panics with a failure error formed from msg and args, immediately
// terminating execution of Do().
func (j *JDbg) fail(msg string, args ...interface{}) {
	j.err(fmt.Errorf(msg, args...))
}

// err panics with a failure error holding err, immediately terminating
// execution of Do().
func (j *JDbg) err(err error) {
	panic(failure{err})
}

// Try calls f, catching and returning any exceptions thrown.
func Try(f func() error) (err error) {
	defer func() {
		switch r := recover().(type) {
		case nil:
		case failure:
			err = r
		default:
			panic(r)
		}
	}()
	err = f()
	return err
}
