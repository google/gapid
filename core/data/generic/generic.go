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

	// TO is a placeholder type that represents the type that is being checked
	// for interface compliance (ty parameter to Implements).
	TO struct{}

	// TP is a placeholder type that represents a pointer to the type that is
	// being checked for interface compliance.
	TP struct{}

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

	// TOTy is the type of TO.
	TOTy = reflect.TypeOf(TO{})

	// TPTy is the type of TP.
	TPTy = reflect.TypeOf(TP{})

	anyTy = reflect.TypeOf(Any{})
)

// Match is the result of Implements and CheckSigs.
type Match struct {
	// Errors found matching the interface to the implementation.
	Errors []error

	// Bindings of generic type to matched type.
	Bindings map[reflect.Type]reflect.Type
}

// Ok returns true if the type implemented the generic interface.
func (m Match) Ok() bool { return len(m.Errors) == 0 }

func newSubs(generics ...reflect.Type) map[reflect.Type]reflect.Type {
	subs := map[reflect.Type]reflect.Type{}
	subs[T1Ty] = nil
	subs[T2Ty] = nil
	subs[T3Ty] = nil
	subs[T4Ty] = nil
	for _, g := range generics {
		subs[g] = nil
	}
	return subs
}

// Implements checks that ty implements the generic interface iface.
// Implements returns a Match which lists any generic type inconsistencies, and
// lists the mappings of generic types to implementation types.
// generics is a list of open types used by the generic interface iface.
func Implements(ty, iface reflect.Type, generics ...reflect.Type) Match {
	var errs []error
	subs := newSubs(generics...)
	subs[TOTy] = ty
	subs[TPTy] = reflect.PtrTo(ty)

	for i, c := 0, iface.NumMethod(); i < c; i++ {
		iM := iface.Method(i)

		oM, ok := ty.MethodByName(iM.Name)
		if !ok {
			errs = append(errs, fmt.Errorf("'%v' does not implement method '%v'", ty, iM.Name))
			continue
		}

		if err := checkMethod(oM, iM, subs, true); err != nil {
			errs = append(errs, err)
		}
	}

	return Match{errs, subs}
}

// Sig is a pair of functions used for validation by CheckSigs.
type Sig struct {
	// Name of the function.
	Name string
	// The signature the function must match.
	Interface interface{}
	// The provided function being checked.
	Function interface{}
}

// CheckSigs checks that each function signature implements the generic
// interface signature.
// CheckSigs returns a Match which lists any generic type inconsistencies, and
// lists the mappings of generic types to implementation types.
func CheckSigs(sigs ...Sig) Match {
	var errs []error
	subs := newSubs()
	for i, s := range sigs {
		fun := reflect.TypeOf(s.Function)
		if fun.Kind() != reflect.Func {
			panic(fmt.Errorf("Signature for '%v' requires a function for Function", s.Name))
		}
		iface := reflect.TypeOf(s.Interface)
		if iface.Kind() != reflect.Func {
			panic(fmt.Errorf("Signature for '%v' requires a function for Interface", s.Name))
		}

		oM := reflect.Method{Name: s.Name, Type: fun, Index: i}
		iM := reflect.Method{Name: s.Name, Type: iface, Index: i}
		if err := checkMethod(oM, iM, subs, false); err != nil {
			errs = append(errs, err)
		}
	}
	return Match{errs, subs}
}

func checkMethod(oM, iM reflect.Method, subs map[reflect.Type]reflect.Type, hasReceiver bool) error {
	fail := func(err error) error {
		return fmt.Errorf("%v\nInterface:   %v\nImplementor: %v",
			err, printFunc(iM.Type, 0), printFunc(oM.Type, 1))
	}

	numReceivers := 0
	if hasReceiver {
		numReceivers = 1
	}

	if oC, iC := oM.Type.NumIn()-numReceivers, iM.Type.NumIn(); oC == iC {
		for i, c := 0, iM.Type.NumIn(); i < c; i++ {
			iTy, oTy := iM.Type.In(i), oM.Type.In(i+numReceivers)
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
