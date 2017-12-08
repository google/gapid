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

package reporting

import (
	"bytes"
	"io"
	"mime/multipart"
)

type encoder struct {
	appName    string
	appVersion string
	osName     string
	osVersion  string
}

const (
	crashProduct      = "GAPID"
	multipartBoundary = "::multipart-boundary::"
)

func (e encoder) encodeStacktrace(stacktrace string) (io.Reader, string, error) {
	stacktrace = filterStack(stacktrace)

	buf := bytes.Buffer{}
	w := multipart.NewWriter(&buf)
	if err := w.SetBoundary(multipartBoundary); err != nil {
		return nil, "", err
	}
	defer w.Close()
	if err := w.WriteField("product", crashProduct); err != nil {
		return nil, "", err
	}
	if err := w.WriteField("version", e.appName+":"+e.appVersion); err != nil {
		return nil, "", err
	}
	if e.osName != "" {
		if err := w.WriteField("osName", e.osName); err != nil {
			return nil, "", err
		}
	}
	if e.osVersion != "" {
		if err := w.WriteField("osVersion", e.osVersion); err != nil {
			return nil, "", err
		}
	}
	if stacktrace != "" {
		if err := w.WriteField("exception_info", stacktrace); err != nil {
			return nil, "", err
		}
	}
	return &buf, w.FormDataContentType(), nil
}

func (e encoder) encodeMinidump(minidumpName string, minidumpData []byte) (io.Reader, string, error) {
	buf := bytes.Buffer{}
	w := multipart.NewWriter(&buf)
	if err := w.SetBoundary(multipartBoundary); err != nil {
		return nil, "", err
	}
	defer w.Close()
	if err := w.WriteField("product", crashProduct); err != nil {
		return nil, "", err
	}
	if err := w.WriteField("version", e.appName+":"+e.appVersion); err != nil {
		return nil, "", err
	}
	if e.osName != "" {
		if err := w.WriteField("osName", e.osName); err != nil {
			return nil, "", err
		}
	}
	if e.osVersion != "" {
		if err := w.WriteField("osVersion", e.osVersion); err != nil {
			return nil, "", err
		}
	}
	if minidumpName != "" && minidumpData != nil {
		filefield, err := w.CreateFormFile("uploadFileMinidump", minidumpName)
		if err != nil {
			return nil, "", err
		}
		if _, err = io.Copy(filefield, bytes.NewReader(minidumpData)); err != nil {
			return nil, "", err
		}
	}
	return &buf, w.FormDataContentType(), nil
}
