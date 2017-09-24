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

package analytics

import "fmt"

// OnError is called whenever there was a problem sending analytics.
// By default errors are simply ignored.
var OnError = func(err error) {}

// ErrPayloadTooLarge is the error returned when attempting to send a
// payload that is too large to send.
type ErrPayloadTooLarge struct {
	Payload Payload
	Size    int
}

func (e ErrPayloadTooLarge) Error() string {
	return fmt.Sprintf("Payload too large. Got: %v, max: %v", int(e.Size), maxHitSize)
}
