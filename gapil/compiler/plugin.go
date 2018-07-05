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
	"github.com/google/gapid/gapil/semantic"
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
	Name string                              // Name of the field
	Type codegen.Type                        // Type of the field
	Init func(s *S, fieldPtr *codegen.Value) // Optional initializer
	Term func(s *S, fieldPtr *codegen.Value) // Optional terminator
}

// ContextDataPlugin is the interface implemented by plugins that require
// additional data to be stored in the runtime context.
type ContextDataPlugin interface {
	// OnPreBuildContext is called just before the context structure is built.
	// This can be used to build any additional types that are returned by
	// ContextData().
	OnPreBuildContext(*C)

	// ContextData returns a slice of additional ContextFields that will be
	// augmented to the context structure.
	ContextData(*C) []ContextField
}

// FunctionExposerPlugin is the interface implemented by plugins that build
// public functions. These functions will be exposed on the output Program.
type FunctionExposerPlugin interface {
	Functions() map[string]*codegen.Function
}

// OnBeginCommandListener is the interface implemented by plugins that generate
// custom logic at the start of the command.
type OnBeginCommandListener interface {
	OnBeginCommand(s *S, cmd *semantic.Function)
}

// OnFenceListener is the interface implemented by plugins that generate
// custom logic at the fence of the command.
type OnFenceListener interface {
	OnFence(s *S)
}

// OnEndCommandListener is the interface implemented by plugins that generate
// custom logic at the end of the command.
type OnEndCommandListener interface {
	OnEndCommand(s *S, cmd *semantic.Function)
}

// OnReadListener is the interface implemented by plugins that generate custom
// logic when slices are read.
type OnReadListener interface {
	OnRead(s *S, slice *codegen.Value, ty *semantic.Slice)
}

// OnWriteListener is the interface implemented by plugins that generate custom
// logic when slices are written.
type OnWriteListener interface {
	OnWrite(s *S, slice *codegen.Value, ty *semantic.Slice)
}
