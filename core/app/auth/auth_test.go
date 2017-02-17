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

package auth_test

import (
	"bytes"
	"testing"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/assert"
)

func TestWrite(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		name     string
		token    auth.Token
		expected []byte
	}{
		{
			name:     "no-auth",
			token:    auth.NoAuth,
			expected: []byte{},
		},
		{
			name:     "auth-abcdef",
			token:    auth.Token("abc"),
			expected: []byte{'A', 'U', 'T', 'H', 'a', 'b', 'c', 0},
		},
	} {
		buf := &bytes.Buffer{}
		auth.Write(buf, test.token)
		assert.For(test.name).ThatSlice(buf.Bytes()).Equals(test.expected)
	}
}

func TestGenToken(t *testing.T) {
	assert := assert.To(t)
	token := auth.GenToken()
	assert.For("length").That(len(token)).Equals(8)
}

type readCloser struct {
	*bytes.Buffer
	closed bool
}

func (r *readCloser) Close() error {
	r.closed = true
	return nil
}
