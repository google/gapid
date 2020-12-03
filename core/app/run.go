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
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/gapid/core/app/analytics"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/crash/reporting"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/text"
	"github.com/pkg/errors"
)

const (
	exitSuccess = 0
	exitFailure = 1
)

var (
	// Name is the full name of the application
	Name string
	// Flags is the main application flags.
	Flags AppFlags
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

// VersionSpecFromTag parses the version from a git tag name.
func VersionSpecFromTag(tag string) (VersionSpec, error) {
	var version VersionSpec
	numFields, _ := fmt.Sscanf(tag, "v%d.%d.%d-%s", &version.Major, &version.Minor, &version.Point, &version.Build)
	if numFields < 3 {
		return version, fmt.Errorf("Failed to parse version tag: %s", tag)
	}
	return version, nil
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

// GreaterThanDevVersion returns true if v is greater than o, respecting dev versions.
func (v VersionSpec) GreaterThanDevVersion(o VersionSpec) bool {
	if v.GreaterThan(o) {
		return true
	}
	if !v.Equal(o) {
		return false
	}

	// dev-releases are previews of the next release, so e.g. 1.2.3 is more recent than 1.2.3-dev-456
	vDev, oDev := v.GetDevVersion(), o.GetDevVersion()
	return oDev >= 0 && (vDev < 0 || vDev > oDev)
}

// Equal returns true if v is equal to o, ignoring the Build field.
func (v VersionSpec) Equal(o VersionSpec) bool {
	return (v.Major == o.Major) && (v.Minor == o.Minor) && (v.Point == o.Point)
}

func (v VersionSpec) String() string {
	return fmt.Sprintf("%v", v)
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

// GetDevVersion returns the dev version, or -1 if no dev version is found.
func (v VersionSpec) GetDevVersion() int {
	// A dev release has a Build string prefixed by "dev-YYYYMMDD-",
	// where YYYYMMDD is the dev version.
	if strings.HasPrefix(v.Build, "dev-") && len(v.Build) >= 12 {
		if devVersion, err := strconv.Atoi(v.Build[4:12]); err == nil {
			return devVersion
		}
	}
	return -1
}

func init() {
	Name = file.Abs(os.Args[0]).Basename()
	Flags.Log = logDefaults()
	// TODO(awoloszyn): Figure out why object churn is soo bad, and try to
	//                  minimize it.
	//                  At that point we can remove this.
	debug.SetGCPercent(40)
}

// getArgs preprocesses the command line arguments to
// split out any packed arguments passed in as a string.
func getArgs() []string {
	// parse the command line
	args := os.Args[1:]
	for {
		expanded := false
		newArgs := []string{}
		for i := 0; i < len(args); i++ {
			arg := args[i]
			if (arg == "-args" ||
				arg == "--args") &&
				i != len(args)-1 {
				newArgs = append(newArgs, text.SplitArgs(args[i+1])...)
				i++
				expanded = true
			} else {
				newArgs = append(newArgs, arg)
			}
		}
		args = newArgs
		if !expanded {
			return args
		}
	}
}

// Run wraps doRun in order to let doRun use deferred functions. This
// is because os.Exit does not execute deferred functions.
func Run(main task.Task) {
	os.Exit(doRun(main))
}

// doRun performs all the work needed to start up an application.
// It parsers the main command line arguments, builds a primary context that will be cancelled on exit
// runs the provided task, cancels the primary context and then waits for either the maximum shutdown delay
// or all registered signals whichever comes first.
func doRun(main task.Task) int {
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

	// If run via 'bazel run', use the shell's CWD, not bazel's.
	if cwd := os.Getenv("BUILD_WORKING_DIRECTORY"); cwd != "" {
		os.Chdir(cwd)
	}

	// install all the common application flags
	rootCtx := prepareContext(&Flags.Log)

	args := getArgs()

	flag.CommandLine.Usage = func() { Usage(rootCtx, "") }
	verbMainPrepare(&Flags)
	globalVerbs.flags.Parse(nil, args...)

	// Force the global verb's flags back into the default location for
	// main programs that still look in flag.Args()
	// TODO: We need to stop doing this
	globalVerbs.flags.ForceCommandLine()

	if Flags.FullHelp {
		Usage(rootCtx, "")
		return exitSuccess
	}

	if Flags.DecodeStack != "" {
		stack := decodeCrashCode(Flags.DecodeStack)
		fmt.Fprintf(os.Stdout, "Stack dump:\n%v\n", stack)
		return exitSuccess
	}

	if Flags.Version {
		fmt.Fprint(os.Stdout, Name, " version ", Version, "\n")
		return exitSuccess
	}

	endProfile := applyProfiler(rootCtx, &Flags.Profile)

	ctx, cancel := task.WithCancel(rootCtx)

	ctx = updateContext(ctx, &Flags.Log)

	if Flags.Analytics != "" {
		analytics.Enable(ctx, Flags.Analytics, analytics.AppVersion{
			Name: Name, Build: Version.Build,
			Major: Version.Major, Minor: Version.Minor, Point: Version.Point,
		})
		analytics.SendEvent("app", "start", Name)
	}

	if Flags.CrashReport {
		reporting.Enable(ctx, Name, Version.String())
	}

	if Flags.Log.Status {
		status.RegisterLogger(time.Second)
	}

	if Flags.Profile.Trace != "" {
		f, err := os.Create(Flags.Profile.Trace)
		if err != nil {
			log.E(ctx, "Could not start trace profiling")
		} else {
			status.RegisterTracer(f)
			defer f.Close()
		}
	}

	// Defer the shutdown code
	shutdownOnce := sync.Once{}
	shutdown := func() {
		shutdownOnce.Do(func() {
			analytics.Flush()
			cancel()
			if !WaitForCleanup(rootCtx) {
				log.E(ctx, "Timeout waiting for cleanup")
			}
			endProfile()
			LogHandler.Close()
		})
	}

	defer shutdown()

	// Add the abort and crash signal handlers
	handleAbortSignals(ctx, shutdown)
	handleCrashSignals(ctx, shutdown)

	// Now we are ready to run the main task
	err := main(ctx)
	if errors.Cause(err) == Restart {
		err = doRestart()
	}
	if err != nil {
		log.E(ctx, "Main failed\nError: %v", err)
		return exitFailure
	}
	return exitSuccess
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
