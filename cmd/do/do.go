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

// The do command wraps CMake, simplifying the building GAPID in common
// configurations.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
)

const (
	cfgPath = ".gapid-config"

	// Changes to this version will force a full clean build.
	// This is useful for situations where the CMake flags have changed and
	// regenerating the files is required.
	versionMajor = 1
	versionMinor = 0 // For future use.
)

var (
	timeDo = flag.Bool("time", false, "Time the action")

	// Root of the GAPID source tree.
	srcRoot file.Path

	// Path to the GO executable found on PATH.
	goExePath file.Path

	// Extension to use for host executables.
	hostExeExt string
)

const (
	RunTests TestMode = iota
	BuildTests
	DisableTests
)

type (
	TestMode uint8

	InitOptions struct {
		Force bool `help:"Force init to redo steps that have already been done"`
	}
	ConfigOptions struct {
		Reset       bool `help:"Reset configuration to the defaults"`
		Interactive bool `help:"Interactive mode"`
	}
	BuildOptions struct {
		Rescan   bool     `help:"Re-run build configuration"`
		DryRun   bool     `help:"Do the build in dry run mode"`
		Verbose  bool     `help:"Do the build in verbose mode"`
		Install  bool     `help:"Run the install step after building"`
		NumJobs  int      `help:"Number of jobs passed to ninja"`
		Test     TestMode `help:"Control the test mode"`
		BuildNum int      `help:"The build number to use for the package"`
		BuildSha string   `help:"The commit sha ot use for the package"`
	}
	RunOptions struct {
		WD file.Path `help:"_Path to use as current working directory"`
	}
	BuildAndRunOptions struct {
		BuildOptions
		RunOptions
	}
	CleanOptions struct {
		Generated bool `help:"Delete generatd files in the source tree"`
	}
	GapitOptions struct {
		BuildAndRunOptions
	}
	RobotOptions struct {
		BuildAndRunOptions
		ServerAddress string `help:"The master server address"`
	}
	UploadOptions struct {
		RobotOptions
		CL           string `help:"The build CL, will be guessed if not set"`
		Description  string `help:"An optional build description"`
		Tag          string `help:"The optional build tag"`
		Track        string `help:"The package's track, will be guessed if not set"`
		Uploader     string `help:"The uploading entity, will be guessed if not set"`
		BuilderAbi   string `help:"The abi of the builder device, will assume this device if not set"`
		ArtifactPath string `help:"The file path where the zipped artifact will be stored"`
	}
	initVerb   struct{ InitOptions }
	configVerb struct{ ConfigOptions }
	buildVerb  struct{ BuildOptions }
	globVerb   struct{}
	cleanVerb  struct{ CleanOptions }
	runVerb    struct{ BuildAndRunOptions }
	gapitVerb  struct{ GapitOptions }
	robotVerb  struct{ RobotOptions }
	uploadVerb struct{ UploadOptions }
	goVerb     struct{ RunOptions }
	jdocVerb   struct{ BuildOptions }
)

func findRootSourcePath() file.Path {
	f := func() string {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			panic("Cannot find directory of do.go")
		}
		return file
	}()
	return file.Abs(f).Parent().Parent().Parent()
}

func checkGoVersion() {
	var major, minor, point int
	fmt.Sscanf(runtime.Version(), "go%d.%d.%d", &major, &minor, &point)
	if major != 1 || minor < 8 {
		fmt.Fprintf(os.Stderr, "Requires Go version greater than 1.8.x, got %v.%v.%v", major, minor, point)
		os.Exit(1)
	}
}

func init() {
	if runtime.GOOS == "windows" {
		hostExeExt = ".exe"
	}

	checkGoVersion()

	srcRoot = findRootSourcePath()
	if path, err := file.FindExecutable("go"); err != nil {
		panic("go executable not found on PATH")
	} else {
		goExePath = path
	}

	app.AddVerb(&app.Verb{
		Name:      "init",
		ShortHelp: "initialise all pre-requisites to build gapid",
		Action:    &initVerb{InitOptions: InitOptions{Force: true}},
	})
	app.AddVerb(&app.Verb{
		Name:      "config",
		ShortHelp: "set configuration parameters",
		Action:    &configVerb{ConfigOptions: ConfigOptions{Interactive: true}},
	})
	app.AddVerb(&app.Verb{
		Name:      "build",
		ShortHelp: "start a build of the optional target",
		Action:    &buildVerb{},
	})
	app.AddVerb(&app.Verb{
		Name:      "glob",
		ShortHelp: "update CMakeFiles.cmake",
		Action:    &globVerb{},
	})
	app.AddVerb(&app.Verb{
		Name:      "clean",
		ShortHelp: "delete the output directory",
		Action:    &cleanVerb{},
	})
	app.AddVerb(&app.Verb{
		Name:      "run",
		ShortHelp: "build and run a target",
		Action:    &runVerb{},
	})
	app.AddVerb(&app.Verb{
		Name:      "gapit",
		ShortHelp: "build and run gapit",
		Action:    &gapitVerb{},
	})
	app.AddVerb(&app.Verb{
		Name:      "robot",
		ShortHelp: "build and run robot",
		Action:    &robotVerb{},
	})
	app.AddVerb(&app.Verb{
		Name:      "upload",
		ShortHelp: "build gapid package and upload to robot",
		Action:    &uploadVerb{},
	})
	app.AddVerb(&app.Verb{
		Name:      "go",
		ShortHelp: "run the go tool with the correct environment",
		Action:    &goVerb{},
	})
	app.AddVerb(&app.Verb{
		Name:      "jdoc",
		ShortHelp: "generte the gapic javadoc",
		Action:    &jdocVerb{},
	})
}

var gopath string

func main() {
	gopath = os.Getenv("GOPATH")
	if len(gopath) > 0 {
		if s := strings.Split(gopath, string(os.PathListSeparator)); len(s) > 1 {
			gopath = s[0]
		}
	}

	start := time.Now()
	app.ShortHelp = "Do is the build front end for the graphics api debugger system."
	app.Run(app.VerbMain)
	if *timeDo {
		fmt.Printf("Time taken: %v\n", time.Since(start))
	}
}

var testModeNames = map[TestMode]string{
	RunTests:     "run",
	BuildTests:   "build",
	DisableTests: "disable",
}

func (t *TestMode) Choose(c interface{}) { *t = c.(TestMode) }
func (t TestMode) String() string        { return testModeNames[t] }

func closeOnInterrupt(ctx context.Context) {
	crash.Go(func() {
		// Ensure that ctrl-c interrupts actually stop the application.
		// This is caught by the application framework and simply cancels the
		// context so it can shutdown cleanly. For do, we just want to stop
		// immediately.
		<-task.ShouldStop(ctx)
		time.Sleep(time.Second) // Wait a little to let messages get printed.
		os.Exit(0)
	})
}

func (verb *initVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	closeOnInterrupt(ctx)
	doInit(ctx, verb.InitOptions)
	return nil
}

func (verb *configVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	closeOnInterrupt(ctx)
	if flags.NArg() != 0 {
		app.Usage(ctx, "config does not take arguments")
		return nil
	}
	fetchValidConfig(ctx, verb.ConfigOptions)
	return nil
}

func (verb *buildVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	closeOnInterrupt(ctx)
	cfg := doInit(ctx, InitOptions{})
	doBuild(ctx, cfg, verb.BuildOptions, flags.Args()...)
	return nil
}

func (verb *globVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	closeOnInterrupt(ctx)
	if flags.NArg() != 0 {
		app.Usage(ctx, "glob does not take arguments")
		return nil
	}
	cfg := doInit(ctx, InitOptions{})
	doGlob(ctx, cfg)
	return nil
}

func (verb *cleanVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	closeOnInterrupt(ctx)
	if flags.NArg() != 0 {
		app.Usage(ctx, "clean does not take arguments")
		return nil
	}
	cfg := doInit(ctx, InitOptions{})
	doClean(ctx, verb.CleanOptions, cfg)
	return nil
}

func (verb *runVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	closeOnInterrupt(ctx)
	cfg := doInit(ctx, InitOptions{})
	args := flags.Args()
	if len(args) < 1 {
		app.Usage(ctx, "run must be told the target name")
		return nil
	}
	doRunTarget(ctx, cfg, verb.BuildAndRunOptions, args[0], args[1:]...)
	return nil
}

func (verb *gapitVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	closeOnInterrupt(ctx)
	cfg := doInit(ctx, InitOptions{})
	doGapit(ctx, cfg, verb.GapitOptions, flags.Args()...)
	return nil
}

func (verb *robotVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	closeOnInterrupt(ctx)
	cfg := doInit(ctx, InitOptions{})
	doRobot(ctx, cfg, verb.RobotOptions, flags.Args()...)
	return nil
}

func (verb *uploadVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	closeOnInterrupt(ctx)
	cfg := doInit(ctx, InitOptions{})
	doUpload(ctx, cfg, verb.UploadOptions, flags.Args()...)
	return nil
}

func (verb *goVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	closeOnInterrupt(ctx)
	cfg := doInit(ctx, InitOptions{})
	doGo(ctx, cfg, verb.RunOptions, flags.Args()...)
	return nil
}

func (verb *jdocVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	cfg := doInit(ctx, InitOptions{})
	gapic(ctx, cfg).jdoc(ctx, verb.BuildOptions)
	return nil
}

func run(ctx context.Context, cwd file.Path, exe file.Path, env *shell.Env, args ...string) {
	err := shell.
		Command(exe.System(), args...).
		In(cwd.System()).
		Read(os.Stdin).
		Capture(os.Stdout, os.Stderr).
		Env(env).
		Run(ctx)
	if err != nil {
		fmt.Printf("Error running %s %v: %v\n", exe, args, err)
		os.Exit(1)
	}
}

func splitEnv(s string) (key string, vals []string) {
	parts := strings.Split(s, "=")
	if len(parts) != 2 {
		return "", nil
	}
	return parts[0], strings.Split(parts[1], string(os.PathListSeparator))
}

func joinEnv(key string, vals []string) string {
	return key + "=" + strings.Join(vals, string(os.PathListSeparator))
}
