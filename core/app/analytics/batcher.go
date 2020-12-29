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

import "time"

const (
	batchBuffer = 2000
	batchDelay  = time.Second
)

type batcher struct {
	endpoint endpoint
	input    chan Payload
	sync     chan func()
	encoder  encoder
}

func newBatcher(p endpoint, e encoder) sender {
	out := &batcher{
		endpoint: p,
		input:    make(chan Payload, batchBuffer),
		sync:     make(chan func()),
		encoder:  e,
	}
	go out.run()
	return out
}

func (b *batcher) send(p Payload) {
	b.input <- p
}

func (b *batcher) flush() {
	done := make(chan struct{})
	b.sync <- func() { close(done) }
	<-done
}

func (b *batcher) run() {
	size, payloads := 0, make([]string, 0, maxHitsPerBatch)

	flush := func() {
		if len(payloads) > 0 {
			if err := b.endpoint(payloads); err != nil {
				OnError(err)
			}
		}
		size, payloads = 0, payloads[:0]
	}

	for {
		select {
		case p := <-b.input:
			buf, err := b.encoder(p)
			if err != nil {
				OnError(err)
				continue
			}

			bufSizeWithNL := buf.Len() + 1

			if bufSizeWithNL > maxBatchSize {
				flush()
			}

			size += bufSizeWithNL
			payloads = append(payloads, buf.String())

			if len(payloads) == maxHitsPerBatch {
				flush()
			}

		case done := <-b.sync:
			flush()
			done()

		case <-time.After(batchDelay):
			flush()
		}
	}
}
