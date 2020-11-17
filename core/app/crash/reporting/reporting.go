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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/fault/stacktrace"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/host"
)

const (
	crashStagingURL = "https://clients2.google.com/cr/staging_report"
	crashProdURL    = "https://clients2.google.com/cr/report"
	crashURL        = crashProdURL
)

var (
	mutex   sync.Mutex
	disable func()
)

// Enable turns on crash reporting if the running processes panics inside a
// crash.Go block.
func Enable(ctx context.Context, appName, appVersion string) {
	mutex.Lock()
	defer mutex.Unlock()
	if disable == nil {
		disable = crash.Register(func(e interface{}, s stacktrace.Callstack) {
			var osName, osVersion string
			if h := host.Instance(ctx); h != nil {
				if os := h.GetConfiguration().GetOS(); os != nil {
					osName = os.GetName()
					osVersion = fmt.Sprintf("%v %v.%v.%v", os.GetBuild(), os.GetMajorVersion(), os.GetMinorVersion(), os.GetPointVersion())
				}
			}
			res, err := Reporter{
				appName,
				appVersion,
				osName,
				osVersion,
			}.reportStacktrace(s, crashURL)
			if err != nil {
				log.E(ctx, "%v", err)
			} else if res != "" {
				log.I(ctx, "%v", res)
			}
		})
	}
}

// Disable turns off crash reporting previously enabled by Enable()
func Disable() {
	mutex.Lock()
	defer mutex.Unlock()
	if disable != nil {
		disable()
		disable = nil
	}
}

// ReportMinidump encodes and sends a minidump report to the crashURL endpoint.
func ReportMinidump(r Reporter, minidumpName string, minidumpData []byte) (string, error) {
	if disable != nil {
		return r.reportMinidump(minidumpName, minidumpData, crashURL)
	}
	return "Error reporting disabled", nil
}

func (r Reporter) sendReport(body io.Reader, contentType, endpoint string) (string, error) {
	appNameAndVersion := r.AppName + ":" + r.AppVersion
	url := fmt.Sprintf("%v?product=%v&version=%v", endpoint, url.QueryEscape(crashProduct), url.QueryEscape(appNameAndVersion))

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("Couldn't create new crash report request: %v", err)
	}

	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	res, err := client.Do(req)
	if err == nil {
		defer res.Body.Close()
	}
	if err != nil {
		return "", fmt.Errorf("Failed to upload report request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Failed to upload report request: got HTTP status code %v", res.StatusCode)
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(res.Body); err != nil {
		return "", fmt.Errorf("Failed to write out response buffer: %v", err)
	}

	return buf.String(), nil
}

func (r Reporter) reportStacktrace(s stacktrace.Callstack, endpoint string) (string, error) {
	body, contentType, err := encoder{
		appName:    r.AppName,
		appVersion: r.AppVersion,
		osName:     r.OSName,
		osVersion:  r.OSVersion,
	}.encodeStacktrace(s.String())
	if err != nil {
		return "", fmt.Errorf("Couldn't encode crash report: %v", err)
	}

	return r.sendReport(body, contentType, endpoint)
}

func (r Reporter) reportMinidump(minidumpName string, minidumpData []byte, endpoint string) (string, error) {
	body, contentType, err := encoder{
		appName:    r.AppName,
		appVersion: r.AppVersion,
		osName:     r.OSName,
		osVersion:  r.OSVersion,
	}.encodeMinidump(minidumpName, minidumpData)
	if err != nil {
		return "", fmt.Errorf("Couldn't encode minidump crash report: %v", err)
	}

	return r.sendReport(body, contentType, endpoint)
}
