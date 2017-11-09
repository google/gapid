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

// +build crashreporting

// Package reporting implements a crash reporter to send GAPID crashes to a
// Google server.
package reporting

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/fault/stacktrace"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/host"
)

// TODO(baldwinn860): Send to production url when we get approval.
const (
	crashStagingURL = "https://clients2.google.com/cr/staging_report"
	crashURL        = crashStagingURL
)

// Enable turns on crash reporting if the running processes panics inside a
// crash.Go block.
func Enable(ctx context.Context, appName, appVersion string) {
	crash.Register(func(e interface{}, s stacktrace.Callstack) {
		var osName, osVersion string
		if h := host.Instance(ctx); h != nil {
			if os := h.GetConfiguration().GetOS(); os != nil {
				osName = os.GetName()
				osVersion = fmt.Sprintf("%v %v.%v.%v", os.GetBuild(), os.GetMajor(), os.GetMinor(), os.GetPoint())
			}
		}
		err := reporter{appName, appVersion, osName, osVersion}.report(s, crashURL)
		if err != nil {
			log.E(ctx, "%v", err)
		}
	})
}

type reporter struct {
	appName    string
	appVersion string
	osName     string
	osVersion  string
}

func (r reporter) report(s stacktrace.Callstack, endpoint string) error {
	stacktrace := s.String()

	url := fmt.Sprintf("%v?product=%v&version=%v", crashURL, url.QueryEscape(r.appName), url.QueryEscape(r.appVersion))

	body, contentType, err := encoder{
		appName:    r.appName,
		appVersion: r.appVersion,
		osName:     r.osName,
		osVersion:  r.osVersion,
		stacktrace: stacktrace,
	}.encode()
	if err != nil {
		return fmt.Errorf("Couldn't encode crash report: %v", err)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("Couldn't create new crash report request: %v", err)
	}

	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to upload report request: %v (%v)", err, res.StatusCode)
	}

	return nil
}
