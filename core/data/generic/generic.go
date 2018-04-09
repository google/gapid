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

// Package generic provides methods to test whether a type conforms to a generic
// interface.
package generic

import (
	"bytes"
	"fmt"
	"reflect"
)

type (
	// T1 is a placeholder type that can be used for generic subsitutions.
	T1 struct{}

	// T2 is a placeholder type that can be used for generic subsitutions.
	T2 struct{}

	// T3 is a placeholder type that can be used for generic subsitutions.
	T3 struct{}

	// T4 is a placeholder type that can be used for generic subsitutions.
	T4 struct{}

	// Any is the type used for open types.
	Any struct{}
)

var (
	// T1Ty is the type of T1.
	T1Ty = reflect.TypeOf(T1{})

	// T2Ty is the type of T2.
	T2Ty = reflect.TypeOf(T2{})

	// T3Ty is the type of T3.
	T3Ty = reflect.TypeOf(T3{})

	// T4Ty is the type of T4.
	T4Ty = reflect.TypeOf(T4{})

	anyTy = reflect.TypeOf(Any{})
)

// Match is the result of Implements.
type Match struct {
	// Errors found matching the interface to the implementation.
	Errors []error

	// Bindings of generic type to matched type.
	Bindings map[reflect.Type]reflect.Type
}

// Ok returns true if the type implemented the generic interface.
func (m Match) Ok() bool { return len(m.Errors) == 0 }

// Implements returns nil if ty implements the generic interface iface,
// otherwise Implements returns the list of interface violations.
// generics is a list of open types.
func Implements(ty, iface reflect.Type, generics ...reflect.Type) Match {
	var errs []error

	subs := map[reflect.Type]reflect.Type{}
	for _, g := range generics {
		subs[g] = nil
	}

	for i, c := 0, iface.NumMethod(); i < c; i++ {
		iM := iface.Method(i)

		oM, ok := ty.MethodByName(iM.Name)
		if !ok {
			errs = append(errs, fmt.Errorf("'%v' does not implement method '%v'", ty, iM.Name))
			continue
		}

		if err := checkMethod(oM, iM, subs); err != nil {
			errs = append(errs, err)
		}
	}

	return Match{errs, subs}
}

func checkMethod(oM, iM reflect.Method, subs map[reflect.Type]reflect.Type) error {
	fail := func(err error) error {
		return fmt.Errorf("%v\nInterface:   %v\nImplementor: %v",
			err, printFunc(iM.Type, 0), printFunc(oM.Type, 1))
	}

	if oC, iC := oM.Type.NumIn()-1, iM.Type.NumIn(); oC == iC { // -1 to skip the this.
		for i, c := 0, iM.Type.NumIn(); i < c; i++ {
			iTy, oTy := iM.Type.In(i), oM.Type.In(i+1)
			if err := checkType(oTy, iTy, subs); err != nil {
				return fail(err)
			}
		}
	} else {
		if oC < iC {
			return fail(fmt.Errorf("method '%v' has too few parameters", iM.Name))
		}
		return fail(fmt.Errorf("method '%v' has too many parameters", iM.Name))
	}

	if oC, iC := oM.Type.NumOut(), iM.Type.NumOut(); oC >= iC {
		// Too many return values is okay so long as the ones that are returned match.
		for i, c := 0, iM.Type.NumOut(); i < c; i++ {
			iTy, oTy := iM.Type.Out(i), oM.Type.Out(i)
			if err := checkType(oTy, iTy, subs); err != nil {
				return fail(err)
			}
		}
	} else {
		return fail(fmt.Errorf("method '%v' has too few return values", iM.Name))
	}

	return nil
}

func checkType(oTy, iTy reflect.Type, subs map[reflect.Type]reflect.Type) error {
	if iTy.Kind() == oTy.Kind() {
		switch iTy.Kind() {
		case reflect.Slice, reflect.Array:
			return checkType(oTy.Elem(), iTy.Elem(), subs)
		case reflect.Map:
			if err := checkType(oTy.Key(), iTy.Key(), subs); err != nil {
				return err
			}
			return checkType(oTy.Elem(), iTy.Elem(), subs)
		}
	}

	switch iTy {
	case oTy, anyTy: // type match
		return nil
	}
	sub, ok := subs[iTy]
	if !ok {
		return fmt.Errorf("type mismatch '%v' and '%v'", iTy, oTy)
	}
	switch sub {
	case nil: // First time this generic was seen. Map it to oTy.
		subs[iTy] = oTy
	case oTy, anyTy: // Consistent generic usage.
	default:
		// We're erroring about inconsistent usage of this generic.
		// Mark it as any to prevent repeated errors of this generic.
		subs[iTy] = anyTy
		return fmt.Errorf("mixed use of generic type '%v'. First used as '%v', now used as '%v'",
			iTy, sub, oTy)
	}
	return nil
}

func printFunc(f reflect.Type, fisrtParm int) string {
	if f.Kind() != reflect.Func {
		return "Not a function"
	}
	b := bytes.Buffer{}
	b.WriteString("func(")
	for i := fisrtParm; i < f.NumIn(); i++ {
		if i > fisrtParm {
			b.WriteString(", ")
		}
		b.WriteString(f.In(i).String())
	}
	b.WriteString(")")
	switch f.NumOut() {
	case 0:
	case 1:
		b.WriteString(" ")
		b.WriteString(f.Out(0).String())
	default:
		b.WriteString(" (")
		for i := 0; i < f.NumOut(); i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(f.Out(i).String())
		}
		b.WriteString(")")
	}
	return b.String()
}
