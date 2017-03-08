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
	"testing"

	"strings"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
)

func TestBroadcast(t *testing.T) {
	assert := assert.To(t)
	aWri, aBuf := log.Buffer()
	bWri, bBuf := log.Buffer()
	cWri, cBuf := log.Buffer()
	broadcaster := log.Broadcast(
		log.Raw.Handler(aWri),
		log.Detailed.Handler(bWri),
		log.Normal.Handler(cWri),
	)
	for _, test := range testMessages {
		aBuf.Reset()
		bBuf.Reset()
		cBuf.Reset()
		test.send(broadcaster)
		aStr := strings.TrimRight(aBuf.String(), "\n")
		bStr := strings.TrimRight(bBuf.String(), "\n")
		cStr := strings.TrimRight(cBuf.String(), "\n")
		assert.For("%s A", test.msg).That(aStr).Equals(test.raw)
		assert.For("%s B", test.msg).That(bStr).Equals(test.detailed)
		assert.For("%s C", test.msg).That(cStr).Equals(test.normal)
	}
}

type messages []*log.Message

func (l *messages) Handle(m *log.Message) { *l = append(*l, m) }
func (l *messages) Close()                {}

func TestBroadcasterListen(t *testing.T) {
	assert := assert.To(t)
	b := log.Broadcast()

	// check that the broadcaster copes with no listeners
	b.Handle(&log.Message{})

	// check that the broadcaster broadcasts
	p1 := messages{}
	b.Listen(&p1)
	b.Handle(&log.Message{})
	assert.For("p1").ThatSlice(p1).IsNotEmpty()

	// check that the broadcaster respects unlisten
	p2 := messages{}
	unlisten := b.Listen(&p2)
	unlisten()
	b.Handle(&log.Message{})
	assert.For("p2").ThatSlice(p2).IsEmpty()
}
