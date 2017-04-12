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

package database

import (
	"context"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/log"
)

func toProto(ctx context.Context, v interface{}) (proto.Message, error) {
	// If v is a proto message, then there's no work to do.
	if v, ok := v.(proto.Message); ok {
		return v, nil
	}
	if b := pod.NewValue(v); b != nil {
		return b, nil
	}
	// Check the registered proto converters.
	msg, err := protoconv.ToProto(ctx, v)
	switch err.(type) {
	case nil:
		return msg, nil
	case protoconv.ErrNoConverterRegistered:
		return nil, log.Errf(ctx, err, "Cannot encode type %T (%v)", v, reflect.TypeOf(v).Kind())
	default:
		return nil, log.Errf(ctx, err, "Failed to convert %T to proto", v)
	}
}
