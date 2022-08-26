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

// Regress is a tool to display build and runtime statistics over a range of
// changelists.
package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/git"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/shell"
)

var (
	root      = flag.String("root", "", "Path to the root GAPID source directory")
	verbose   = flag.Bool("verbose", false, "Verbose logging")
	incBuild  = flag.Bool("inc", true, "Time incremental builds")
	optimize  = flag.Bool("optimize", false, "Build using '-c opt'")
	pkg       = flag.String("pkg", "", "Partial name of a package name to capture")
	atSHA     = flag.String("at", "", "The SHA or branch of the first changelist to profile")
	count     = flag.Int("count", 2, "The number of changelists to profile since HEAD")
	tracePath = flag.String("trace", "", "Path to a .gfxtrace used for report timing")
)

func main() {
	app.ShortHelp = "Regress is a tool to perform performance measurments over a range of CLs."
	app.Run(run)
}

type stats struct {
	SHA                  string   `name:"sha"`
	IncrementalBuildTime float64  `name:"incremental-build"` // in seconds
	FileSizes            struct { // in bytes
		LibGAPII                   int `name:"libgapii"`
		LibVkLayerVirtualSwapchain int `name:"libVkLayer_VirtualSwapchain"`
		LibVkLayerCPUTiming        int `name:"libVkLayer_CPUTiming"`
		LibVkLayerMemoryTracker    int `name:"libVkLayer_MemoryTracker"`
		GAPIDARMv8aAPK             int `name:"gapid-arm64-v8a"`
		GAPIDARMv7aAPK             int `name:"gapid-armeabi-v7a"`
		GAPIDX86APK                int `name:"gapid-x86"`
		GAPID                      int `name:"gapid"`
		GAPIR                      int `name:"gapir"`
		GAPIS                      int `name:"gapis"`
		GAPIT                      int `name:"gapit"`
	}
	CaptureStats struct {
		Frames   int `name:"frames"`
		Draws    int `name:"draws"`
		Commands int `name:"commands"`
	}
	ReplayStats struct {
		ReportTime    float64 `name:"report-time"`    // in seconds
		LinearizeTime float64 `name:"linearize-time"` // in seconds
	}
}

func run(ctx context.Context) error {
	if *root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		*root = wd
	}

	g, err := git.New(*root)
	if err != nil {
		return err
	}
	s, err := g.Status(ctx)
	if err != nil {
		return err
	}
	if !s.Clean() {
		return fmt.Errorf("Local changes found. Please submit any changes and run again")
	}

	branch, err := g.CurrentBranch(ctx)
	if err != nil {
		return err
	}

	defer g.CheckoutBranch(ctx, branch)

	cls, err := g.LogFrom(ctx, *atSHA, *count)
	if err != nil {
		return err
	}

	rnd := rand.New(rand.NewSource(time.Now().Unix()))

	res := []stats{}
	for i := range cls {
		i := len(cls) - 1 - i
		cl := cls[i]
		sha := cl.SHA.String()[:6]

		r := stats{SHA: sha}

		log.I(ctx, "HEAD~%.2d: Building at %v: %v", i, sha, cl.Subject)
		if err := g.Checkout(ctx, cl.SHA); err != nil {
			return err
		}

		_, err := build(ctx)
		if err != nil {
			continue
		}

		// Gather file size build stats
		pkgDir := filepath.Join(*root, "bazel-bin", "pkg")
		for _, f := range []struct {
			path string
			size *int
		}{
			{filepath.Join(pkgDir, "lib", dllExt("libgapii")), &r.FileSizes.LibGAPII},
			{filepath.Join(pkgDir, "lib", dllExt("libVkLayer_VirtualSwapchain")), &r.FileSizes.LibVkLayerVirtualSwapchain},
			{filepath.Join(pkgDir, "lib", dllExt("libVkLayer_CPUTiming")), &r.FileSizes.LibVkLayerCPUTiming},
			{filepath.Join(pkgDir, "lib", dllExt("libVkLayer_MemoryTracker")), &r.FileSizes.LibVkLayerMemoryTracker},
			{filepath.Join(pkgDir, "gapid-armeabi-v7a.apk"), &r.FileSizes.GAPIDARMv7aAPK},
			{filepath.Join(pkgDir, "gapid-arm64-v8a.apk"), &r.FileSizes.GAPIDARMv8aAPK},
			{filepath.Join(pkgDir, "gapid-x86.apk"), &r.FileSizes.GAPIDX86APK},
			{filepath.Join(pkgDir, exeExt("gapid")), &r.FileSizes.GAPID},
			{filepath.Join(pkgDir, exeExt("gapir")), &r.FileSizes.GAPIR},
			{filepath.Join(pkgDir, exeExt("gapis")), &r.FileSizes.GAPIS},
			{filepath.Join(pkgDir, exeExt("gapit")), &r.FileSizes.GAPIT},
		} {
			fi, err := os.Stat(f.path)
			if err != nil {
				log.W(ctx, "Couldn't stat file '%v': %v", f.path, err)
				continue
			}
			*f.size = int(fi.Size())
		}

		// Gather capture stats
		if *pkg != "" {
			file, err := trace(ctx)
			if err != nil {
				log.W(ctx, "Couldn't capture trace: %v", err)
				continue
			}
			defer os.Remove(file)
			frames, draws, cmds, err := captureStats(ctx, file)
			if err != nil {
				continue
			}
			r.CaptureStats.Frames = frames
			r.CaptureStats.Draws = draws
			r.CaptureStats.Commands = cmds
			*tracePath = file
		}

		if *tracePath != "" {
			start := time.Now()
			if err := report(ctx, *tracePath); err != nil {
				return err
			}
			r.ReplayStats.ReportTime = time.Since(start).Seconds()

			start = time.Now()
			if err := linearize(ctx, *tracePath); err != nil {
				return err
			}
			r.ReplayStats.LinearizeTime = time.Since(start).Seconds()
		}

		// Gather incremental build stats
		if *incBuild {
			if err := withTouchedVulkan(ctx, rnd, func() error {
				log.I(ctx, "HEAD~%.2d: Building incremental change at %v: %v", i, sha, cl.Subject)
				if duration, err := build(ctx); err == nil {
					r.IncrementalBuildTime = duration.Seconds()
				}
				return nil
			}); err != nil {
				continue
			}
		}

		res = append(res, r)
	}

	w := tabwriter.NewWriter(os.Stdout, 1, 4, 0, ' ', 0)
	defer w.Flush()

	var display func(get func(stats) reflect.Value, ty reflect.Type, name string)
	display = func(get func(stats) reflect.Value, ty reflect.Type, name string) {
		switch ty.Kind() {
		case reflect.Struct:
			for i, c := 0, ty.NumField(); i < c; i++ {
				f := ty.Field(i)
				get := func(s stats) reflect.Value { return get(s).Field(i) }
				name := f.Name
				if n := f.Tag.Get("name"); n != "" {
					name = n
				}
				display(get, f.Type, name)
			}
		default:
			fmt.Fprint(w, name)
			var prev reflect.Value
			for i, s := range res {
				v := get(s)
				var old, new float64
				if i > 0 {
					switch v.Kind() {
					case reflect.Int:
						old, new = float64(prev.Int()), float64(v.Int())
					case reflect.Float64:
						old, new = prev.Float(), v.Float()
					}
				}
				if old != new {
					percent := 100 * (new - old) / old
					fmt.Fprintf(w, "\t | %v \t(%+4.1f%%)", v.Interface(), percent)
				} else {
					fmt.Fprintf(w, "\t | %v \t", v.Interface())
				}
				prev = v
			}
			fmt.Fprintln(w)
		}
	}

	display(
		func(s stats) reflect.Value { return reflect.ValueOf(s) },
		reflect.TypeOf(stats{}),
		"")

	return nil
}

func withTouchedVulkan(ctx context.Context, r *rand.Rand, f func() error) error {
	vulkanAPIPath := filepath.Join(*root, "gapis", "api", "vulkan", "vulkan.api")
	fi, err := os.Stat(vulkanAPIPath)
	if err != nil {
		return err
	}
	vulkanAPI, err := ioutil.ReadFile(vulkanAPIPath)
	if err != nil {
		return err
	}
	modVulkanAPI := []byte(fmt.Sprintf("%v\ncmd void fake_cmd_%d() {}\n", string(vulkanAPI), r.Int()))
	ioutil.WriteFile(vulkanAPIPath, modVulkanAPI, fi.Mode().Perm())
	defer ioutil.WriteFile(vulkanAPIPath, vulkanAPI, fi.Mode().Perm())
	return f()
}

func build(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	args := []string{"build"}
	if *optimize {
		args = append(args, "-c", "opt")
	}
	args = append(args, "pkg")
	cmd := shell.Cmd{
		Name:      "bazel",
		Args:      args,
		Verbosity: *verbose,
		Dir:       *root,
	}
	if _, err := cmd.Call(ctx); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

func dllExt(n string) string {
	switch runtime.GOOS {
	case "windows":
		return n + ".dll"
	case "darwin":
		return n + ".dylib"
	case "darwin_arm64":
		return n + ".dylib"
	default:
		return n + ".so"
	}
}

func exeExt(n string) string {
	switch runtime.GOOS {
	case "windows":
		return n + ".exe"
	default:
		return n
	}
}

func gapitPath() string { return filepath.Join(*root, "bazel-bin", "pkg", exeExt("gapit")) }

func trace(ctx context.Context) (string, error) {
	file := filepath.Join(os.TempDir(), "gapid-regres.gfxtrace")
	cmd := shell.Cmd{
		Name:      gapitPath(),
		Args:      []string{"--log-style", "raw", "trace", "--for", "60s", "--out", file, *pkg},
		Verbosity: *verbose,
	}
	_, err := cmd.Call(ctx)
	if err != nil {
		os.Remove(file)
		return "", err
	}
	return file, err
}

func report(ctx context.Context, trace string) error {
	cmd := shell.Cmd{
		Name:      gapitPath(),
		Args:      []string{"--log-style", "raw", "report", trace},
		Verbosity: *verbose,
	}
	_, err := cmd.Call(ctx)
	return err
}

func linearize(ctx context.Context, trace string) error {
	args := []string{
		"run", "cmd/linearize_trace",
		"-c", "opt",
		"--",
		"--file", trace,
	}
	args = append(args, "pkg")
	cmd := shell.Cmd{
		Name:      "bazel",
		Args:      args,
		Verbosity: *verbose,
		Dir:       *root,
	}
	_, err := cmd.Call(ctx)
	return err
}

func captureStats(ctx context.Context, file string) (numFrames, numDraws, numCmds int, err error) {
	cmd := shell.Cmd{
		Name:      gapitPath(),
		Args:      []string{"--log-style", "raw", "--log-level", "error", "stats", file},
		Verbosity: *verbose,
	}
	stdout, err := cmd.Call(ctx)
	if err != nil {
		return 0, 0, 0, nil
	}
	re := regexp.MustCompile(`([a-zA-Z]+):\s+([0-9]+)`)
	for _, matches := range re.FindAllStringSubmatch(stdout, -1) {
		if len(matches) != 3 {
			continue
		}
		n, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}
		switch matches[1] {
		case "Frames":
			numFrames = n
		case "Draws":
			numDraws = n
		case "Commands":
			numCmds = n
		}
	}
	return
}
