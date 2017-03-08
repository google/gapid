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

package log_test

import (
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
)

func TestChannel(t *testing.T) {
	assert := assert.To(t)
	w, b := log.Buffer()
	ping := make(chan struct{})
	handler := log.Channel(pingHandler{log.Normal.Handler(w), ping}, 0)
	defer handler.Close()
	for _, test := range testMessages {
		b.Reset()
		test.send(handler)
		<-ping
		got := strings.TrimRight(b.String(), "\n")
		assert.For(test.msg).That(got).Equals(test.normal)
	}
}

type pingHandler struct {
	h log.Handler
	c chan struct{}
}

func (h pingHandler) Handle(m *log.Message) {
	h.h.Handle(m)
	h.c <- struct{}{}
}

func (h pingHandler) Close() {
	h.h.Close()
	close(h.c)
}
