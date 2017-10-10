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
	"os"
	"sync"
	"syscall"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/pkg/errors"
)

var (
	// Name is the full name of the application
	Name string
	// ExitFuncForTesting can be set to change the behaviour when there is a command line parsing failure.
	// It defaults to os.Exit
	ExitFuncForTesting = os.Exit
	// ShortHelp should be set to add a help message to the usage text.
	ShortHelp = ""
	// ShortUsage is usage text for the additional non-flag arguments.
	ShortUsage = ""
	// UsageFooter is printed at the bottom of the usage text
	UsageFooter = ""
	// Version holds the version specification for the application.
	// The default version is the one defined in version.cmake.
	// If valid a command line option to report it will be added automatically.
	Version VersionSpec
	// Restart is the error to return to cause the app to restart itself.
	Restart = fault.Const("Restart")
)

// VersionSpec is the structure for the version of an application.
type VersionSpec struct {
	// Major version, the version structure is in valid if <0
	Major int
	// Minor version, not used if <0
	Minor int
	// Point version, not used if <0
	Point int
	// The build identifier, not used if an empty string
	Build string
}

// IsValid reports true if the VersionSpec is valid, ie it has a Major version.
func (v VersionSpec) IsValid() bool {
	return v.Major >= 0
}

// GreaterThan returns true if v is greater than o.
func (v VersionSpec) GreaterThan(o VersionSpec) bool {
	switch {
	case v.Major > o.Major:
		return true
	case v.Major < o.Major:
		return false
	case v.Minor > o.Minor:
		return true
	case v.Minor < o.Minor:
		return false
	case v.Point > o.Point:
		return true
	default:
		return false
	}
}

// Format implements fmt.Formatter to print the version.
func (v VersionSpec) Format(f fmt.State, c rune) {
	fmt.Fprint(f, v.Major)
	if v.Minor >= 0 {
		fmt.Fprint(f, ".", v.Minor)
	}
	if v.Point >= 0 {
		fmt.Fprint(f, ".", v.Point)
	}
	if v.Build != "" {
		fmt.Fprint(f, ":", v.Build)
	}
}

func init() {
	Name = file.Abs(os.Args[0]).Basename()
}

// Run performs all the work needed to start up an application.
// It parsers the main command line arguments, builds a primary context that will be cancelled on exit
// runs the provided task, cancels the primary context and then waits for either the maximum shutdown delay
// or all registered signals whichever comes first.
func Run(main task.Task) {
	crash.Register(onCrash)

	// Defer the panic handling
	defer func() {
		switch cause := recover().(type) {
		case nil:
		case ExitCode:
			ExitFuncForTesting(int(cause))
		default:
			crash.Crash(cause)
		}
	}()
	flags := &AppFlags{Log: logDefaults()}

	// install all the common application flags
	rootCtx := prepareContext(&flags.Log)

	// parse the command line
	flag.CommandLine.Usage = func() { Usage(rootCtx, "") }
	verbMainPrepare(flags)
	globalVerbs.flags.Parse(os.Args[1:]...)

	// Force the global verb's flags back into the default location for
	// main programs that still look in flag.Args()
	// TODO: We need to stop doing this
	globalVerbs.flags.ForceCommandLine()

	if flags.DecodeStack != "" {
		stack := decodeCrashCode(flags.DecodeStack)
		fmt.Fprintf(os.Stdout, "Stack dump:\n%v\n", stack)
		return
	}

	if flags.Version {
		fmt.Fprint(os.Stdout, Name, " version ", Version, "\n")
		return
	}

	endProfile := applyProfiler(rootCtx, &flags.Profile)

	ctx, cancel := task.WithCancel(rootCtx)

	ctx = updateContext(ctx, &flags.Log)

	// Defer the shutdown code
	shutdownOnce := sync.Once{}
	shutdown := func() {
		shutdownOnce.Do(func() {
			LogHandler.Close()
			cancel()
			if !WaitForCleanup(rootCtx) {
				fmt.Fprint(os.Stderr, "Timeout waiting for cleanup")
			}
			endProfile()
		})
	}

	defer shutdown()

	// Add the abort and crash signal handlers
	handleAbortSignals(shutdown)
	handleCrashSignals(shutdown)

	// Now we are ready to run the main task
	err := main(ctx)
	if errors.Cause(err) == Restart {
		err = doRestart()
	}
	if err != nil {
		log.F(ctx, "Main failed\nError: %v", err)
	}
}

func doRestart() error {
	argv0 := file.ExecutablePath().System()
	files := make([]*os.File, syscall.Stderr+1)
	files[syscall.Stdin] = os.Stdin
	files[syscall.Stdout] = os.Stdout
	files[syscall.Stderr] = os.Stderr
	wd, err := os.Getwd()
	if nil != err {
		return err
	}
	_, err = os.StartProcess(argv0, os.Args, &os.ProcAttr{
		Dir:   wd,
		Env:   os.Environ(),
		Files: files,
		Sys:   &syscall.SysProcAttr{},
	})
	return err
}
