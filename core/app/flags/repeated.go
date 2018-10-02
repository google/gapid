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

package flags

import (
	"flag"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type repeated struct {
	value  reflect.Value
	single reflect.Value
	parser flag.Value
}

const (
	dummyFlag = "dummy"
)

func newRepeatedFlag(value reflect.Value) flag.Value {
	fs := flag.NewFlagSet("", flag.ContinueOnError)

	single := reflect.New(value.Type().Elem())
	switch s := single.Interface().(type) {
	case *bool:
		fs.BoolVar(s, dummyFlag, *s, "")
	case *int:
		fs.IntVar(s, dummyFlag, *s, "")
	case *int64:
		fs.Int64Var(s, dummyFlag, *s, "")
	case *uint:
		fs.UintVar(s, dummyFlag, *s, "")
	case *uint64:
		fs.Uint64Var(s, dummyFlag, *s, "")
	case *float64:
		fs.Float64Var(s, dummyFlag, *s, "")
	case *string:
		fs.StringVar(s, dummyFlag, *s, "")
	case *time.Duration:
		fs.DurationVar(s, dummyFlag, *s, "")
	case flag.Value:
		fs.Var(s, dummyFlag, "")
	default:
		panic(fmt.Sprintf("Unhandled flag type: %v", single.Type()))
	}

	return &repeated{value, single, fs.Lookup(dummyFlag).Value}
}

func (f *repeated) String() string {
	strs := make([]string, f.value.Len())
	for i := 0; i < len(strs); i++ {
		strs[i] = f.value.Index(i).String()
	}
	return strings.Join(strs, ":")
}

func (f *repeated) Set(value string) error {
	if err := f.parser.Set(value); err != nil {
		return err
	}
	f.value.Set(reflect.Append(f.value, f.single.Elem()))
	return nil
}
