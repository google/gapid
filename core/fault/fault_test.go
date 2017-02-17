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

package fault_test

import (
	"fmt"
	"testing"

	"github.com/google/gapid/core/fault"
)

const (
	errorMessage = "Some message"
	anError      = fault.Const(errorMessage)
	anotherError = fault.Const("another")
)

func TestFrom(t *testing.T) {
	var err error
	err = fault.From(nil)
	if err != nil {
		t.Errorf("errors.From(nil) incorrectly returned a valid error")
	}
	err = fault.From(anError)
	if err != anError {
		t.Errorf("errors.From(anError) returned a different object")
	}
	err = fault.From(fmt.Errorf("Format %s", "error"))
	if err.Error() != "Format error" {
		t.Errorf("errors.From modified a formatted error type")
	}
	err = fault.From(0)
	if err != fault.InvalidErrorType {
		t.Errorf("errors.From of a non error type did not return InvalidErrorType")
	}
	if anError.Error() != errorMessage {
		t.Errorf("InvalidErrorType has the wrong string form, expected %q got %q", errorMessage, anError)
	}
}

func TestList(t *testing.T) {
	list := fault.List{}
	if list.First() != nil {
		t.Errorf("First on empty error list did not return nil")
	}
	list.Collect(anError)
	if len(list) != 1 {
		t.Errorf("Adding one error did not make the list length 1")
	}
	if list.First() != anError {
		t.Errorf("First did not return the first error, got %v", anError)
	}
	list.Collect(anotherError)
	if len(list) != 2 {
		t.Errorf("Adding a second error did not make the list length 2")
	}
	if list.First() != anError {
		t.Errorf("First did not return the first error, got %v", anError)
	}
}

func TestOne(t *testing.T) {
	one := fault.One{}
	if one.First() != nil {
		t.Errorf("First on empty error list did not return nil")
	}
	one.Collect(anError)
	if one.First() != anError {
		t.Errorf("First did not return the first error, got %v", anError)
	}
	one.Collect(anotherError)
	if one.First() != anError {
		t.Errorf("First did not return the first error, got %v", anError)
	}
}
