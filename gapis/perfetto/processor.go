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
	"sync"
	"unsafe"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/perfetto/service"
)

type Processor struct {
	handle C.processor
	mutex  sync.Mutex
}

func NewProcessor(ctx context.Context, data []byte) (*Processor, error) {
	if len(data) == 0 {
		log.W(ctx, "[perfetto] Empty profile data.")
		return nil, log.Errf(ctx, nil, "profile data is empty.")
	}
	p := C.new_processor()
	log.D(ctx, "[perfetto] Parsing %d bytes", len(data))
	if !C.parse_data(p, unsafe.Pointer(&data[0]), C.size_t(len(data))) {
		log.W(ctx, "[perfetto] Parsing failed")
		C.delete_processor(p)
		return nil, log.Errf(ctx, nil, "parsing trace failed")
	}
	return &Processor{handle: p}, nil
}

func (p *Processor) Query(q string) (*service.QueryResult, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	r := &service.QueryResult{}

	qPtr := C.CString(q)
	res := C.execute_query(p.handle, qPtr)
	// Convert the cgo memory pointer to a go slice and parse the proto.
	err := proto.Unmarshal((*[1 << 31]byte)(unsafe.Pointer(res.data))[:int(res.size)], r)
	C.free(unsafe.Pointer(res.data))
	C.free(unsafe.Pointer(qPtr))

	return r, err
}

func (p *Processor) Close() {
	if p == nil {
		return
	}
	C.delete_processor(p.handle)
	p.handle = nil
}
