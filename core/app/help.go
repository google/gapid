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
	"fmt"
	"io"
	"os"


	"github.com/google/gapid/core/app/flags"
)

// MakeDocs
func MakeDoc(ctx context.Context, name string){
	filename := name + ".md"
	w, _ := os.Create(filename)
	globalVerbs.selected = nil
	fmt.Fprintln(w, "**Note**: Autogenerated documentation, do not modify!")
	fmt.Fprintln(w, "# GAPIT Help")

	// Print list of verbs
	fmt.Fprintln(w, "| Command | Short help |")
	fmt.Fprintln(w, "| -- | -- |")
	format := fmt.Sprintf("|[%%s](#%%s)|%%s|")
	backlink := "[Back](#gapit-help)"
	for _, child := range globalVerbs.verbs {
		fmt.Fprintf(w, format, child.Name, child.Name, child.ShortHelp)
		fmt.Fprintln(w)
	}

	for _, verb := range globalVerbs.verbs {
		globalVerbs.selected = verb
		// Header
		fmt.Fprintln(w, "# ", verb.Name)
		fmt.Fprintln(w, backlink)
		fmt.Fprintln(w)
		// Usage
		fmt.Fprintf(w, "*%s*", verb.ShortHelp)
		fmt.Fprintln(w)
		fmt.Fprintln(w)
		fmt.Fprint(w, "**Usage:** ")
		verbUsage(w, &globalVerbs, true)
		// Full flags
		fmt.Fprintln(w, "```")
		verbHelp(w, &globalVerbs, true)
		fmt.Fprintln(w, "```")
		fmt.Fprintln(w, backlink)

	}
}

// Usage prints message with the formatting args to stderr, and then prints the command usage information and
// terminates the program.
func Usage(ctx context.Context, message string, args ...interface{}) {
	usage(ctx, message, Flags.FullHelp, args...)
	panic(UsageExit)
}

func usage(ctx context.Context, message string, verbose bool, args ...interface{}) {
	w := os.Stdout
	if len(message) > 0 || len(args) > 0 {
		w = os.Stderr
	}
	if len(message) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, message, args...)
		fmt.Fprintln(w)
		fmt.Fprintln(w)
	} else if len(args) > 0 {
		fmt.Fprintln(w)
		fmt.Fprint(w, args...)
		fmt.Fprintln(w)
		fmt.Fprintln(w)
	}
	verbShorthelp(w, &globalVerbs, verbose)
	fmt.Fprint(w, "Usage: ")
	verbUsage(w, &globalVerbs, verbose)
	verbHelp(w, &globalVerbs, verbose)
	fmt.Fprintf(w, UsageFooter)

	if !verbose {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Some less common flags have been elided. Use -%s to see the full help.\n", flags.FullHelpFlag)
	}
}

func verbShorthelp(raw io.Writer, v *Verb, verbose bool) {
	if v.ShortHelp != "" {
		fmt.Fprintf(raw, "%s: %s", v.Name, v.ShortHelp)
		fmt.Fprintln(raw)
	}
	if v.selected != nil {
		verbShorthelp(raw, v.selected, verbose)
	}
}

func verbUsage(raw io.Writer, v *Verb, verbose bool) {
	fmt.Fprintf(raw, " %s", v.Name)
	if v.flags.HasVisibleFlags(verbose) {
		fmt.Fprintf(raw, " [%s-flags]", v.Name)
	}
	if v.selected != nil {
		verbUsage(raw, v.selected, verbose)
	} else {
		if v.ShortUsage != "" {
			fmt.Fprintf(raw, " %s", v.ShortUsage)
		} else {
			if len(v.verbs) > 0 {
				fmt.Fprint(raw, " verb [args]")
			}
		}
		fmt.Fprintln(raw)
	}
}

func verbHelp(raw io.Writer, v *Verb, verbose bool) {
	if v.flags.HasVisibleFlags(verbose) {
		fmt.Fprintf(raw, "%s-flags:", v.Name)
		fmt.Fprintln(raw)
		fmt.Fprint(raw, v.flags.Usage(verbose))
		fmt.Fprintln(raw)
	}
	if v.selected != nil {
		verbHelp(raw, v.selected, verbose)
	} else if len(v.verbs) > 0 {
		fmt.Fprintf(raw, "%s verbs:", v.Name)
		fmt.Fprintln(raw)
		longest := 0
		for _, child := range v.verbs {
			if longest < len(child.Name) {
				longest = len(child.Name)
			}
		}
		format := fmt.Sprintf("    • %%-%ds - %%s", longest)
		for _, child := range v.verbs {
			fmt.Fprintf(raw, format, child.Name, child.ShortHelp)
			fmt.Fprintln(raw)
		}
	}
}
