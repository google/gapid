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
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/google/gapid/core/app/flags"
)

// Verb holds information about a runnable api command.
type Verb struct {
	Name       string    // The name of the command
	ShortHelp  string    // Help for the purpose of the command
	ShortUsage string    // Help for how to use the command
	Action     Action    // The verb's action. Must be set.
	flags      flags.Set // The command line flags it accepts
	verbs      []*Verb
	selected   *Verb
}

// Action is the interface for verb actions that can be run.
// Exported fields will be exposed as flags for the verb.
// Use the `help` tag to expose a flag description.
type Action interface {
	// Run executes the action.
	Run(ctx context.Context, flags flag.FlagSet) error
}

var globalVerbs Verb

// Add adds a new verb to the supported set, it will panic if a
// duplicate name is encountered.
func (v *Verb) Add(child *Verb) {
	if child.Action != nil {
		child.flags.Bind("", child.Action, "")
	}
	if len(v.Filter(child.Name)) != 0 {
		panic(fmt.Errorf("Duplicate verb name %s", child.Name))
	}
	v.verbs = append(v.verbs, child)
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
func (v *Verb) Invoke(ctx context.Context, args []string) error {
	if len(args) < 1 {
		Usage(ctx, "Must supply a verb to %s", v.Name)
		return nil
	}
	verb := args[0]
	matches := v.Filter(verb)
	for _, verbs := range matches {
		if verbs.Name == verb {
			matches = []*Verb{verbs}
			break
		}
	}

	switch len(matches) {
	case 1:
		v.selected = matches[0]
		v.selected.flags.Parse(&Flags.FullHelp, args[1:]...)
		if Flags.FullHelp {
			Usage(ctx, "")
		}
		if v.selected.Action != nil {
			return v.selected.Action.Run(ctx, v.selected.flags.Raw)
		}
		return v.selected.Invoke(ctx, v.selected.flags.Raw.Args())
	case 0:
		if verb == "help" {
			Usage(ctx, "")
		} else {
			Usage(ctx, "Verb '%s' is unknown", verb)
		}
	default:
		Usage(ctx, "Verb '%s' is ambiguous", verb)
	}
	return nil
}

// AddVerb adds a new verb to the supported set, it will panic if a
// duplicate name is encountered.
// v is returned so the function can be used in a fluent-style.
func AddVerb(v *Verb) *Verb {
	globalVerbs.Add(v)
	return v
}

// FilterVerbs returns the filtered list of verbs who's names match the specified
// prefix.
func FilterVerbs(prefix string) (result []*Verb) {
	return globalVerbs.Filter(prefix)
}

// VerbMain is a task that can be handed to Run to invoke the verb handling system.
func VerbMain(ctx context.Context) error {
	return globalVerbs.Invoke(ctx, globalVerbs.flags.Args())
}

func verbMainPrepare(flags *AppFlags) {
	globalVerbs.Name = Name
	globalVerbs.ShortHelp = ShortHelp
	globalVerbs.ShortUsage = ShortUsage
	globalVerbs.flags.Raw = *flag.CommandLine
	globalVerbs.flags.Bind("", flags, "")
}
