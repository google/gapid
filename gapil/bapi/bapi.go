// Copyright (C) 2018 Google Inc.
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

// Package bapi exposes functions for serializing and deserializing resolved
// APIs.
package bapi

import "reflect"

func foreach(sli interface{}, cb interface{}, outPtr interface{}) {
	sliV := reflect.ValueOf(sli)
	count := sliV.Len()
	if count == 0 {
		return
	}
	outPtrV := reflect.ValueOf(outPtr)
	outV := outPtrV.Elem()
	outV.Set(reflect.MakeSlice(outPtrV.Type().Elem(), count, count))
	cbV := reflect.ValueOf(cb)
	args := make([]reflect.Value, 1)
	for i := 0; i < count; i++ {
		args[0] = sliV.Index(i)
		res := cbV.Call(args)
		outV.Index(i).Set(res[0])
	}
}
