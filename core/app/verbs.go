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

package app

import (
	"flag"
	"fmt"
	"strings"

	"github.com/google/gapid/core/app/flags"
	"github.com/google/gapid/core/log"
)

// Verb holds information about a runnable api command.
type Verb struct {
	Name       string                                          // The name of the command
	Run        func(ctx log.Context, flags flag.FlagSet) error // the action for the command
	Auto       AutoVerb                                        // If set, the Run and Flags will be automatically filled from this.
	ShortHelp  string                                          // Help for the purpose of the command
	ShortUsage string                                          // Help for how to use the command
	Flags      flags.Set                                       // The command line flags it accepts
	verbs      []*Verb
	selected   *Verb
}

// AutoVerb is the interface for objects that want to
// automatically configure a verb.
type AutoVerb interface {
	// Run is the method to perform the action associated with a verb.
	// See Verb.Run for more details.
	Run(ctx log.Context, flags flag.FlagSet) error
}

var (
	globalVerbs Verb
)

// Add adds a new verb to the supported set, it will panic if a
// duplicate name is encountered.
func (v *Verb) Add(child *Verb) {
	if child.Auto != nil {
		child.Flags.Bind("", child.Auto, "")
		if child.Run == nil {
			child.Run = child.Auto.Run
		}
	}
	if len(v.Filter(child.Name)) != 0 {
		panic(fmt.Errorf("Duplicate verb name %s", child.Name))
	}
	v.verbs = append(v.verbs, child)
	if v.Run == nil {
		v.Run = v.run
	}
}

// Filter returns the filtered list of verbs who's names match the specified prefix.
func (v *Verb) Filter(prefix string) (result []*Verb) {
	for _, child := range v.verbs {
		if strings.HasPrefix(child.Name, prefix) {
			result = append(result, child)
		}
	}
	return result
}

// Invoke runs a verb, handing it the command line arguments it should process.
func (v *Verb) Invoke(ctx log.Context, args []string) error {
	if len(args) < 1 {
		Usage(ctx, "Must supply a verb to %s", v.Name)
		return nil
	}
	verb := args[0]
	matches := v.Filter(verb)
	switch len(matches) {
	case 1:
		v.selected = matches[0]
		v.selected.Flags.Parse(args[1:]...)
		return v.selected.Run(ctx, v.selected.Flags.Raw)
	case 0:
		if verb == "help" {
			autoHelp(ctx, args[1:]...)
		} else {
			Usage(ctx, "Verb '%s' is unknown", verb)
		}
	default:
		Usage(ctx, "Verb '%s' is ambiguous", verb)
	}
	return nil
}

func (v *Verb) run(ctx log.Context, flags flag.FlagSet) error {
	return v.Invoke(ctx, flags.Args())
}

// AddVerb adds a new verb to the supported set, it will panic if a
// duplicate name is encountered.
func AddVerb(v *Verb) {
	globalVerbs.Add(v)
}

// FilterVerbs returns the filtered list of verbs who's names match the specified
// prefix.
func FilterVerbs(prefix string) (result []*Verb) {
	return globalVerbs.Filter(prefix)
}

// VerbMain is a task that can be handed to Run to invoke the verb handling system.
func VerbMain(ctx log.Context) error {
	return globalVerbs.Invoke(ctx, globalVerbs.Flags.Args())
}

func verbMainPrepare(flags *AppFlags) {
	globalVerbs.Name = Name
	globalVerbs.ShortHelp = ShortHelp
	globalVerbs.ShortUsage = ShortUsage
	globalVerbs.Flags.Raw = *flag.CommandLine
	globalVerbs.Flags.Bind("", flags, "")
}
