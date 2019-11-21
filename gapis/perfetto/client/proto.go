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
