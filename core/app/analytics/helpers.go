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

//go:build analytics

package analytics

import (
	"fmt"
	"time"
)

// Stop is the function returned by SendTiming to end the timed region and send
// the result to the Google Analytics server (if enabled).
type Stop func(extras ...Payload)

// SendTiming starts a timing. The returned function will end the timing and
// send the result.
func SendTiming(category, name string) Stop {
	start := time.Now()
	return func(extras ...Payload) {
		Send(append(payloads{
			Timing{
				Category: category,
				Name:     name,
				Duration: time.Since(start),
			},
		}))
	}
}

// SendEvent sends the specific event to the Google Analytics server (if
// enabled).
func SendEvent(category, action, label string, extras ...Payload) {
	Send(append(payloads{
		Event{
			Category: category,
			Action:   action,
			Label:    label,
		},
	}, extras...))
}

// SendException sends the exception to the Google Analytics server (if
// enabled).
func SendException(description string, fatal bool, extras ...Payload) {
	Send(payloads{
		Exception{
			Description: description,
			Fatal:       fatal,
		},
	})
}

// SendBug sends an event indicating that a known bug has been hit.
func SendBug(id int, extras ...Payload) {
	Send(append(payloads{
		Event{
			Category: "bug",
			Action:   fmt.Sprint(id),
		},
	}, extras...))
}
