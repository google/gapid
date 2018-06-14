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

package compiler

import (
	"reflect"

	"github.com/google/gapid/core/codegen"
)

// Plugin is a extension for the compiler.
type Plugin interface {
	Build(*C)
}

type plugins []Plugin

func (l plugins) foreach(cb interface{}) {
	cbV := reflect.ValueOf(cb)
	cbT := cbV.Type()
	if cbT.Kind() != reflect.Func || cbT.NumIn() != 1 {
		panic("foreach() requires a function of the signature func(T)")
	}
	ty := cbT.In(0)
	for _, p := range l {
		pV := reflect.ValueOf(p)
		if pV.Type().Implements(ty) {
			cbV.Call([]reflect.Value{pV})
		}
	}
}

// ContextField represents a single additional context field added by a
// ContextDataPlugin.
type ContextField struct {
	Name string                       // Name of the field
	Type codegen.Type                 // Type of the field
	Init func(s *S) *codegen.Value    // Optional initializer
	Term func(s *S, v *codegen.Value) // Optional terminator
}

// ContextDataPlugin is the interface implemented by plugins that require
// additional data to be stored in the runtime context.
type ContextDataPlugin interface {
	ContextData(*C) []ContextField
}

