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
	"bytes"
	"fmt"
	"net/url"
	"time"

	"github.com/google/gapid/core/app/analytics/param"
)

// Smaller than the real timeout, but there's no harm in being conservative.
const sessionTimeout = time.Minute * 10

type encoder func(p Payload) (*bytes.Buffer, error)

func newEncoder(clientID, hostOS, hostGPU string, version AppVersion) encoder {
	var lastSessionStart time.Time

	return func(p Payload) (*bytes.Buffer, error) {
		newSession := time.Since(lastSessionStart) > sessionTimeout
		if newSession {
			lastSessionStart = time.Now()
		}
		buf := bytes.Buffer{}
		add := func(key param.Parameter, val interface{}) {
			if buf.Len() > 0 {
				buf.WriteRune('&')
			}
			buf.WriteString(url.QueryEscape(string(key)))
			buf.WriteRune('=')
			buf.WriteString(url.QueryEscape(fmt.Sprint(val)))
		}

		add(param.ProtocolVersion, 1)
		add(param.ClientID, clientID)
		add(param.TrackingID, trackingID)
		add(param.ApplicationName, version.Name)
		add(param.ApplicationVersion, fmt.Sprintf("%v.%v.%v:%v", version.Major, version.Minor, version.Point, version.Build))

		if newSession {
			// We only need to send this data if we have a new session.
			// It doesn't matter what hit this is on.
			add(param.HostOS, hostOS)
			add(param.HostGPU, hostGPU)
		}

		p.values(add)

		if buf.Len() > maxHitSize {
			return nil, ErrPayloadTooLarge{p, buf.Len()}
		}

		return &buf, nil
	}
}
