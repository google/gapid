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

package flags

import (
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	// FullHelpFlag is the name of the flag used to show the full help.
	FullHelpFlag = "fullhelp"
)

type (
	Set struct {
		// Raw is the underlying flag set
		// TODO: hide this once we stop things relying on it
		Raw flag.FlagSet
	}
)

type U64Slice []uint64

func (i *U64Slice) String() string {
	return fmt.Sprintf("%d", *i)
}

func (i *U64Slice) Set(v string) error {
	*i = make(U64Slice, 0)
	if v[0] != '[' || v[len(v)-1] != ']' {
		tmp, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("Expected '[n, ...]' or 'n', could not parse %s", v)
		}
		*i = append(*i, uint64(tmp))
		return nil
	}
	x := strings.Split(v[1:len(v)-1], ",")

	for _, val := range x {
		tmp, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		if err != nil {
			return fmt.Errorf("Could not parse slice %s", val)
		}
		*i = append(*i, uint64(tmp))
	}
	return nil
}

type StringSlice []string

func (i *StringSlice) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *StringSlice) Set(v string) error {
	*i = make(StringSlice, 0)
	if v[0] != '[' || v[len(v)-1] != ']' {
		*i = append(*i, v)
		return nil
	}
	x := strings.Split(v[1:len(v)-1], ",")

	for _, val := range x {
		*i = append(*i, strings.TrimSpace(val))
	}
	return nil
}

func bindName(prefix string, name string, partial string, full string) string {
	if partial != "" {
		name = partial
	}
	if full != "" {
		return full
	}
	if prefix == "" {
		return name
	}
	if name == "" {
		return prefix
	}
	return prefix + "-" + name
}

// Bind uses reflection to bind flag values to the verb.
// It will recurse into nested structures adding all leaf fields.
func (s *Set) Bind(name string, value interface{}, help string) {
	switch val := value.(type) {
	case *bool:
		s.Raw.BoolVar(val, name, *val, help)
		return
	case *int:
		s.Raw.IntVar(val, name, *val, help)
		return
	case *int64:
		s.Raw.Int64Var(val, name, *val, help)
		return
	case *uint:
		s.Raw.UintVar(val, name, *val, help)
		return
	case *uint64:
		s.Raw.Uint64Var(val, name, *val, help)
		return
	case *float64:
		s.Raw.Float64Var(val, name, *val, help)
		return
	case *string:
		s.Raw.StringVar(val, name, *val, help)
		return
	case *time.Duration:
		s.Raw.DurationVar(val, name, *val, help)
		return
	case Choosable:
		chooser := val.Chooser()
		s.Raw.Var(chooser, name, fmt.Sprintf("%s [one of: %s]", help, chooser.Choices))
		return
	case Enum:
		chooser := ForEnum(val)
		s.Raw.Var(chooser, name, fmt.Sprintf("%s [one of: %s]", help, chooser.Choices))
		return
	case flag.Value:
		s.Raw.Var(val, name, help)
		return
	}
	rv := reflect.ValueOf(value)

	if rv.Kind() != reflect.Ptr {
		panic(fmt.Sprint("Flag value not a pointer: %v", rv.Type()))
	}

	switch e := rv.Elem(); e.Kind() {
	case reflect.Slice:
		s.Raw.Var(newRepeatedFlag(e), name, help)
	case reflect.Struct:
		t := e.Type()
		for i := 0; i < e.NumField(); i++ {
			tf := t.Field(i)
			if tf.PkgPath != "" {
				continue // Unexported.
			}
			field := e.Field(i)
			if !field.CanSet() {
				panic(fmt.Sprintf("Unsettable field %q : %v", tf.Name, field.Type()))
			}
			tags := tf.Tag
			fname := strings.ToLower(tf.Name)
			fullname := tags.Get("fullname")
			partialname := tags.Get("name")
			usage := tags.Get("help")
			if tf.Anonymous {
				fname = ""
			}
			if partialname != "" {
				fname = partialname
			}
			switch {
			case fullname != "":
				// all done
			case fname == "":
				fullname = name
			case name == "":
				fullname = fname
			default:
				fullname = name + "-" + fname
			}
			s.Bind(fullname, field.Addr().Interface(), usage)
		}
	default:
		panic(fmt.Sprintf("Unhandled flag type: %v", rv.Type()))
	}
}

// HasVisibleFlags returns true if the set has bound flags for the specified verbosity.
func (s *Set) HasVisibleFlags(verbose bool) bool {
	result := false
	s.Raw.VisitAll(func(f *flag.Flag) {
		if _, _, hidden := getFlagUsage(f, verbose); !hidden {
			result = true
		}
	})
	return result
}

func getFlagUsage(f *flag.Flag, verbose bool) (string, string, bool) {
	name, usage := flag.UnquoteUsage(f)
	forceHide := f.Name == FullHelpFlag
	if !strings.HasPrefix(usage, "_") {
		return name, usage, forceHide
	}
	return name, usage[1:], forceHide || !verbose
}

func dumpDefault(fl *flag.Flag) string {
	typ := reflect.TypeOf(fl.Value)
	var z reflect.Value
	if typ.Kind() == reflect.Ptr {
		z = reflect.New(typ.Elem())
	} else {
		z = reflect.Zero(typ)
	}
	value := z.Interface().(flag.Value)
	isString := false
	if getter, isGetter := value.(flag.Getter); isGetter {
		_, isString = getter.Get().(string)
	}
	if fl.DefValue == value.String() {
		return ""
	}
	switch fl.DefValue {
	case "false":
		return ""
	case "":
		return ""
	case "0":
		return ""
	}
	if isString {
		return fmt.Sprintf(" (default %q)", fl.DefValue)
	}
	return fmt.Sprintf(" (default %v)", fl.DefValue)
}

// Usage returns the usage string for the flags.
func (s *Set) Usage(verbose bool) string {
	result := ""
	s.Raw.VisitAll(func(fl *flag.Flag) {
		name, usage, hidden := getFlagUsage(fl, verbose)
		if hidden {
			return
		}
		if result != "" {
			result += "\n"
		}
		result += fmt.Sprintf("  -%s %s\n\t", fl.Name, name)
		result += usage
		result += dumpDefault(fl)
	})
	return result
}

// Parse processes the args to fill in the flags.
// see flag.Parse for more details.
func (s *Set) Parse(fullHelp *bool, args ...string) {
	if fullHelp != nil && s.Raw.Lookup(FullHelpFlag) == nil {
		s.Raw.BoolVar(fullHelp, FullHelpFlag, *fullHelp, "")
	}
	s.Raw.Usage = flag.CommandLine.Usage
	s.Raw.Parse(args)
}

// Args returns the unprocessed part of the command line passed to Parse.
func (s *Set) Args() []string {
	return s.Raw.Args()
}

// ForceCommandLine is an ugly hack for things that try to directly use the flag package
// TODO: remove this
func (s *Set) ForceCommandLine() {
	flag.CommandLine = &s.Raw
}
