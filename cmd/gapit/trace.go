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

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/android/apk"
	"github.com/google/gapid/core/os/process"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/core/vulkan/loader"
	"github.com/google/gapid/gapii/client"
)

type traceVerb struct{ TraceFlags }

func init() {
	verb := &traceVerb{}
	app.AddVerb(&app.Verb{
		Name:      "trace",
		ShortHelp: "Captures a gfx trace from an application",
		Auto:      verb,
	})
}

// These are hard-coded and need to be kept in sync with the api_index
// in the *.api files.
const coreAPI = uint32(1 << 0)
const glesAPI = uint32(1 << 1)
const vulkanAPI = uint32(1 << 2)

func (verb *traceVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	options := client.Options{
		ObserveFrameFrequency: uint32(verb.Observe.Frames),
		ObserveDrawFrequency:  uint32(verb.Observe.Draws),
		StartFrame:            uint32(verb.Start.At.Frame),
		FramesToCapture:       uint32(verb.Capture.Frames),
		APIs:                  uint32(0xFFFFFFFF),
		APK:                   verb.APK,
	}

	if verb.Disable.PCS {
		options.Flags |= client.DisablePrecompiledShaders
	}
	if verb.Record.Errors {
		options.Flags |= client.RecordErrorState
	}
	if verb.Start.Defer {
		options.Flags |= client.DeferStart
	}

	switch verb.API {
	case "vulkan":
		options.APIs = coreAPI | vulkanAPI
	case "gles":
		options.APIs = coreAPI | glesAPI
	case "":
		options.APIs = uint32(0xFFFFFFFF)
	default:
		return fmt.Errorf("Unknown API %s", verb.API)
	}

	if !verb.Local.App.IsEmpty() {
		cleanup, err := verb.startLocalApp(ctx)
		defer func() { cleanup() }()
		if err != nil {
			return err
		}
	}

	if verb.Local.Port != 0 {
		return verb.captureLocal(ctx, flags, verb.Local.Port, options)
	}

	return verb.captureADB(ctx, flags, options)
}

func (verb *traceVerb) startLocalApp(ctx context.Context) (func(), error) {
	// Run the local application with VK_LAYER_PATH, VK_INSTANCE_LAYERS,
	// VK_DEVICE_LAYERS and LD_PRELOAD set to correct values to load the spy
	// layer.
	env := shell.CloneEnv()
	cleanup, err := loader.SetupTrace(ctx, env)
	if err != nil {
		return cleanup, err
	}

	r := regexp.MustCompile("'.+'|\".+\"|\\S+")
	args := r.FindAllString(verb.Local.Args, -1)

	boundPort, err := process.Start(ctx, verb.Local.App.System(), process.StartOptions{
		Env:  env,
		Args: args,
	})

	if err != nil {
		return cleanup, err
	}
	if verb.Local.Port == 0 {
		verb.Local.Port = boundPort
	}
	return cleanup, nil
}

func (verb *traceVerb) captureLocal(ctx context.Context, flags flag.FlagSet, port int, options client.Options) error {
	output := verb.Out
	if output == "" {
		output = "capture.gfxtrace"
	}
	return doCapture(ctx, options, port, output, verb.For)
}

func (verb *traceVerb) captureADB(ctx context.Context, flags flag.FlagSet, options client.Options) error {
	d, err := getADBDevice(ctx, verb.Gapii.Device)
	if err != nil {
		return err
	}
	var pkg *android.InstalledPackage
	var a *android.ActivityAction
	switch {
	case !options.APK.IsEmpty():
		// Install APK, trace it, uninstall
		ctx := log.V{"source": options.APK}.Bind(ctx)
		log.I(ctx, "Installing APK")
		data, err := ioutil.ReadFile(options.APK.System())
		if err != nil {
			return log.Err(ctx, err, "Read APK")
		}
		info, err := apk.Analyze(ctx, data)
		if err != nil {
			return log.Err(ctx, err, "Analyse APK")
		}
		if err := d.InstallAPK(ctx, options.APK.System(), true, true); err != nil {
			return log.Err(ctx, err, "Install APK")
		}
		pkg = &android.InstalledPackage{
			Name:       info.Name,
			Device:     d,
			ABI:        d.Instance().GetConfiguration().PreferredABI(info.ABI),
			Debuggable: info.Debuggable,
		}
		defer func() {
			log.I(ctx, "Uninstall APK")
			pkg.Uninstall(ctx)
		}()
		a = &android.ActivityAction{
			Name:     info.Action,
			Package:  pkg,
			Activity: info.Activity,
		}

	case verb.Android.Attach:
		if verb.TraceFlags.Android.Package == "" {
			return fmt.Errorf("Package name needs to be specified")
		}
		packages, err := d.InstalledPackages(ctx)
		if err != nil {
			return err
		}
		pkg = packages.FindByName(verb.TraceFlags.Android.Package)
		if pkg == nil {
			return fmt.Errorf("Package '%v' not found", verb.TraceFlags.Android.Package)
		}

	case verb.TraceFlags.Android.Package != "":
		packages, err := d.InstalledPackages(ctx)
		if err != nil {
			return err
		}
		pkg = packages.FindByName(verb.TraceFlags.Android.Package)
		if pkg == nil {
			return fmt.Errorf("Package '%v' not found", verb.TraceFlags.Android.Package)
		}
		a = pkg.ActivityActions.FindByName(verb.TraceFlags.Android.Action, verb.TraceFlags.Android.Activity)
		if a == nil {
			lines := make([]string, len(pkg.ActivityActions))
			for i, a := range pkg.ActivityActions {
				lines[i] = a.String()
			}
			return fmt.Errorf("Action '%v:%v' not found. All package actions:\n  %v",
				verb.TraceFlags.Android.Action, verb.TraceFlags.Android.Activity,
				strings.Join(lines, "\n  "))
		}

	default:
		if flags.NArg() != 1 {
			app.Usage(ctx, "Invalid number of arguments. Expected 1, got %d", flags.NArg())
			return nil
		}
		activity := flags.Arg(0)
		a, err = getAction(ctx, d, activity)
		if err != nil {
			return err
		}
		pkg = a.Package
	}
	if pkg.Debuggable {
		log.I(ctx, "Package is debuggable")
	} else {
		err = d.Root(ctx)
		switch err {
		case nil:
		case adb.ErrDeviceNotRooted:
			return err
		default:
			return fmt.Errorf("Failed to restart ADB as root: %v", err)
		}
		log.I(ctx, "Device is rooted")
	}

	// Filenames - if no name specified, use package name.
	output := verb.Out
	if output == "" {
		output = pkg.Name + ".gfxtrace"
	}
	inputFile := verb.Input.File
	if inputFile == "" {
		inputFile = pkg.Name + ".inputs"
	}

	if verb.Clear.Cache {
		log.I(ctx, "Clearing package cache")
		if err := pkg.ClearCache(ctx); err != nil {
			return err
		}
	}

	if wasScreenOn, _ := d.IsScreenOn(ctx); !wasScreenOn {
		defer d.TurnScreenOff(ctx) // Think green!
	}

	port, cleanup, err := client.StartOrAttach(ctx, pkg, a)
	if err != nil {
		return err
	}
	defer cleanup(ctx)

	ctx, stop := task.WithCancel(ctx)
	if verb.Record.Inputs {
		log.I(ctx, "Starting input recorder")
		cleanup, err := startRecordingInputs(ctx, d, inputFile)
		if err != nil {
			return err
		}
		defer cleanup()
	} else if verb.Replay.Inputs {
		log.I(ctx, "Starting input replayer")
		if err := startReplayingInputs(ctx, d, inputFile, stop); err != nil {
			return err
		}
	}

	return doCapture(ctx, options, int(port), output, verb.For)
}

func doCapture(ctx context.Context, options client.Options, port int, out string, duration time.Duration) error {
	log.I(ctx, "Creating file '%v'", out)
	os.MkdirAll(filepath.Dir(out), 0755)
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()

	signal, fireSignal := task.NewSignal()
	if duration == 0 {
		var cancel task.CancelFunc
		ctx, cancel = task.WithCancel(ctx)
		go func() {
			reader := bufio.NewReader(os.Stdin)
			if (options.Flags & client.DeferStart) != 0 {
				println("Press enter to start capturing...")
				_, _ = reader.ReadString('\n')
				fireSignal(ctx)
			}
			println("Press enter to stop capturing...")
			_, _ = reader.ReadString('\n')
			cancel()
		}()
	} else {
		ctx, _ = task.WithTimeout(ctx, duration)
	}
	_, err = client.Capture(ctx, port, signal, file, options)
	if err != nil {
		return err
	}
	return nil
}

func getAction(ctx context.Context, d adb.Device, pattern string) (*android.ActivityAction, error) {
	re := regexp.MustCompile("(?i)" + pattern)
	packages, err := d.InstalledPackages(ctx)
	if err != nil {
		return nil, err
	}
	if len(packages) == 0 {
		return nil, fmt.Errorf("No packages found")
	}
	matchingActions := []*android.ActivityAction{}
	for _, p := range packages {
		for _, action := range p.ActivityActions {
			if re.MatchString(action.String()) {
				matchingActions = append(matchingActions, action)
			}
		}
	}
	if len(matchingActions) == 0 {
		return nil, fmt.Errorf("No actions matching %s found", pattern)
	} else if len(matchingActions) > 1 {
		fmt.Println("Matching actions:")
		for _, test := range matchingActions {
			fmt.Print("    ")
			fmt.Println(test)
		}
		return nil, fmt.Errorf("Multiple actions matching %q found", pattern)
	}
	log.I(ctx, "action: %v", matchingActions[0])
	return matchingActions[0], nil
}
