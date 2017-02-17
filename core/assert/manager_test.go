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
	info  bytes.Buffer
}

func (f *fakeT) Fatal(args ...interface{}) {
	fmt.Fprintln(&f.fatal, args...)
}
func (f *fakeT) Error(args ...interface{}) {
	fmt.Fprintln(&f.error, args...)
}
func (f *fakeT) Log(args ...interface{}) {
	fmt.Fprintln(&f.info, args...)
}

func Testmanager(t *testing.T) {
	const (
		expectInfo  = "Info:log to info\n"
		expectError = "Error:log to error\n"
		expectFatal = "Fatal:log to fatal\n"
	)
	fake := &fakeT{}
	assert := assert.To(fake).For("")
	assert.Log("log to info")
	assert.Error("log to error")
	assert.Fatal("log to fatal")
	if fake.info.String() != expectInfo {
		t.Errorf("For info got %q expected %q", fake.info.String(), expectInfo)
	}
	if fake.error.String() != expectError {
		t.Errorf("For error got %q expected %q", fake.error.String(), expectError)
	}
	if fake.fatal.String() != expectFatal {
		t.Errorf("For fatal got %q expected %q", fake.fatal.String(), expectFatal)
	}
}
