// Copyright (C) 2019 Google Inc.
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

package perfetto

// #include <stdlib.h> // free
// #include "cc/processor.h"
import "C"
import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service"
)

var (
	lastQueryId    int32
	pendingQueries sync.Map
)

type Processor struct {
	handle C.processor
	mutex  sync.Mutex
}

type Result struct {
	Ready  task.Signal
	Result service.PerfettoQueryResult
	done   task.Task
}

func NewProcessor(ctx context.Context, data []byte) (*Processor, error) {
	p := C.new_processor()
	log.D(ctx, "[perfetto] Parsing %d bytes", len(data))
	if !C.parse_data(p, unsafe.Pointer(&data[0]), C.size_t(len(data))) {
		log.W(ctx, "[perfetto] Parsing failed")
		C.delete_processor(p)
		return nil, log.Errf(ctx, nil, "parsing trace failed")
	}
	return &Processor{handle: p}, nil
}

func (p *Processor) Query(q string) *Result {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	s, done := task.NewSignal()
	r := &Result{
		Ready: s,
		done:  done,
	}
	id := atomic.AddInt32(&lastQueryId, 1)
	pendingQueries.Store(id, r)

	qPtr := C.CString(q)
	C.execute_query(p.handle, C.int(id), qPtr)
	C.free(unsafe.Pointer(qPtr))

	return r
}

func (p *Processor) Close() {
	C.delete_processor(p.handle)
	p.handle = nil
}

//export on_query_complete
func on_query_complete(id C.int, data unsafe.Pointer, size C.ulong) {
	if rp, ok := pendingQueries.Load(int32(id)); ok {
		r := rp.(*Result)
		pendingQueries.Delete(int32(id))
		if err := proto.Unmarshal((*[1 << 31]byte)(data)[:int(size)], &r.Result); err != nil {
			r.Result.Error = fmt.Sprintf("Failed to parse proto response: %v")
		}
		r.done(nil)
	}
}
