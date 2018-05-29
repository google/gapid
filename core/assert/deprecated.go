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

package assert

import (
	"context"
)

// *************************************************************
// ** This file contains legacy shims only, it should be removed
// ** once all existing tests are converted
// *************************************************************

// Deprecated: For(m) is deprecated, use assert.To() instead.
func For(t interface{}, msg string, args ...interface{}) *Assertion {
	switch t := t.(type) {
	case Manager:
		return t.For(msg, args...)
	case context.Context:
		return To(logOutput{t}).For(msg, args...)
	default:
		panic("Not a valid assertion manager source")
	}
}
