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
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type (
	// Choice is the interface to something that represents a value for an enumerated type.
	Choice interface {
		String() string
	}

	// Choices is a slice of Enum values for a given type.
	Choices []Choice

	// Enum is the interface to a enumerated value.
	// Anything that implements Enum is direclty usesable as a flag value.
	Enum interface {
		// String represents the current value in a string form, used for choice matching.
		String() string
		// Choose is handed a value to set from the Choices list for the enum.
		Choose(interface{})
	}

	// Choosable is the interface to type that can return a chooser for itself.
	// Anything that implements Choosable is direclty usesable as a flag value.
	Choosable interface {
		// Chooser returns a Chooser object that can be used to select values for the implementing type.
		Chooser() Chooser
	}

	// Chooser is is used to select from amongst a set of choices.
	// It conforms to the flag.Value interface.
	Chooser struct {
		// Value is the value we are choosing for.
		Value Enum
		// Choices is the full set of choices available.
		Choices Choices
	}
)

// String returns the string form of the current value.
func (c Chooser) String() string {
	if c.Value == nil {
		return ""
	}
	return c.Value.String()
}

// Set chooses the choice that matches the string.
func (c Chooser) Set(value string) error {
	for _, e := range c.Choices {
		name := e.String()
		if strings.EqualFold(name, value) {
			c.Value.Choose(e)
			return nil
		}
	}
	return fmt.Errorf("Unknown value %q, valid options are: %s", value, c.Choices)
}

// String returns the full set of options as a comma delimited string.
func (c Choices) String() string {
	var b bytes.Buffer
	for _, e := range c {
		if b.Len() > 0 {
			fmt.Fprint(&b, ", ")
		}
		fmt.Fprintf(&b, "%q", e.String())
	}
	return b.String()
}

// Len returns the number of choices available.
func (c Choices) Len() int { return len(c) }

// Swap does an in place swap of two items on the page.
func (c Choices) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

// Less can be used to form a total ordering of the choices by their name.
func (c Choices) Less(i, j int) bool {
	return strings.Compare(c[i].String(), c[j].String()) < 0
}

// ForEnum automatically builds a Chooser for a 0 based sequential enumerated
// integer type up to 1000.
// It expects that the values will all have names from the first to the last, starting at 0,
// and that after the end of the valid range the string method for the type will either return the
// empty string or strconv.Itoa of the base value.
func ForEnum(v Enum) Chooser {
	t := reflect.ValueOf(v).Elem().Type()
	c := Chooser{Value: v}
	for i := 0; i < 1000; i++ {
		ptr := reflect.New(t).Elem()
		switch ptr.Kind() {
		default:
			panic("Invalid enum kind")
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			ptr.SetInt(int64(i))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			ptr.SetUint(uint64(i))
		}
		e := ptr.Interface().(Choice)
		name := e.String()
		if name == strconv.Itoa(i) || name == "" {
			return c
		}
		c.Choices = append(c.Choices, e)
	}
	return c
}
