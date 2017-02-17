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

package objview

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/google/gapid/test/robot/web/client/dom"
)

type nameObjPair struct {
	name string
	obj  interface{}
	fmts *Formatters
}

// View is a DOM element that can render arbitrary go structures as tables
// for inspection purposes. It supports a stack of objects with breadcrumb
// navigation.
type View struct {
	*dom.Div
	objStack  []nameObjPair
	selection int
}

// Formatters is a collection of functions that can format the values on
// certain paths (e.g. myStruct/myField/mySubfield) in an object hierarchy
// in an application-specific way (e.g. render a URL string as a link).
type Formatters struct {
	formatters []func(string, interface{}) (bool, interface{})
}

// Representer is implemented by objects wishing to provide their own
// representation for use in the object viewer.
type Representer interface {
	Representation() interface{}
}

// NewFormatters instantiates a new Formatters collection.
func NewFormatters() *Formatters {
	return &Formatters{}
}

func (f *Formatters) apply(path string, obj interface{}) (bool, interface{}) {
	if f == nil {
		return false, nil
	}
	for _, fmt := range f.formatters {
		if ok, res := fmt(path, obj); ok {
			return ok, res
		}
	}
	return false, nil
}

// Add registers a formatting function that is applied to paths matching the given pattern.
func (f *Formatters) Add(pattern string, fun func(string, interface{}) interface{}) *Formatters {
	re := regexp.MustCompile(pattern)
	f.formatters = append(f.formatters, func(path string, obj interface{}) (bool, interface{}) {
		if !re.MatchString(path) {
			return false, nil
		}
		return true, fun(path, obj)
	})
	return f
}

// NewView instantiates a new object viewer.
func NewView() *View {
	v := &View{Div: dom.NewDiv(), selection: -1}
	return v
}

// Expandable is a formatting function that formats a value as a link
// that pushes that value onto the object viewer stack.
func (j *View) Expandable(path string, obj interface{}) interface{} {
	return j.NewPusher("(view...)", path, obj, nil)
}

// Set sets the viewer stack to the single passed object, with the
// given name (in the navigation bar) and formatters.
func (j *View) Set(name string, obj interface{}, fmt *Formatters) {
	j.objStack = []nameObjPair{{name, obj, fmt}}
	j.render(0)
}

// Pop navigates downwards in the object stack.
func (j *View) Pop() {
	if j.selection == 0 {
		return
	}
	j.render(j.selection - 1)
}

// Push pushes a new object onto the stack with the given name
// and formatters. Any objects in the navigation bar past the currently
// selected one are discarded and replaced with the new object.
func (j *View) Push(name string, obj interface{}, fmt *Formatters) {
	j.objStack = j.objStack[0 : j.selection+1]
	j.objStack = append(j.objStack, nameObjPair{name, obj, fmt})
	j.render(len(j.objStack) - 1)
}

func (j *View) clearView() {
	for c := j.Get("firstChild"); c != nil; {
		j.Call("removeChild", c)
		c = j.Get("firstChild")
	}
}
func (j *View) link(s string, cb func(dom.MouseEvent)) interface{} {
	a := dom.NewA()
	a.Set("href", "#")
	a.Append(s)
	a.OnClick(func(m dom.MouseEvent) {
		cb(m)
		m.PreventDefault()
	})
	return a
}

func (j *View) addCrumbs() {
	crumbs := dom.NewDiv()
	crumbs.Append("Object Viewer: ")

	crumbs.Append(j.link("(hide)", func(m dom.MouseEvent) {
		j.clearView()
		j.selection = -1
		j.addCrumbs()
	}))

	for i, o := range j.objStack {
		i := i
		crumbs.Append(" / ")
		var txt string
		if i == j.selection {
			txt = fmt.Sprintf("[%s]", o.name)
		} else {
			txt = fmt.Sprintf("%s", o.name)
		}
		crumbs.Append(j.link(txt, func(m dom.MouseEvent) {
			j.render(i)
		}))
	}
	j.Append(crumbs)
}
func (j *View) addViewer() {
	sel := j.objStack[j.selection]
	jsonView := j.buildObjectView("", sel.obj, sel.fmts)
	j.Append(jsonView)
}

func (j *View) render(idx int) {
	j.selection = idx
	j.clearView()
	j.addCrumbs()
	j.addViewer()
}

// NewPusher returns a link which, when clicked, pushes the given object
// onto the stack.
func (j *View) NewPusher(label string, name string, obj interface{}, fmts *Formatters) interface{} {
	a := dom.NewA()
	a.Set("href", "#")
	a.Append(label)
	a.OnClick(func(m dom.MouseEvent) {
		j.Push(name, obj, fmts)
		m.PreventDefault()
	})
	return a
}

func (j *View) buildObjectView(path string, obj interface{}, fmts *Formatters) interface{} {
	if ok, res := fmts.apply(path, obj); ok {
		return res
	}

	if r, ok := obj.(Representer); ok {
		obj = r.Representation()
	}

	val := reflect.ValueOf(obj)
	switch val.Kind() {
	case reflect.Func:
		return j.buildObjectView(path, val.Call([]reflect.Value{})[0].Interface(), fmts)
	case reflect.Ptr:
		return j.buildObjectView(path, val.Elem().Interface(), fmts)
	case reflect.Struct:
		m := map[string]interface{}{}
		for i := 0; i < val.NumField(); i++ {
			field := val.Type().Field(i)
			v := val.Field(i)
			if v.CanInterface() {
				m[field.Name] = v.Interface()
			}
		}
		return j.buildObjectView(path, m, fmts)
	case reflect.Map:
		table := dom.NewTable()
		for _, key := range val.MapKeys() {
			k := key.Interface()
			v := val.MapIndex(key).Interface()
			r := dom.NewTr()
			td := dom.NewTd()
			td.Append(fmt.Sprintf("%s", k))
			r.Append(td)
			td = dom.NewTd()
			td.Append(j.buildObjectView(fmt.Sprintf("%s/%s", path, k), v, fmts))
			r.Append(td)
			table.Append(r)
		}
		return table
	case reflect.Slice, reflect.Array:
		table := dom.NewTable()
		for i := 0; i < val.Len(); i++ {
			v := val.Index(i).Interface()
			r := dom.NewTr()
			td := dom.NewTd()
			td.Append(j.buildObjectView(fmt.Sprintf("%s/%d", path, i), v, fmts))
			r.Append(td)
			table.Append(r)
		}
		return table
	default:
		return obj
	}
}
