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

package assert_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
)

type fakeT struct {
	fatal bytes.Buffer
	error bytes.Buffer
	log   bytes.Buffer
}

func (f *fakeT) Fatal(args ...interface{}) {
	fmt.Fprintln(&f.fatal, args...)
}
func (f *fakeT) Error(args ...interface{}) {
	fmt.Fprintln(&f.error, args...)
}
func (f *fakeT) Log(args ...interface{}) {
	fmt.Fprintln(&f.log, args...)
}

func Testmanager(t *testing.T) {
	const (
		expectLog   = "Info:manager test\n    log to info\n"
		expectError = "Error:manager test\n    log to error\n"
		expectFatal = "Critical:manager test\n    log to fatal\n"
	)
	fake := &fakeT{}
	assert.To(fake).For("manager test").Log("log to info")
	assert.To(fake).For("manager test").Error("log to error")
	assert.To(fake).For("manager test").Fatal("log to fatal")
	if fake.log.String() != expectLog {
		t.Errorf("For info got %q expected %q", fake.log.String(), expectLog)
	}
	if fake.error.String() != expectError {
		t.Errorf("For error got %q expected %q", fake.error.String(), expectError)
	}
	if fake.fatal.String() != expectFatal {
		t.Errorf("For fatal got %q expected %q", fake.fatal.String(), expectFatal)
	}
}
