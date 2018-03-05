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
	"runtime"
	"text/tabwriter"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/git"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/shell"
)

var (
	root     = flag.String("root", "", "Path to the root GAPID source directory")
	verbose  = flag.Bool("verbose", false, "Verbose logging")
	optimize = flag.Bool("optimize", false, "Build using '-c opt'")
	output   = flag.String("out", "", "The results output file. Empty writes to stdout")
	count    = flag.Int("count", 2, "The number of changelists to profile since HEAD")
)

func main() {
	app.ShortHelp = "Regress is a tool to perform performance measurments over a range of CLs."
	app.Run(run)
}

type stats struct {
	SHA                  string
	BuildTime            float64  // in seconds
	IncrementalBuildTime float64  // in seconds
	FileSizes            struct { // in bytes
		LibGAPII                   int64
		LibVkLayerVirtualSwapchain int64
		GAPIDAarch64APK            int64
		GAPIDArmeabi64APK          int64
		GAPIDX86APK                int64
		GAPID                      int64
		GAPIR                      int64
		GAPIS                      int64
		GAPIT                      int64
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

	cls, err := g.Log(ctx, *count)
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

		duration, err := build(ctx)
		if err != nil {
			continue
		}
		r.BuildTime = duration.Seconds()

		if err := withTouchedGLES(ctx, rnd, func() error {
			log.I(ctx, "HEAD~%.2d: Building incremental change at %v: %v", i, sha, cl.Subject)
			if duration, err := build(ctx); err == nil {
				r.IncrementalBuildTime = duration.Seconds()
			}
			return nil
		}); err != nil {
			continue
		}

		pkg := filepath.Join(*root, "bazel-bin", "pkg")
		for _, f := range []struct {
			path string
			size *int64
		}{
			{filepath.Join(pkg, "lib", dllExt("libgapii")), &r.FileSizes.LibGAPII},
			{filepath.Join(pkg, "lib", dllExt("libVkLayer_VirtualSwapchain")), &r.FileSizes.LibVkLayerVirtualSwapchain},
			{filepath.Join(pkg, "gapid-aarch64.apk"), &r.FileSizes.GAPIDAarch64APK},
			{filepath.Join(pkg, "gapid-armeabi.apk"), &r.FileSizes.GAPIDArmeabi64APK},
			{filepath.Join(pkg, "gapid-x86.apk"), &r.FileSizes.GAPIDX86APK},
			{filepath.Join(pkg, exeExt("gapid")), &r.FileSizes.GAPID},
			{filepath.Join(pkg, exeExt("gapir")), &r.FileSizes.GAPIR},
			{filepath.Join(pkg, exeExt("gapis")), &r.FileSizes.GAPIS},
			{filepath.Join(pkg, exeExt("gapit")), &r.FileSizes.GAPIT},
		} {
			fi, err := os.Stat(f.path)
			if err != nil {
				continue
			}
			*f.size = fi.Size()
		}
		res = append(res, r)
	}

	fmt.Printf("-----------------------\n")

	w := tabwriter.NewWriter(os.Stdout, 1, 4, 0, ' ', 0)
	defer w.Flush()

	fmt.Fprint(w, "sha"+
		"\t | build_time"+
		"\t | incremental_build_time"+
		"\t | lib_gapii"+
		"\t | lib_vklayervirtualswapchain"+
		"\t | gapid-aarch64.apk"+
		"\t | gapid-armeabi64.apk"+
		"\t | gapid-x86.apk"+
		"\t | gapid"+
		"\t | gapir"+
		"\t | gapis"+
		"\t | gapit\n")
	for _, r := range res {
		fmt.Fprintf(w, "%v,", r.SHA)
		fmt.Fprintf(w, "\t   %v,", r.BuildTime)
		fmt.Fprintf(w, "\t   %v,", r.IncrementalBuildTime)
		fmt.Fprintf(w, "\t   %v,", r.FileSizes.LibGAPII)
		fmt.Fprintf(w, "\t   %v,", r.FileSizes.LibVkLayerVirtualSwapchain)
		fmt.Fprintf(w, "\t   %v,", r.FileSizes.GAPIDAarch64APK)
		fmt.Fprintf(w, "\t   %v,", r.FileSizes.GAPIDArmeabi64APK)
		fmt.Fprintf(w, "\t   %v,", r.FileSizes.GAPIDX86APK)
		fmt.Fprintf(w, "\t   %v,", r.FileSizes.GAPID)
		fmt.Fprintf(w, "\t   %v,", r.FileSizes.GAPIR)
		fmt.Fprintf(w, "\t   %v,", r.FileSizes.GAPIS)
		fmt.Fprintf(w, "\t   %v", r.FileSizes.GAPIT)
		fmt.Fprintf(w, "\n")
	}
	return nil
}

func withTouchedGLES(ctx context.Context, r *rand.Rand, f func() error) error {
	glesAPIPath := filepath.Join(*root, "gapis", "api", "gles", "gles.api")
	fi, err := os.Stat(glesAPIPath)
	if err != nil {
		return err
	}
	glesAPI, err := ioutil.ReadFile(glesAPIPath)
	if err != nil {
		return err
	}
	modGlesAPI := []byte(fmt.Sprintf("%v\ncmd void fake_cmd_%d() {}\n", string(glesAPI), r.Int()))
	ioutil.WriteFile(glesAPIPath, modGlesAPI, fi.Mode().Perm())
	defer ioutil.WriteFile(glesAPIPath, glesAPI, fi.Mode().Perm())
	return f()
}

func build(ctx context.Context) (time.Duration, error) {
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
	start := time.Now()
	_, err := cmd.Call(ctx)
	return time.Since(start), err
}

func dllExt(n string) string {
	switch runtime.GOOS {
	case "windows":
		return n + ".dll"
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
