// Copyright (C) 2018 Google Inc.
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

package android

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	perfetto_pb "perfetto/config"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/android/apk"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapidapk"
	"github.com/google/gapid/gapidapk/pkginfo"
	gapii "github.com/google/gapid/gapii/client"
	"github.com/google/gapid/gapis/perfetto"
	perfetto_android "github.com/google/gapid/gapis/perfetto/android"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/trace/android/adreno"
	"github.com/google/gapid/gapis/trace/android/validate"
	"github.com/google/gapid/gapis/trace/tracer"
)

const (
	activityName                            = "com.samples.cube.VKCubeActivity"
	intentAction                            = "android.intent.action.MAIN"
	validationPackageName                   = "com.samples.cube"
	bufferSizeKb                            = uint32(131072)
	counterPeriodNs                         = uint64(50000000)
	durationMs                              = 7000
	gpuCountersDataSourceDescriptorName     = "gpu.counters"
	gpuRenderStagesDataSourceDescriptorName = "gpu.renderstages"
)

// Only update the package list every 30 seconds at most
var packageUpdateTime = 30.0

type androidTracer struct {
	b                    adb.Device
	packages             *pkginfo.PackageList
	lastIconDensityScale float32
	lastPackageUpdate    time.Time
	v                    validate.Validator
}

func newValidator(dev bind.Device) validate.Validator {
	gpu := dev.Instance().GetConfiguration().GetHardware().GetGPU()
	if strings.Contains(gpu.GetName(), "Adreno") {
		return &adreno.AdrenoValidator{}
	}
	return nil
}

func traceOptions(ctx context.Context, v validate.Validator) (*service.TraceOptions, error) {
	counters := v.GetCounters()
	ids := make([]uint32, len(counters))
	for i, counter := range counters {
		ids[i] = counter.Id
	}
	return &service.TraceOptions{
		DeferStart: true,
		PerfettoConfig: &perfetto_pb.TraceConfig{
			Buffers: []*perfetto_pb.TraceConfig_BufferConfig{
				{SizeKb: proto.Uint32(bufferSizeKb)},
			},
			DurationMs: proto.Uint32(durationMs),
			DataSources: []*perfetto_pb.TraceConfig_DataSource{
				{
					Config: &perfetto_pb.DataSourceConfig{
						Name: proto.String(gpuRenderStagesDataSourceDescriptorName),
					},
				},
				{
					Config: &perfetto_pb.DataSourceConfig{
						Name: proto.String(gpuCountersDataSourceDescriptorName),
						GpuCounterConfig: &perfetto_pb.GpuCounterConfig{
							CounterPeriodNs: proto.Uint64(counterPeriodNs),
							CounterIds:      ids,
						},
					},
				},
			},
		},
	}, nil
}

func (t *androidTracer) GetDevice() bind.Device {
	return t.b
}

func (t *androidTracer) Validate(ctx context.Context) error {
	ctx = status.Start(ctx, "Android Device Validation")
	defer status.Finish(ctx)
	if t.v == nil {
		return log.Errf(ctx, nil, "No validator found for device %d", t.b.Instance().ID.ID())
	}
	d := t.b.(adb.Device)
	osConfiguration := d.Instance().GetConfiguration()
	if osConfiguration.GetPerfettoCapability() == nil {
		return log.Errf(ctx, nil, "No Perfetto Capability found on device %d", d.Instance().ID.ID())
	}
	if gpuProfiling := osConfiguration.GetPerfettoCapability().GetGpuProfiling(); gpuProfiling == nil || gpuProfiling.GetGpuCounterDescriptor() == nil {
		return log.Errf(ctx, nil, "No GPU profiling capabilities found on device %d", d.Instance().ID.ID())
	}

	// Get ActivityAction
	packages, _ := d.InstalledPackages(ctx)
	pkg := packages.FindByName(validationPackageName)
	if pkg == nil {
		return log.Errf(ctx, nil, "Package %v not found", validationPackageName)
	}
	activityAction := pkg.ActivityActions.FindByName(intentAction, activityName)
	if activityAction == nil {
		return log.Errf(ctx, nil, "Activity %v not found in %v", activityName, validationPackageName)
	}

	// Construct trace config
	traceOpts, err := traceOptions(ctx, t.v)
	if err != nil {
		return log.Err(ctx, err, "Could not get the trace configuration")
	}
	process, cleanup, err := perfetto_android.Start(ctx, d, activityAction, traceOpts, nil, []string{})
	if err != nil {
		cleanup.Invoke(ctx)
		return log.Err(ctx, err, "Error when start Perfetto tracing.")
	}
	defer cleanup.Invoke(ctx)

	// Force to stop the application, ignore any error that happens as it
	// doesn't affect validation.
	defer d.ForceStop(ctx, validationPackageName)

	var buf bytes.Buffer
	var written int64
	// Start to capture.
	status.Do(ctx, "Tracing", func(ctx context.Context) {
		startSignal, startFunc := task.NewSignal()
		readyFunc := task.Noop()
		stopSignal, _ := task.NewSignal()
		doneSignal, doneFunc := task.NewSignal()

		crash.Go(func() {
			_, err = process.Capture(ctx, startSignal, stopSignal, readyFunc, &buf, &written)
			doneFunc(ctx)
		})

		// TODO(lpy): If we start tracing too early, the data has a lot of noise
		// at the beginning, need to figure out a sweet spot.
		startFunc(ctx)
		if !doneSignal.Wait(ctx) {
			err = log.Err(ctx, err, "Fail to wait for done signal from Perfetto.")
		}
		log.I(ctx, "Perfetto trace size %v bytes", written)
	})
	if err != nil {
		return err
	}

	var processor *perfetto.Processor
	status.Do(ctx, "Trace loading", func(ctx context.Context) {
		// Load Perfetto trace and create trace processor.
		rawData := make([]byte, written)
		_, err = buf.Read(rawData)
		processor, err = perfetto.NewProcessor(ctx, rawData)
	})
	if err != nil {
		return err
	}

	ctx = status.Start(ctx, "Validation")
	defer status.Finish(ctx)
	return t.v.Validate(ctx, processor)
}

func (t *androidTracer) GetPackages(ctx context.Context, isRoot bool, iconDensityScale float32) (*pkginfo.PackageList, error) {
	refresh := time.Since(t.lastPackageUpdate).Seconds() > packageUpdateTime ||
		t.lastIconDensityScale != iconDensityScale

	if t.packages != nil && !isRoot {
		refresh = false
	}
	if refresh {
		packages, err := gapidapk.PackageList(ctx, t.b, true, iconDensityScale)
		if err != nil {
			return nil, err
		}
		pkgList := &pkginfo.PackageList{
			Packages:       []*pkginfo.Package{},
			Icons:          packages.Icons,
			OnlyDebuggable: packages.OnlyDebuggable,
		}

		for _, p := range packages.Packages {
			for _, activity := range p.Activities {
				if len(activity.Actions) > 0 {
					pkgList.Packages = append(pkgList.Packages, p)
					break
				}
			}
		}

		t.packages = pkgList
		t.lastPackageUpdate = time.Now()
		t.lastIconDensityScale = iconDensityScale
	}
	return t.packages, nil
}

// NewTracer returns a new Tracer for Android.
func NewTracer(dev bind.Device) tracer.Tracer {
	return &androidTracer{dev.(adb.Device), nil, 1.0, time.Time{}, newValidator(dev)}
}

// TraceConfiguration returns the device's supported trace configuration.
func (t *androidTracer) TraceConfiguration(ctx context.Context) (*service.DeviceTraceConfiguration, error) {
	apis := make([]*service.TraceTypeCapabilities, 0, 3)
	if t.b.Instance().GetConfiguration().GetDrivers().GetOpengl().GetVersion() != "" {
		apis = append(apis, tracer.GLESTraceOptions())
	}
	if len(t.b.Instance().GetConfiguration().GetDrivers().GetVulkan().GetPhysicalDevices()) > 0 {
		apis = append(apis, tracer.VulkanTraceOptions())
	}
	if t.b.SupportsPerfetto(ctx) {
		apis = append(apis, tracer.PerfettoTraceOptions())
	}

	return &service.DeviceTraceConfiguration{
		Apis:                 apis,
		ServerLocalPath:      false,
		CanSpecifyCwd:        false,
		CanUploadApplication: true,
		CanSpecifyEnv:        false,
		PreferredRootUri:     "",
		HasCache:             true,
	}, nil
}

func (t *androidTracer) GetTraceTargetNode(ctx context.Context, uri string, iconDensity float32) (*tracer.TraceTargetTreeNode, error) {
	packages, err := t.GetPackages(ctx, uri == "", iconDensity)

	if err != nil {
		return nil, err
	}
	if uri == "" {
		r := &tracer.TraceTargetTreeNode{}
		for _, x := range packages.Packages {
			r.Children = append(r.Children, x.Name)
		}
		sort.Strings(r.Children)
		return r, nil
	}

	pkgName := uri
	actionName := ""
	if strings.Contains(pkgName, ":") {
		actionName = pkgName[0:strings.Index(pkgName, ":")]
		pkgName = pkgName[strings.Index(pkgName, ":")+1:]
	}

	activityName := ""
	if strings.Contains(pkgName, "/") {
		ap := strings.SplitN(pkgName, "/", 2)
		pkgName = ap[0]
		activityName = ap[1]
	}

	pkg := packages.FindByName(pkgName)
	if pkg == nil {
		return nil, log.Errf(ctx, nil, "Could not find package %s", pkgName)
	}

	if activityName != "" {
		var activity *pkginfo.Activity
		for _, a := range pkg.Activities {
			if a.Name == activityName {
				activity = a
				break
			}
		}
		if activity == nil {
			return nil, log.Errf(ctx, nil, "Could not find activity %s, in package %s", activityName, pkgName)
		}

		if actionName != "" {
			for _, a := range activity.Actions {
				if a.Name == actionName {
					return &tracer.TraceTargetTreeNode{
						Name:            actionName,
						Icon:            packages.GetIcon(pkg),
						URI:             fmt.Sprintf("%s:%s/%s", actionName, pkgName, activityName),
						TraceURI:        fmt.Sprintf("%s:%s/%s", actionName, pkgName, activityName),
						Parent:          fmt.Sprintf("%s/%s", pkgName, activityName),
						ApplicationName: pkgName,
					}, nil
				}
			}
			return nil, log.Errf(ctx, nil, "Could not find Intent %s, in package %s/%s", actionName, pkgName, activityName)
		}

		r := &tracer.TraceTargetTreeNode{
			Name:            activityName,
			Icon:            packages.GetIcon(pkg),
			URI:             fmt.Sprintf("%s/%s", pkgName, activityName),
			Parent:          pkgName,
			ApplicationName: pkgName,
		}
		sort.Slice(activity.Actions, func(i, j int) bool { return activity.Actions[i].Name < activity.Actions[j].Name })
		for _, a := range activity.Actions {
			r.Children = append(r.Children, fmt.Sprintf("%s:%s/%s", a.Name, pkgName, activityName))
		}
		if a := findBestAction(activity.Actions); a != nil {
			r.TraceURI = fmt.Sprintf("%s:%s/%s", a.Name, pkgName, activityName)
		}
		return r, nil
	}

	r := &tracer.TraceTargetTreeNode{
		Name:            pkgName,
		Icon:            packages.GetIcon(pkg),
		URI:             pkgName,
		ApplicationName: pkgName,
	}

	var firstActivity *pkginfo.Activity
	var defaultAction string
	sort.Slice(pkg.Activities, func(i, j int) bool { return pkg.Activities[i].Name < pkg.Activities[j].Name })
	for _, activity := range pkg.Activities {
		if len(activity.Actions) > 0 {
			r.Children = append(r.Children, fmt.Sprintf("%s/%s", pkgName, activity.Name))
			if action := findBestAction(activity.Actions); action != nil && action.IsLaunch {
				defaultAction = fmt.Sprintf("%s:%s/%s", action.Name, pkgName, activity.Name)
			}
			if firstActivity == nil {
				firstActivity = activity
			}
		}
	}

	if defaultAction == "" && len(r.Children) == 1 {
		if action := findBestAction(firstActivity.Actions); action != nil {
			defaultAction = fmt.Sprintf("%s:%s/%s", action.Name, pkgName, firstActivity.Name)
		}
	}
	r.TraceURI = defaultAction
	return r, nil
}

// findBestAction returns the best action candidate for tracing from the given
// list. It is either the launch action, the "main" action if no launch action
// was found, the first-and-only action in the list, or nil.
func findBestAction(l []*pkginfo.Action) *pkginfo.Action {
	if len(l) == 1 {
		return l[0]
	}
	var main *pkginfo.Action
	for _, a := range l {
		if a.IsLaunch {
			return a
		}
		if a.Name == "android.intent.action.MAIN" {
			main = a
		}
	}
	return main
}

// InstallPackage installs the given package onto the android device.
// If it is a zip file that contains an apk and an obb file
// then we install them seperately.
// Returns a function used to clean up the package and obb
func (t *androidTracer) InstallPackage(ctx context.Context, o *service.TraceOptions) (*android.InstalledPackage, app.Cleanup, error) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(tempDir)
	// Call it .apk now, because it may actually be our apk
	zipName := filepath.Join(tempDir + "zip.apk")
	apkName := zipName
	obbName := ""

	if err = ioutil.WriteFile(zipName, o.GetUploadApplication(), os.FileMode(0600)); err != nil {
		return nil, nil, err
	}

	r, err := zip.OpenReader(zipName)
	defer r.Close()
	if err != nil {
		return nil, nil, err
	}
	hasObb := false
	if len(r.File) == 2 {
		if (strings.HasSuffix(r.File[0].Name, ".apk") &&
			strings.HasSuffix(r.File[1].Name, ".obb")) ||
			(strings.HasSuffix(r.File[1].Name, ".apk") &&
				strings.HasSuffix(r.File[0].Name, ".obb")) {
			hasObb = true
		}
	}

	if hasObb {
		// We should extract the .zip file into a .apk and a .obb file
		apkName = filepath.Join(tempDir, "a.apk")
		obbName = filepath.Join(tempDir, "a.obb")

		apkFile, err := os.OpenFile(apkName, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			return nil, nil, err
		}
		obbFile, err := os.OpenFile(obbName, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			apkFile.Close()
			return nil, nil, err
		}

		for i := 0; i < 2; i++ {
			if f, err := r.File[i].Open(); err == nil {
				if strings.HasSuffix(r.File[i].Name, ".apk") {
					io.Copy(apkFile, f)
				} else {
					io.Copy(obbFile, f)
				}
				f.Close()
			}

		}
		apkFile.Close()
		obbFile.Close()
	}

	apkData, err := ioutil.ReadFile(apkName)
	if err != nil {
		return nil, nil, log.Err(ctx, err, "Could not read apk file")
	}
	info, err := apk.Analyze(ctx, apkData)
	if err != nil {
		return nil, nil, log.Err(ctx, err, "Could not analyze apk file, not an APK?")
	}

	if err := t.b.InstallAPK(ctx, apkName, true, true); err != nil {
		return nil, nil, log.Err(ctx, err, "Failed to install APK")
	}

	pkg := &android.InstalledPackage{
		Name:        info.Name,
		Device:      t.b,
		ABI:         t.b.Instance().GetConfiguration().PreferredABI(info.ABI),
		Debuggable:  info.Debuggable,
		VersionCode: int(info.VersionCode),
		VersionName: info.VersionName,
	}
	var cleanup app.Cleanup
	if obbName != "" {
		if err := pkg.PushOBB(ctx, obbName); err != nil {
			pkg.Uninstall(ctx)
			return nil, nil, log.Err(ctx, err, "Pushing OBB failed")
		}
		cleanup = func(ctx context.Context) {
			pkg.Uninstall(ctx)
			pkg.RemoveOBB(ctx)
		}
		if err = pkg.GrantExternalStorageRW(ctx); err != nil {
			log.W(ctx, "Failed to grant OBB read/write permission, (app likely already has them). Ignoring: %s", err)
		}
	} else {
		cleanup = func(ctx context.Context) {
			pkg.Uninstall(ctx)
		}
	}

	o.App = &service.TraceOptions_Uri{
		Uri: info.URI(),
	}
	return pkg, cleanup, nil
}

func (t *androidTracer) getActions(ctx context.Context, pattern string) ([]string, error) {
	re := regexp.MustCompile("(?i)" + pattern)
	packages, err := t.GetPackages(ctx, pattern == "", t.lastIconDensityScale)
	if err != nil {
		return nil, err
	}
	if len(packages.Packages) == 0 {
		return nil, fmt.Errorf("No packages found")
	}
	matchingActions := []string{}
	for _, p := range packages.Packages {
		for _, activity := range p.Activities {
			for _, action := range activity.Actions {
				uri := fmt.Sprintf("%s:%s/%s", action.Name, p.Name, activity.Name)
				if re.MatchString(uri) {
					matchingActions = append(matchingActions, uri)
				}
			}
		}
	}
	if len(matchingActions) == 0 {
		return nil, fmt.Errorf("No actions matching %s found", pattern)
	}
	return matchingActions, nil
}

func (t *androidTracer) FindTraceTargets(ctx context.Context, str string) ([]*tracer.TraceTargetTreeNode, error) {
	uris, err := t.getActions(ctx, str)
	if err != nil {
		return nil, err
	}

	nodes := make([]*tracer.TraceTargetTreeNode, len(uris))
	for i, uri := range uris {
		n, err := t.GetTraceTargetNode(ctx, uri, t.lastIconDensityScale)
		if err != nil {
			return nil, err
		}
		nodes[i] = n
	}
	return nodes, nil
}

func (t *androidTracer) SetupTrace(ctx context.Context, o *service.TraceOptions) (tracer.Process, app.Cleanup, error) {
	var err error
	var cleanup app.Cleanup
	var pkg *android.InstalledPackage
	var a *android.ActivityAction
	ret := &gapii.Process{}
	if len(o.GetUploadApplication()) > 0 {
		pkg, cleanup, err = t.InstallPackage(ctx, o)
		if err != nil {
			return ret, nil, err
		}
	}

	// Handle the special port:<pipe>:<abi> syntax.
	re := regexp.MustCompile("^port:([^:\\s]+):([^:\\s]+)$")
	match := re.FindStringSubmatch(o.GetUri())

	if len(match) == 3 {
		process, err := gapii.Connect(ctx, t.b, device.ABIByName(match[2]), match[1], tracer.GapiiOptions(o))
		if err != nil {
			return ret, cleanup.Invoke(ctx), err
		}
		return process, cleanup, nil
	}

	// Find the package by URI
	re = regexp.MustCompile("([^:]*):([^/]*)/\\.?(.*)")
	match = re.FindStringSubmatch(o.GetUri())

	if len(match) == 4 {
		packages, err := t.b.InstalledPackages(ctx)
		if err != nil {
			return ret, nil, err
		}
		pkg = packages.FindByName(match[2])
		if pkg == nil {
			return ret, cleanup.Invoke(ctx), fmt.Errorf("Package '%v' not found", match[2])
		}
		a = pkg.ActivityActions.FindByName(match[1], match[3])
		if a == nil {
			lines := make([]string, len(pkg.ActivityActions))
			for i, a := range pkg.ActivityActions {
				lines[i] = a.String()
			}
			return ret, cleanup.Invoke(ctx), fmt.Errorf("Action '%v:%v' not found. All package actions:\n  %v",
				match[1], match[3],
				strings.Join(lines, "\n  "))
		}
	} else if o.Type != service.TraceType_Perfetto || len(o.GetUri()) != 0 {
		return ret, nil, fmt.Errorf("Could not find package matching %s", o.GetUri())
	}

	if pkg != nil {
		if !pkg.Debuggable {
			err = t.b.Root(ctx)
			switch err {
			case nil:
			case adb.ErrDeviceNotRooted:
				return ret, cleanup.Invoke(ctx), log.Err(ctx, err, "Cannot trace non-debuggable app")
			default:
				return ret, cleanup.Invoke(ctx), fmt.Errorf("Failed to restart ADB as root: %v", err)
			}
			log.I(ctx, "Device is rooted")
		}

		if o.ClearCache {
			log.I(ctx, "Clearing package cache")
			if err := pkg.ClearCache(ctx); err != nil {
				return ret, nil, err
			}
		}
	}

	var process tracer.Process
	if o.Type == service.TraceType_Perfetto {
		var layers []string
		var packageABI *device.ABI
		if pkg != nil {

			d := pkg.Device.(adb.Device)
			abi := pkg.ABI
			if abi.SameAs(device.UnknownABI) {
				abi = pkg.Device.Instance().GetConfiguration().PreferredABI(nil)
			}
			// For NativeBridge emulated devices opt for the native ABI of the emulator.
			packageABI = d.NativeBridgeABI(ctx, abi)
			layers = tracer.LayersFromOptions(ctx, o)
		}
		var perfettoCleanup app.Cleanup
		log.E(ctx, "Setting up layers %+v: %+v", packageABI, layers)
		process, perfettoCleanup, err = perfetto_android.Start(ctx, t.b, a, o, packageABI, layers)
		cleanup = cleanup.Then(perfettoCleanup)
	} else {
		log.I(ctx, "Starting with options %+v", tracer.GapiiOptions(o))
		var gapiiCleanup app.Cleanup
		process, gapiiCleanup, err = gapii.Start(ctx, pkg, a, tracer.GapiiOptions(o))
		cleanup = cleanup.Then(gapiiCleanup)
	}
	if err != nil {
		return ret, cleanup.Invoke(ctx), err
	}
	return process, cleanup, nil
}
