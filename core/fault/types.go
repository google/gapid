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

package fault

// Const is the type for constant error values.
type Const string

// Error implements error for Const returning the string value of the const.
func (e Const) Error() string { return string(e) }

// InvalidErrorType is the error returned by From when the type is not an error.
const InvalidErrorType = Const("Invalid type for error")

// From converts from any value to an error safely.
// If the value is a nil, an untyped nil is returned.
// If the value is not nil, but does not implement error, InvalidErrorType
// is returned.
func From(value interface{}) error {
	switch err := value.(type) {
	case nil:
		return nil
	case error:
		return err
	default:
		return InvalidErrorType
	}
}
