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

package grpcutil

import (
	"context"
	"io"
	"reflect"

	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/log"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// ToProducer adapts an input grpc stream to an event producer.
func ToProducer(stream interface{}) event.Producer {
	s := stream.(grpc.Stream)
	meth, _ := reflect.TypeOf(stream).MethodByName("Recv")
	t := meth.Type.Out(0).Elem()
	return func(ctx context.Context) interface{} {
		m := reflect.New(t)
		if err := s.RecvMsg(m.Interface()); err != nil {
			if errors.Cause(err) == io.EOF {
				return nil
			}
			log.E(ctx, "%v", err)
			return nil
		}
		return m.Interface()
	}
}
