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
	personalityFn *Function
	exceptionTy   Type
}

func (e *exceptions) init(m *Module) {
	voidPtr := m.Types.Pointer(m.Types.Void)

	e.exceptionTy = m.Types.TypeOf(struct {
		*uint8
		int32
	}{})
	e.personalityFn = m.Function(m.Types.Int32, "__gxx_personality_v0", Variadic)

	// Declare the unwind resume function. This isn't directly referenced by
	// this package, but is referenced by the lowered assembly. By declaring
	// this function we will automatically check that the symbol can be found
	// when JIT compiling.
	//
	// void _Unwind_Resume(struct _Unwind_Exception * object);
	m.Function(m.Types.Void, "_Unwind_Resume", voidPtr)
}
