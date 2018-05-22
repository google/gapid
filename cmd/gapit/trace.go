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
	"github.com/google/gapid/core/app/analytics"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/android/apk"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/process"
	"github.com/google/gapid/core/vulkan/loader"
	"github.com/google/gapid/gapii/client"
)

type traceVerb struct{ TraceFlags }

func init() {
	verb := &traceVerb{}
	verb.TraceFlags.Disable.PCS = true

	app.AddVerb(&app.Verb{
		Name:      "trace",
		ShortHelp: "Captures a gfx trace from an application",
		Action:    verb,
	})
}

func (verb *traceVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	ctx = bind.PutRegistry(ctx, bind.NewRegistry())

	options := traceOptions{
		Options: client.Options{
			ObserveFrameFrequency: uint32(verb.Observe.Frames),
			ObserveDrawFrequency:  uint32(verb.Observe.Draws),
			StartFrame:            uint32(verb.Start.At.Frame),
			FramesToCapture:       uint32(verb.Capture.Frames),
			APIs:                  uint32(0xFFFFFFFF),
			APK:                   verb.APK,
			AdditionalFlags:       verb.Android.AdditionalArgs,
		},
		monitorLogcat: verb.TraceFlags.Android.Logcat,
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
	if verb.No.Buffer {
		options.Flags |= client.NoBuffer
	}

	if verb.ADB != "" {
		adb.ADB = file.Abs(verb.ADB)
	}

	switch verb.API {
	case "vulkan":
		options.APIs = client.VulkanAPI
	case "gles":
		// TODO: Separate these two out once we can trace Vulkan with OpenGL ES.
		options.APIs = client.GlesAPI | client.GvrAPI
	case "":
		options.APIs = uint32(0xFFFFFFFF)
	default:
		return fmt.Errorf("Unknown API %s", verb.API)
	}


	ctx, start := verb.inputHandler(ctx, options.Flags&client.DeferStart != 0)

	if verb.Local.App != "" || verb.Local.Port != 0 {
		device, err := getDesktopTraceDevice(ctx, verb.Gapii); 
		if err != nil {
			return err
		}
		if verb.Local.App != "" {
			cleanup, err := verb.startLocalApp(ctx, device)
			defer func() { cleanup(ctx) }()
			if err != nil {
				return err
			}
		}
		return verb.captureLocal(ctx, device, verb.Local.Port, start, options)
	}

	return verb.captureADB(ctx, flags, start, options)
}

func (verb *traceVerb) inputHandler(ctx context.Context, deferStart bool) (context.Context, task.Signal) {
	if verb.For > 0 {
		return ctx, task.FiredSignal
	}
	startSignal, start := task.NewSignal()
	var cancel task.CancelFunc
	ctx, cancel = task.WithCancel(ctx)
	crash.Go(func() {
		reader := bufio.NewReader(os.Stdin)
		if deferStart {
			println("Press enter to start capturing...")
			_, _ = reader.ReadString('\n')
			start(ctx)
		}
		println("Press enter to stop capturing...")
		_, _ = reader.ReadString('\n')
		cancel()
	})
	return ctx, startSignal
}

func (verb *traceVerb) startLocalApp(ctx context.Context, d bind.Device) (func(ctx context.Context), error) {
	// Run the local application with VK_LAYER_PATH, VK_INSTANCE_LAYERS,
	// VK_DEVICE_LAYERS and LD_PRELOAD set to correct values to load the spy
	// layer.
	env, err := d.GetEnv(ctx)
	if err != nil {
		return nil, err
	}
	cleanup, portFile, err := loader.SetupTrace(ctx, d, d.Instance().Configuration.ABIs[0], env)
	if err != nil {
		return cleanup, err
	}

	r := regexp.MustCompile("'.+'|\".+\"|\\S+")
	args := r.FindAllString(verb.Local.Args, -1)
	ctx, cancel := context.WithCancel(ctx)
	for _, x := range verb.Gapii.Env {
		env.Add(x)
	}

	boundPort, err := process.StartOnDevice(ctx, verb.Local.App, process.StartOptions{
		Env:        env,
		Args:       args,
		PortFile:   portFile,
		WorkingDir: verb.Local.WorkingDir,
		Device: 	d,
	})

	if err != nil {
		cancel()
		return cleanup, err
	}
	if verb.Local.Port == 0 {
		verb.Local.Port = boundPort
	}
	return func(ctx context.Context) { cancel(); cleanup(ctx) }, nil
}

type traceOptions struct {
	client.Options
	monitorLogcat bool
}

func (verb *traceVerb) captureLocal(ctx context.Context, device bind.Device, port int, start task.Signal, options traceOptions) error {
	defer analytics.SendTiming("trace", "local")(
		analytics.TargetDevice(device.Instance().GetConfiguration()),
	)

	output := verb.Out
	if output == "" {
		output = "capture.gfxtrace"
	}
	process := &client.Process{Port: port, Device: device, Options: options.Options}
	return doCapture(ctx, process, output, start, verb.For)
}

func (verb *traceVerb) captureADB(ctx context.Context, flags flag.FlagSet, start task.Signal, options traceOptions) error {
	d, err := getADBDevice(ctx, verb.Gapii.Device)
	if err != nil {
		return err
	}

	defer analytics.SendTiming("trace", "adb")(
		analytics.TargetDevice(d.Instance().GetConfiguration()),
	)

	if options.monitorLogcat {
		c := make(chan android.LogcatMessage, 32)
		// this is to prevent logcat messages from triggering failures in robot
		ctx := log.PutHandler(ctx, log.Channel(app.Flags.Log.Style.Handler(log.Stdout()), 32))
		go func() {
			for m := range c {
				m.Log(ctx)
			}
		}()
		go d.Logcat(ctx, c)
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
			Name:        info.Name,
			Device:      d,
			ABI:         d.Instance().GetConfiguration().PreferredABI(info.ABI),
			Debuggable:  info.Debuggable,
			VersionCode: int(info.VersionCode),
			VersionName: info.VersionName,
		}
		if !verb.OBB.IsEmpty() {
			if err := pkg.PushOBB(ctx, verb.OBB.System()); err != nil {
				return log.Err(ctx, err, "Push OBB")
			}
			defer func() {
				log.I(ctx, "Remove OBB")
				pkg.RemoveOBB(ctx)
			}()
			if err := pkg.GrantExternalStorageRW(ctx); err != nil {
				log.W(ctx, "Failed to grant OBB read/write permission, (app likely already has them). Ignoring: %s", err)
			}
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

	process, err := client.StartOrAttach(ctx, pkg, a, options.Options)
	if err != nil {
		return err
	}

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

	return doCapture(ctx, process, output, start, verb.For)
}

func doCapture(ctx context.Context, process *client.Process, out string, start task.Signal, duration time.Duration) error {
	log.I(ctx, "Creating file '%v'", out)
	os.MkdirAll(filepath.Dir(out), 0755)
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()

	if duration > 0 {
		ctx, _ = task.WithTimeout(ctx, duration)
	}

	_, err = process.Capture(ctx, start, file)
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
