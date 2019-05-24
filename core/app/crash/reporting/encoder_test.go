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
	"io/ioutil"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
)

func TestEncoder(t *testing.T) {
	ctx := log.Testing(t)

	r, ty, err := encoder{
		appName:    "TestApp",
		appVersion: "V1.2.3",
		osName:     "BobOS",
		osVersion:  "10",
	}.encodeStacktrace("foo1.bar 10:20\nfoo2.bar 20:30")
	if assert.For(ctx, "err").ThatError(err).Succeeded() {
		assert.For(ctx, "ty").ThatString(ty).Equals("multipart/form-data; boundary=\"" + multipartBoundary + "\"")
		bytes, err := ioutil.ReadAll(r)
		if assert.For(ctx, "ReadAll").ThatError(err).Succeeded() {
			expect := "--" + multipartBoundary + "\r\n" +
				"Content-Disposition: form-data; name=\"product\"\r\n" +
				"\r\n" +
				"GAPID\r\n" +
				"--" + multipartBoundary + "\r\n" +
				"Content-Disposition: form-data; name=\"version\"\r\n" +
				"\r\n" +
				"TestApp:V1.2.3\r\n" +
				"--" + multipartBoundary + "\r\n" +
				"Content-Disposition: form-data; name=\"osName\"\r\n" +
				"\r\n" +
				"BobOS\r\n" +
				"--" + multipartBoundary + "\r\n" +
				"Content-Disposition: form-data; name=\"osVersion\"\r\n" +
				"\r\n" +
				"10\r\n" +
				"--" + multipartBoundary + "\r\n" +
				"Content-Disposition: form-data; name=\"exception_info\"\r\n" +
				"\r\n" +
				"foo1.bar 10:20\n" +
				"foo2.bar 20:30\r\n" +
				"--" + multipartBoundary + "--\r\n"
			if !assert.For(ctx, "body").ThatString(string(bytes)).Equals(expect) {
				assert.For(ctx, "body").ThatSlice(bytes).Equals([]byte(expect))
			}
		}
	}
}
