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

package codegen

type exceptions struct {
	allocFn       *Function
	throwFn       *Function
	personalityFn *Function
	exceptionTy   Type
	typeInfos     map[Type]Global
}

func (e *exceptions) init(m *Module) {
	voidPtr := m.Types.Pointer(m.Types.Void)

	e.exceptionTy = m.Types.TypeOf(struct {
		*uint8
		int32
	}{})
	e.personalityFn = m.Function(m.Types.Int32, "__gxx_personality_v0", Variadic)

	// void* __cxa_allocate_exception(size_t size)
	e.allocFn = m.Function(voidPtr, "__cxa_allocate_exception", m.Types.Size)
	// void __cxa_throw(void *thrown_object, std::type_info *tinfo, void (*destructor)(void *)) {
	e.throwFn = m.Function(m.Types.Void, "__cxa_throw", voidPtr, voidPtr, voidPtr)

	e.typeInfos = map[Type]Global{
		m.Types.Int:    m.Extern("_ZTIi", voidPtr),
		m.Types.Uint32: m.Extern("_ZTIj", voidPtr),
		m.Types.Uint64: m.Extern("_ZTIm", voidPtr),
	}
}

func (e *exceptions) throw(b *Builder, v *Value) {
	t := b.m.Types
	voidPtr := t.Pointer(b.m.Types.Void)
	ex := b.Call(e.allocFn, b.Scalar(v.Type().SizeInBits()/8).Cast(t.Size))
	ex.Cast(t.Pointer(v.Type())).Store(v)
	ty, ok := e.typeInfos[v.Type()]
	if !ok {
		fail("Unsuporrted exception type %v", v.Type())
	}
	destructor := b.Zero(voidPtr)
	b.Call(e.throwFn, ex, ty.Value(b).Cast(voidPtr), destructor)
}
