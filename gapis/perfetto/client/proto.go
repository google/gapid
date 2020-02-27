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

package client

import (
	"context"
	"errors"

	"github.com/golang/protobuf/proto"

	common "protos/perfetto/common"
	ipc "protos/perfetto/ipc"
)

// NewQuerySync returns an InvokeSync that handles unmarshalling the proto bytes
// and invokes the given callback.
func NewQuerySync(ctx context.Context, cb func(*common.TracingServiceState) error) *InvokeSync {
	return NewInvokeSync(ctx, func(data []byte) error {
		resp := &ipc.QueryServiceStateResponse{}
		if err := proto.Unmarshal(data, resp); err != nil {
			return err
		}
		return cb(resp.GetServiceState())
	})
}

// NewTraceHandler returns an InvokeHandler that handles unmarshalling the proto
// bytes of an EnableTrace response and invokes the given callback.
func NewTraceHandler(ctx context.Context, cb func(*ipc.EnableTracingResponse, error)) InvokeHandler {
	return func(data []byte, more bool, err error) {
		if err != nil {
			cb(nil, err)
			return
		}
		if more {
			cb(nil, errors.New("Got a streaming reply to the non-stream EnableTracing RPC"))
			return
		}
		resp := &ipc.EnableTracingResponse{}
		if err := proto.Unmarshal(data, resp); err != nil {
			cb(nil, err)
			return
		}
		cb(resp, nil)
	}
}

// NewReadHandler returns an InvokeHandler that handles unmarshalling the proto
// bytes of a ReadBuffers response and invokes the given callback.
func NewReadHandler(ctx context.Context, cb func(*ipc.ReadBuffersResponse, bool, error)) InvokeHandler {
	return func(data []byte, more bool, err error) {
		if err != nil {
			cb(nil, false, err)
			return
		}
		resp := &ipc.ReadBuffersResponse{}
		if err := proto.Unmarshal(data, resp); err != nil {
			cb(nil, false, err)
			return
		}
		cb(resp, more, nil)
	}
}

// NewIgnoreHandler returns an InvokeHanlder that ignores the returned result,
// but propagates errors and ensure the response is not streaming.
func NewIgnoreHandler(ctx context.Context, cb func(error)) InvokeHandler {
	return func(data []byte, more bool, err error) {
		if err != nil {
			cb(err)
			return
		}
		if more {
			cb(errors.New("Got a streaming response to non-streaming request"))
			return
		}
		cb(nil)
	}
}
