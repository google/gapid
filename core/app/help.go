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
	"fmt"
	"io"

	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/log"
)

// Usage prints message with the formatting args to stderr, and then prints the command usage information and
// terminates the program.
func Usage(ctx log.Context, message string, args ...interface{}) {
	usage(ctx, message, false, args...)
	panic(UsageExit)
}

func usage(ctx log.Context, message string, verbose bool, args ...interface{}) {
	level := severity.Notice
	if len(message) > 0 || len(args) > 0 {
		level = severity.Error
	}
	raw := ctx.Severity(level).Raw("").Writer()
	defer raw.Close()
	if len(message) > 0 {
		fmt.Fprintln(raw)
		fmt.Fprintf(raw, message, args...)
		fmt.Fprintln(raw)
		fmt.Fprintln(raw)
	} else if len(args) > 0 {
		fmt.Fprintln(raw)
		fmt.Fprint(raw, args...)
		fmt.Fprintln(raw)
		fmt.Fprintln(raw)
	}
	verbShorthelp(raw, &globalVerbs, verbose)
	fmt.Fprint(raw, "Usage: ")
	verbUsage(raw, &globalVerbs, verbose)
	verbHelp(raw, &globalVerbs, verbose)
	fmt.Fprintf(raw, UsageFooter)
}

func autoHelp(ctx log.Context, args ...string) {
	if len(args) == 0 {
		usage(ctx, "", false)
		panic(SuccessExit)
	}
	usage(ctx, "full help", true)
	panic(UsageExit)
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
	if v.Flags.HasVisibleFlags(verbose) {
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
	if v.Flags.HasVisibleFlags(verbose) {
		fmt.Fprintf(raw, "%s-flags:", v.Name)
		fmt.Fprintln(raw)
		fmt.Fprint(raw, v.Flags.Usage(verbose))
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
