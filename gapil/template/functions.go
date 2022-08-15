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

package template

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/gapil/semantic"
)

const (
	// ErrBadArgumentCount may be returned by a template invocation that has unpaired arguments
	ErrBadArgumentCount = fault.Const("bad argument count, must be in pairs")
	// ErrBadInvokeKeys may be returned by a template invocation that has invalid arguments keys
	ErrBadInvokeKeys = fault.Const("invoke keys must be strings")
)

type Functions struct {
	ctx       context.Context // Held because functions are invoked through templates
	templates *template.Template
	funcs     template.FuncMap
	globals   globalMap
	mappings  *semantic.Mappings
	active    *template.Template
	writer    io.Writer
	basePath  string
	apiFile   string
	api       *semantic.API
	cs        *ConstantSets
	loader    func(filename string) ([]byte, error)
}

type Options struct {
	Dir     string
	APIFile string
	Loader  func(filename string) ([]byte, error)
	Funcs   template.FuncMap
	Globals []string
	Tracer  string
}

// NewFunctions builds a new template management object that can be used to run templates over an API file.
// The apiFile name is used in error messages, and should be the name of the file the api was loaded from.
// loader can be used to intercept file system access from within the templates, specifically used when
// including other templates.
// The functions in funcs are made available to the templates, and can override the functions from this
// package if needed.
func NewFunctions(ctx context.Context, api *semantic.API, mappings *semantic.Mappings, options Options) (*Functions, error) {
	basePath, err := filepath.Abs(options.Dir)
	if err != nil {
		return nil, fmt.Errorf("Could not get absolute path to directory: '%s'. %v", options.Dir, err)
	}
	f := &Functions{
		ctx:       ctx,
		templates: template.New("FunctionHolder"),
		funcs:     template.FuncMap{},
		globals:   globalMap{},
		mappings:  mappings,
		basePath:  basePath,
		apiFile:   options.APIFile,
		api:       api,
		loader:    options.Loader,
	}
	v := reflect.ValueOf(f)
	t := v.Type()
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		r, _ := utf8.DecodeRuneInString(m.Name)
		if unicode.IsUpper(r) {
			c := v.MethodByName(m.Name)
			f.funcs[m.Name] = c.Interface()
		}
	}
	initNodeTypes(f)
	initGlobals(f, options.Globals)
	for k, v := range options.Funcs {
		if gen, ok := v.(func(*Functions) interface{}); ok {
			f.funcs[k] = gen(f)
		} else {
			f.funcs[k] = v
		}
	}
	if options.Tracer != "" {
		pattern := regexp.MustCompile(options.Tracer)
		for n, c := range f.funcs {
			if pattern.MatchString(n) {
				f.funcs[n] = trace(n, c)
			}
		}
	}
	f.templates.Funcs(f.funcs)
	return f, nil
}

func trace(name string, f interface{}) func(values ...interface{}) (interface{}, error) {
	depth := ""
	return func(values ...interface{}) (interface{}, error) {
		fmt.Print(depth, name)
		depth += " |  "
		args := make([]reflect.Value, len(values))
		for i, v := range values {
			if v == nil {
				args[i] = reflect.ValueOf(&v).Elem()
			} else {
				args[i] = reflect.ValueOf(v)
			}
			switch v := v.(type) {
			case string:
				fmt.Printf(" %q", v)
			case semantic.Type:
				fmt.Printf(" [%s]", v.Name())
			default:
				fmt.Printf(" <%T>", v)
			}
		}
		fmt.Println()
		defer func() {
			depth = depth[:len(depth)-4]
		}()
		result := reflect.ValueOf(f).Call(args)
		value := result[0].Interface()
		var err error
		if len(result) > 1 {
			err, _ = result[1].Interface().(error)
		}
		return value, err
	}
}

// IsNil returns true if v is nil.
func (f *Functions) IsNil(v interface{}) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map:
		return r.IsNil()
	default:
		return false
	}
}

// Error raises an error terminating execution of the template.
//
//	{{Error "Foo returned error: %s" $err}}
func (f *Functions) Error(s string, args ...interface{}) (string, error) {
	return "", fmt.Errorf(s, args...)
}

// Log prints s and optional format arguments to stdout. Example:
//
//	{{Log "%s %s" "Hello" "world}}
func (f *Functions) Log(s string, args ...interface{}) string {
	fmt.Printf(s+"\n", args...)
	return ""
}

func (*Functions) buildArgs(values ...interface{}) (map[string]interface{}, error) {
	var base map[string]interface{}
	i := 0
	if len(values) > 0 {
		var ok bool
		if base, ok = values[0].(map[string]interface{}); ok {
			i = 1
		}
	}
	pairs := (len(values) - i) / 2
	if (len(values)-i)%2 != 0 {
		return nil, ErrBadArgumentCount
	}
	data := make(map[string]interface{}, pairs+len(base))
	for k, v := range base {
		data[k] = v
	}
	for ; i < len(values)-1; i += 2 {
		switch k := values[i].(type) {
		case string:
			data[k] = values[i+1]
		default:
			return nil, ErrBadInvokeKeys
		}
	}
	return data, nil
}

// Args builds a template argument object from a list of arguments.
// If no arguments are passed then the result will be nil.
// If a single argument is passed then the result will be the value
// of that argument.
// If the first argument is a map, it is assumed to be a base argument set to
// be augmented.
// Remaining arguments must come in name-value pairs.
// For example:
//
//	{{define "SingleParameterMacro"}}
//	    $ is: {{$}}
//	{{end}}
//
//	{{define "MultipleParameterMacro"}}
//	    $.ArgA is: {{$.ArgA}}, $.ArgB is: {{$.ArgB}}
//	{{end}}
//
//	{{template "SingleParameterMacro" (Args)}}
//	{{/* Returns "$ is: nil" */}}
//
//	{{template "SingleParameterMacro" (Args 42)}}
//	{{/* Returns "$ is: 42" */}}
//
//	{{template "MultipleParameterMacro" (Args "ArgA" 4 "ArgB" 2)}}
//	{{/* Returns "$.ArgA is: 4, $.ArgB is: 2" */}}
func (f *Functions) Args(arguments ...interface{}) (interface{}, error) {
	switch len(arguments) {
	case 0:
		return nil, nil
	case 1:
		return arguments[0], nil
	default:
		return f.buildArgs(arguments...)
	}
}

// Macro invokes the template macro with the specified name and returns the
// template output as a string. See Args for how the arguments are processed.
func (f *Functions) Macro(name string, arguments ...interface{}) (string, error) {
	arg, err := f.Args(arguments...)
	if err != nil {
		return "", err
	}
	t := f.templates.Lookup(name)
	if t == nil {
		return "", fmt.Errorf("Cannot find template %s", name)
	}
	buf := &bytes.Buffer{}
	err = f.execute(t, buf, arg)
	return buf.String(), err
}

// Template invokes the template with the specified name writing the output to
// the current output writer. See Args for how the arguments are processed.
func (f *Functions) Template(name string, arguments ...interface{}) (string, error) {
	arg, err := f.Args(arguments...)
	if err != nil {
		return "", err
	}
	t := f.templates.Lookup(name)
	if t == nil {
		return "", fmt.Errorf("Cannot find template %s", name)
	}
	return "", f.execute(t, nil, arg)
}

func nodename(node interface{}) string {
	if node == nil {
		return "Nil"
	}
	nt := reflect.TypeOf(node)
	if nt.Kind() == reflect.Ptr {
		nt = nt.Elem()
	}
	return nt.Name()
}

func (f *Functions) node(writer io.Writer, prefix string, node semantic.Node, arguments ...interface{}) error {
	// Collect the arguments to the template
	args, err := f.buildArgs(arguments...)
	if err != nil {
		return err
	}
	args["Node"] = node
	try := make([]string, 0, 4)
	ty, err := f.TypeOf(node)
	if node != nil && err == nil {
		args["Type"] = ty
		try = append(try,
			prefix+"#"+ty.Name(),
			prefix+"."+nodename(ty),
		)
	}
	if node != ty {
		try = append(try, prefix+"."+nodename(node))
	}
	try = append(try, prefix+"_Default")
	for _, name := range try {
		if tmpl := f.templates.Lookup(name); tmpl != nil {
			return f.execute(tmpl, writer, args)
		}
	}
	return fmt.Errorf(`Cannot find templates "%s"`, strings.Join(try, `","`))
}

// Node dispatches to the template that matches the node best, writing the
// result to the current output writer.
// If the node is a Type or Expression then the type semantic.Type name is tried,
// then the class of type (the name of the semantic class that represents the type).
// The actual name of the node type is then tried, and if none of those matches,
// the "Default" template is used if present.
// If no possible template could be matched, and error is generated.
// eg: {{Node "TypeName" $}} where $ is a boolean and expression would try
//
//	"TypeName#Bool"
//	"TypeName.Builtin"
//	"TypeName.BinaryOp"
//	"TypeName_Default"
//
// See Args for how the arguments are processed, in addition the Node arg will
// be added in and have the value of node, and if the node had a type
// discovered, the Type arg will be added in as well.
func (f *Functions) Node(prefix string, node semantic.Node, arguments ...interface{}) (string, error) {
	return "", f.node(nil, prefix, node, arguments...)
}

// SNode dispatches to the template that matches the node best, capturing the
// result and returning it.
// See Node for the dispatch rules used.
func (f *Functions) SNode(prefix string, node semantic.Node, arguments ...interface{}) (string, error) {
	buf := &bytes.Buffer{}
	err := f.node(buf, prefix, node, arguments...)
	return buf.String(), err
}
