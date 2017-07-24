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

package deep

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/core/data"
)

// Clone makes a deep copy of v.
func Clone(v interface{}) (interface{}, error) {
	s := reflect.ValueOf(v)
	d := reflect.New(s.Type())
	if err := reflectCopy(d.Elem(), s, "val", map[reflect.Value]reflect.Value{}); err != nil {
		return nil, err
	}
	return d.Elem().Interface(), nil
}

// MustClone makes a deep copy of v, or panics if it could not.
func MustClone(v interface{}) interface{} {
	s := reflect.ValueOf(v)
	d := reflect.New(s.Type())
	if err := reflectCopy(d.Elem(), s, "val", map[reflect.Value]reflect.Value{}); err != nil {
		panic(err)
	}
	return d.Elem().Interface()
}

// Copy recursively copies all fields, map and slice elements from the value
// src to the pointer dst.
func Copy(dst, src interface{}) error {
	d, s := reflect.ValueOf(dst), reflect.ValueOf(src)
	if d.Kind() != reflect.Ptr {
		return fmt.Errorf("dst should be a pointer, got %T", dst)
	}
	return reflectCopy(d.Elem(), s, "val", map[reflect.Value]reflect.Value{})
}

func reflectCopy(d, s reflect.Value, path string, seen map[reflect.Value]reflect.Value) error {
	//	fmt.Printf("%v: d:%v (%v), s:%v (%v) %+v\n", path, d.Type(), d.Kind(), s.Type(), s.Kind(), s.Interface())
	if !d.CanSet() {
		return fmt.Errorf("Cannot assign to %v", path)
	}
	if a, ok := d.Addr().Interface().(data.Assignable); ok && a.Assign(s.Interface()) {
		return nil
	}

	if d.Kind() != s.Kind() && d.Kind() != reflect.Interface {
		if d.Kind() == reflect.Ptr && s.Kind() == reflect.Interface {
			// To workaround https://github.com/golang/go/issues/20013, cyclic
			// type declarations are declared using an interface{} for the
			// pointer.
			// See box.go.
			return reflectCopy(d, s.Elem(), path, seen)
		}
		return fmt.Errorf("Kind mismatch at %v. %v (%v) != %v (%v)",
			path, d.Kind(), d.Type(), s.Kind(), s.Type())
	}

	switch d.Kind() {
	case reflect.Struct:
		for i, c := 0, d.Type().NumField(); i < c; i++ {
			f := d.Type().Field(i)
			if f.PkgPath != "" {
				continue // Unexported.
			}
			d, s := d.Field(i), s.FieldByName(f.Name)
			if !s.IsValid() {
				continue // Source is missing field
			}
			if err := reflectCopy(d, s, path+"."+f.Name, seen); err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		if s.IsNil() {
			d.Set(s)
			return nil
		}
		d.Set(reflect.MakeMap(d.Type()))
		for _, k := range s.MapKeys() {
			v := reflect.New(d.Type().Elem()).Elem()
			path := path + fmt.Sprintf("[%v]", k.Interface())
			if err := reflectCopy(v, s.MapIndex(k), path, seen); err != nil {
				return err
			}
			d.SetMapIndex(k, v)
		}
		return nil
	case reflect.Slice:
		if s.IsNil() {
			d.Set(reflect.New(d.Type()).Elem()) // Assign nil
			return nil
		}
		d.Set(reflect.MakeSlice(s.Type(), s.Len(), s.Len()))
		for i, c := 0, s.Len(); i < c; i++ {
			path := path + fmt.Sprintf("[%v]", i)
			if err := reflectCopy(d.Index(i), s.Index(i), path, seen); err != nil {
				return err
			}
		}
		return nil
	case reflect.Ptr:
		if s.IsNil() {
			d.Set(reflect.New(d.Type()).Elem()) // Assign nil
			return nil
		}
		d.Set(reflect.New(d.Type().Elem())) // Assign new(T)
		if s, cyclic := seen[s]; cyclic {
			d.Elem().Set(s.Elem())
			return nil
		}
		seen[s] = d
		return reflectCopy(d.Elem(), s.Elem(), path, seen)
	default:
		v := s.Convert(d.Type())
		d.Set(v)
		return nil
	}
}
