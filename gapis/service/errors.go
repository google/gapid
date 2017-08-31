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

package service

import "fmt"

func (e *ErrDataUnavailable) Error() string {
	return fmt.Sprintf("The requested data is unavailable. Reason: %v", e.Reason.Text(nil))
}

func (e *ErrInvalidPath) Error() string {
	return fmt.Sprintf("The path '%v' is invalid. Reason: %v", e.Path, e.Reason.Text(nil))
}

func (e *ErrInvalidArgument) Error() string {
	return fmt.Sprintf("The argument is invalid. Reason: %v", e.Reason.Text(nil))
}

func (e *ErrPathNotFollowable) Error() string {
	return "The path is not followable"
}

func (e *ErrInternal) Error() string {
	return fmt.Sprintf("Internal error: %s", e.Message)
}

func (e *ErrUnsupportedVersion) Error() string {
	return fmt.Sprintf("Unsupported version: %v", e.Reason.Text(nil))
}
