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
	"crypto/tls"
	"fmt"
	"net/http"
)

type endpoint func(payloads []string) error

func newBatchEndpoint(useragent string) endpoint {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	return func(payloads []string) error {
		data := bytes.Buffer{}
		for i, p := range payloads {
			if i > 0 {
				data.WriteRune('\n')
			}
			data.WriteString(p)
		}

		req, err := http.NewRequest("POST", "https://www.google-analytics.com/batch", &data)
		if err != nil {
			return err
		}

		if useragent != "" {
			req.Header.Set("User-Agent", useragent)
		}

		res, err := client.Do(req)
		if err != nil {
			return err
		}
		if res.StatusCode != 200 {
			return fmt.Errorf("Got status %v", res.StatusCode)
		}

		return nil
	}
}

func newValidateEndpoint() endpoint {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	return func(payloads []string) error {
		for _, p := range payloads {
			data := bytes.NewBufferString(p)

			req, err := http.NewRequest("POST", "https://www.google-analytics.com/debug/collect", data)
			if err != nil {
				return err
			}

			res, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			if res.StatusCode != 200 {
				panic(fmt.Errorf("Got status %v", res.StatusCode))
			}
		}
		return nil
	}
}
