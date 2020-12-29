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

//go:build !crashreporting

package reporting

// This file contains stub implementations of the reporting package (used when
// the crashreporting build tag is omitted).
//
// As cmd/do references this package, be very careful not to pull in imports
// that cannot be built with do. For example anything that uses protobufs.

import "context"

// Enable turns on crash reporting if the running processes panics inside a
// crash.Go block.
func Enable(ctx context.Context, appName, appVersion string) {}

// Disable turns off crash reporting previously enabled by Enable()
func Disable() {}

// ReportMinidump encodes and sends a minidump report to the crashURL endpoint.
func ReportMinidump(r Reporter, minidumpName string, minidumpData []byte) (string, error) {
	return "", nil
}
