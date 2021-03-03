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
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/gapis/service"
	gapidPath "github.com/google/gapid/gapis/service/path"
)

// These versions must match the ones in gapidapk/android/apk/AndroidManifest.xml.in
const minSdkVersion = 21
const targetSdkVersion = 26

type exportReplayVerb struct{ ExportReplayFlags }

func init() {
	verb := &exportReplayVerb{
		ExportReplayFlags{Out: "replay_export", Apk: "", SdkPath: "", LoopCount: 1},
	}
	app.AddVerb(&app.Verb{
		Name:      "export_replay",
		ShortHelp: "Export replay vm instruction and assets.",
		Action:    verb,
	})
}

func isPackageNameValid(name string) bool {
	// See https://developer.android.com/studio/build/application-id
	packageNameRE := regexp.MustCompile(`^[[:alpha:]][[:word:]]*(\.[[:alpha:]][[:word:]]*)+$`)
	return packageNameRE.MatchString(name)
}

func (verb *exportReplayVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	// Early argument check
	var replayAPK, replayPackage string
	if verb.Apk != "" {
		if !strings.HasSuffix(verb.Apk, ".apk") {
			app.Usage(ctx, "APK name must be a valid Android package name followed by '.apk', e.g. com.example.myapp.replay.apk")
			return nil
		}
		replayAPK = filepath.Base(verb.Apk)
		replayPackage = strings.TrimSuffix(replayAPK, ".apk")
		if !isPackageNameValid(replayPackage) {
			app.Usage(ctx, "APK package name '%s' is invalid, make sure to use alphanum characters and at least one '.' separator, e.g. com.example.myapp.replay.apk (see https://developer.android.com/studio/build/application-id)", replayPackage)
			return nil
		}
		if _, err := os.Stat(verb.Apk); err == nil {
			app.Usage(ctx, "APK archive file must not exists")
			return nil
		}
	}

	client, capturePath, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	var device *gapidPath.Device
	if !verb.OriginalDevice {
		device, err = getDevice(ctx, client, capturePath, verb.Gapir)
		if err != nil {
			return err
		}
	}

	var fbreqs []*gapidPath.FramebufferAttachment
	var tsreq *service.GetTimestampsRequest
	onscreen := false
	switch verb.Mode {
	case ExportPlain, ExportDiagnostics:
		// It's the default, do nothing.
	case ExportOnScreen:
		onscreen = true
	case ExportFrames:
		filter, err := verb.CommandFilterFlags.commandFilter(ctx, client, capturePath)
		if err != nil {
			return log.Err(ctx, err, "Couldn't get filter")
		}

		requestEvents := gapidPath.Events{
			Capture:     capturePath,
			LastInFrame: true,
			Filter:      filter,
		}

		// Get the end-of-frame events.
		eofEvents, err := getEvents(ctx, client, &requestEvents)
		if err != nil {
			return log.Err(ctx, err, "Couldn't get frame events")
		}

		for _, e := range eofEvents {
			fbreqs = append(fbreqs, &gapidPath.FramebufferAttachment{
				After:          e.Command,
				Index:          0,
				RenderSettings: &gapidPath.RenderSettings{DisableReplayOptimization: true},
				Hints:          nil,
			})
		}
	case ExportTimestamps:
		// There are no useful field in GetTimestampsRequest as of now.
		tsreq = &service.GetTimestampsRequest{}
	}

	opts := &service.ExportReplayOptions{
		FramebufferAttachments: fbreqs,
		GetTimestampsRequest:   tsreq,
		DisplayToSurface:       onscreen,
	}

	if err := client.ExportReplay(ctx, capturePath, device, verb.Out, opts); err != nil {
		return log.Err(ctx, err, "Failed to export replay")
	}

	if verb.Apk != "" {

		// Create stand-alone APK
		log.I(ctx, "Create replay apk: %s with package name %s", replayAPK, replayPackage)

		boxedCapture, err := client.Get(ctx, capturePath.Path(), nil)
		if err != nil {
			return log.Err(ctx, err, "Failed to load the capture")
		}
		capture := boxedCapture.(*service.Capture)

		// Save current directory
		startdir, err := os.Getwd()
		if err != nil {
			return err
		}

		// Operate in a temporary directory
		tmpdir, err := ioutil.TempDir("", "gapid")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpdir) // clean up

		if err := os.Chdir(tmpdir); err != nil {
			return err
		}
		defer os.Chdir(startdir)

		// Retrieve replay files
		if err := os.Mkdir("assets", os.ModePerm); err != nil {
			return err
		}
		assetsPath := path.Join("assets", "replay_export")
		if err := os.Rename(path.Join(startdir, verb.Out), assetsPath); err != nil {
			return err
		}

		// Extract reference APK
		refAPKPath, err := layout.GapidApk(ctx, capture.ABI)
		if err != nil {
			return err
		}
		if err := file.Unzip(ctx, refAPKPath.String(), tmpdir); err != nil {
			return err
		}

		// Create new manifest
		f, err := os.Create("AndroidManifest.xml")
		if err != nil {
			return err
		}
		t, err := template.New("manifest").Parse(manifest())
		if err != nil {
			return err
		}
		type ManifestEntries struct {
			Package          string
			MinSdkVersion    int
			TargetSdkVersion int
		}
		var entries = ManifestEntries{replayPackage, minSdkVersion, targetSdkVersion}
		t.Execute(f, entries)
		f.Close()

		// Android tools

		// find latest build tools
		sdkPath := verb.SdkPath
		if sdkPath == "" {
			sdkPath = os.ExpandEnv("${ANDROID_SDK_HOME}")
		}
		if _, err := os.Stat(sdkPath); err != nil {
			return log.Err(ctx, err, "Cannot find Android SDK. Please set ANDROID_SDK_HOME, or use the -sdkpath flag")
		}
		toolsPathParent := path.Join(sdkPath, "build-tools")
		matches, err := filepath.Glob(path.Join(toolsPathParent, "*"))
		if err != nil {
			return err
		}
		if len(matches) <= 0 {
			return fmt.Errorf("Cannot find any directory under " + toolsPathParent)
		}
		sort.Strings(matches)
		toolsPath := matches[len(matches)-1]

		aapt := path.Join(toolsPath, "aapt")
		zipalign := path.Join(toolsPath, "zipalign")
		apksigner := path.Join(toolsPath, "apksigner")

		tmpAPK := "tmp.apk"

		// Re-assemble new APK with the new manifest
		baseJar := path.Join(sdkPath, "platforms", "android-"+strconv.Itoa(targetSdkVersion), "android.jar")
		if _, err := os.Stat(verb.Apk); err == nil {
			return fmt.Errorf("Cannot find android platform %d, please install it.", targetSdkVersion)
		}

		if err := shell.Command(aapt, "package", "-f", "-M", "AndroidManifest.xml", "-I", baseJar, "-F", tmpAPK).Run(ctx); err != nil {
			return err
		}

		// Add replay assets, uncompressed
		assets, err := ioutil.ReadDir(assetsPath)
		if err != nil {
			return err
		}
		for _, a := range assets {
			// Arguments ("-0", "") ensure the asset will not be compressed
			if err := shell.Command(aapt, "add", "-0", "", tmpAPK, path.Join(assetsPath, a.Name())).Run(ctx); err != nil {
				return err
			}
		}

		// Add the replay libraries
		abi := capture.ABI.Name

		files := []string{
			"classes.dex",
			path.Join("lib", abi, "libgapir.so"),
			path.Join("lib", abi, "libVkLayer_VirtualSwapchain.so"),
			path.Join("lib", abi, "libVkLayer_CPUTiming.so"),
			path.Join("lib", abi, "libVkLayer_MemoryTracker.so"),
		}

		for _, f := range files {
			if err := shell.Command(aapt, "add", "-0", "", tmpAPK, f).Run(ctx); err != nil {
				return err
			}
		}

		// Zip-align, output in final APK file
		replayAPKPath := path.Join(startdir, replayAPK)
		if err := shell.Command(zipalign, "4", tmpAPK, replayAPKPath).Run(ctx); err != nil {
			return err
		}

		// Sign the new APK
		keystorePath := path.Join(os.ExpandEnv("${HOME}"), ".android", "debug.keystore")
		if _, err := os.Stat(keystorePath); err != nil {
			// No keystore found, create one
			keystorePath = "debug.keystore"
			// https://developer.android.com/studio/publish/app-signing#debug-mode
			keytool := "keytool"
			if _, err := exec.LookPath("keytool"); err != nil {
				// keytool is not found in PATH, look in JAVA_HOME/bin
				keytool = path.Join(os.ExpandEnv("JAVA_HOME"), "bin")
				if _, err := os.Stat(keytool); err != nil {
					return fmt.Errorf("Cannot find the 'keytool' command")
				}
			}
			if err := shell.Command(keytool, "-genkey", "-dname", "CN=Android Debug,O=Android,C=US", "-v", "-keystore", keystorePath, "-storepass", "android", "-alias", "androiddebugkey", "-keypass", "android", "-keyalg", "RSA", "-keysize", "2048", "-validity", "10000").Run(ctx); err != nil {
				return err
			}
		}
		if err := shell.Command(apksigner, "sign", "--ks", keystorePath, "--ks-pass", "pass:android", replayAPKPath).Run(ctx); err != nil {
			return err
		}

	}

	return nil
}

func manifest() string {
	return `<?xml version="1.0" encoding="utf-8"?>
<!--
  Copyright (C) 2018 Google Inc.

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
-->
<manifest xmlns:android="http://schemas.android.com/apk/res/android"
    package="{{.Package}}"
    android:versionCode="1"
    android:versionName="0.1"
    >

    <uses-sdk
        android:minSdkVersion="{{.MinSdkVersion}}"
        android:targetSdkVersion="{{.TargetSdkVersion}}" />

    <application
        android:allowBackup="true"
        android:label="Replay-{{.Package}}"
        android:supportsRtl="true"
        android:debuggable="true"
        >
        <activity android:name="android.app.NativeActivity"
                  android:label="Replay-{{.Package}}">
            <meta-data android:name="android.app.lib_name"
                       android:value="gapir"/>
            <intent-filter>
                <action android:name=".gapir"/>
            </intent-filter>
        </activity>
        <service
            android:name="com.google.android.gapid.DeviceInfoService"
            android:exported="true">
            <intent-filter>
                <action android:name="com.google.android.gapid.action.SEND_DEV_INFO"/>
            </intent-filter>
        </service>
        <service
            android:name="com.google.android.gapid.PackageInfoService"
            android:exported="true">
            <intent-filter>
                <action android:name="com.google.android.gapid.action.SEND_PKG_INFO"/>
            </intent-filter>
        </service>
    </application>
</manifest>
`
}
