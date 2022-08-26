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

// The gapid command launches the GAPID UI. It looks for the JVM (bundled or
// from the system), the GAPIC JAR (bundled or from the build output) and
// launches GAPIC with the correct JVM flags and environment variables.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	versionPrefix = `version "`
	googleInfix   = "-google-"
	minJavaMajor  = 11
	minJavaMinor  = 0
)

type config struct {
	cwd     string
	vm      string
	vmArgs  []string
	gapic   string
	args    []string
	help    bool
	console bool
	verbose bool
}

func main() {
	os.Exit(run())
}

func run() int {
	c := newConfig()

	if c.console {
		createConsole()
		if runtime.GOOS == "windows" {
			defer func() {
				fmt.Println()
				fmt.Println("Press enter to continue")
				os.Stdin.Read(make([]byte, 1))
			}()
		}
	}

	if c.help {
		defer func() {
			fmt.Println()
			fmt.Println("Launcher Flags:")
			fmt.Println(" --jar             Path to the gapic JAR to use")
			fmt.Println(" --vm              Path to the JVM to use")
			fmt.Println(" --vmarg           Extra argument for the JVM (repeatable)")
			fmt.Println(" --console         Run AGI inside a terminal console")
			fmt.Println(" --verbose-startup Log verbosely in the launcher")
		}()
	}

	if err := c.locateCWD(); err != nil {
		fmt.Println(err)
		return 1
	}

	if err := c.locateVM(); err != nil {
		fmt.Println(err)
		if !c.verbose {
			fmt.Println("Use --verbose-startup for additional details")
		}
		return 1
	}

	if err := c.locateGAPIC(); err != nil {
		fmt.Println(err)
		return 1
	}

	fmt.Println("Starting", c.vm, c.gapic)
	cmd := exec.Command(c.vm, append(append(c.vmArgs, "-jar", c.gapic), c.args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "GAPID="+c.cwd)

	if runtime.GOOS == "linux" {
		cmd.Env = append(cmd.Env, "LIBOVERLAY_SCROLLBAR=0")
		cmd.Env = append(cmd.Env, "GTK_OVERLAY_SCROLLING=0")
	}

	// If run via 'bazel run', use the shell's CWD, not bazel's.
	if cwd := os.Getenv("BUILD_WORKING_DIRECTORY"); cwd != "" {
		cmd.Dir = cwd
	}

	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			fmt.Println("Failed to start GAPIC:", err)
		}
		return 1
	}
	return 0
}

func newConfig() *config {
	c := &config{}

	// Doing our own flag handling (rather than using go's flag package) to avoid
	// it attempting to parse the GAPIC flags, which may be in a different format.
	// This loop simply looks for the launcher flags, but hands everything else to
	// GAPIC verbatim.
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--jar" && i < len(args)-1:
			i++
			c.gapic = args[i]
		case args[i] == "--vm" && i < len(args)-1:
			i++
			c.vm = args[i]
		case args[i] == "--vmarg" && i < len(args)-1:
			i++
			c.vmArgs = append(c.vmArgs, args[i])
		case args[i] == "--console":
			c.console = true
		case args[i] == "--verbose-startup":
			c.verbose = true
		default:
			c.help = c.help || args[i] == "--help" || args[i] == "--fullhelp"
			c.args = append(c.args, args[i])
		}
	}

	c.console = c.console || c.help

	if runtime.GOOS == "darwin" || runtime.GOOS == "darwin_arm64" {
		c.vmArgs = append(c.vmArgs, "-XstartOnFirstThread")
	}

	return c
}

func (c *config) logIfVerbose(args ...interface{}) {
	if c.verbose {
		fmt.Println(args...)
	}
}

func (c *config) locateCWD() error {
	cwd, err := os.Executable()
	if err != nil {
		return err
	}
	cwd, err = filepath.EvalSymlinks(cwd)
	if err == nil {
		c.cwd = filepath.Dir(cwd)
		c.logIfVerbose("CWD:", c.cwd)
	}
	return err
}

func (c *config) locateVM() error {
	if c.vm != "" {
		if c.checkVM(c.vm, false) {
			return nil
		}

		if runtime.GOOS == "windows" && c.checkVM(c.vm+".exe", false) {
			c.vm += ".exe"
			return nil
		}

		if java := c.javaInHome(c.vm); c.checkVM(java, false) {
			c.vm = java
			return nil
		}
		return fmt.Errorf("JVM '%s' not found/usable", c.vm)
	}

	if java := c.javaInHome(filepath.Join(c.cwd, "jre")); c.checkVM(java, true) {
		c.vm = java
		return nil
	}

	if runtime.GOOS == "linux" {
		if java := "/usr/lib/jvm/java-11-openjdk-amd64/bin/java"; c.checkVM(java, true) {
			c.vm = java
			return nil
		}
	}

	if home := os.Getenv("JAVA_HOME"); home != "" {
		if java := c.javaInHome(home); c.checkVM(java, true) {
			c.vm = java
			return nil
		}
	}

	if java, err := exec.LookPath(c.javaExecutable()); err == nil && c.checkVM(java, true) {
		c.vm = java
		return nil
	}

	return fmt.Errorf("No suitable JVM found. A JRE >= %d.%d is required.", minJavaMajor, minJavaMinor)
}

func (c *config) javaExecutable() string {
	if runtime.GOOS == "windows" {
		if c.console {
			return "java.exe"
		}
		return "javaw.exe"
	}
	return "java"
}

func (c *config) javaInHome(home string) string {
	return filepath.Join(home, "bin", c.javaExecutable())
}

func (c *config) checkVM(java string, checkVersion bool) bool {
	if stat, err := os.Stat(java); err != nil || stat.IsDir() {
		c.logIfVerbose("Not using " + java + ": not a file")
		return false
	}

	if !checkVersion {
		return true
	}

	version, err := exec.Command(java, "-version").CombinedOutput()
	if err != nil {
		c.logIfVerbose("Not using " + java + ": failed to get version info")
		return false
	}

	versionStr := string(version)

	// Don't use the Google custom JDKs as they don't work with our JNI libs.
	if p := strings.Index(versionStr, googleInfix); p >= 0 {
		c.logIfVerbose("Not using " + java + ": is a Google JDK (go/gapid-jdk)")
		return false
	}

	// Looks for the pattern: <product> version "<major>.<minor>.<micro><build>"
	// Not using regular expressions to avoid binary bloat.
	if p := strings.Index(versionStr, versionPrefix); p >= 0 {
		p += len(versionPrefix)
		if q := strings.Index(versionStr[p:], "."); q > 0 {
			if r := strings.Index(versionStr[p+q+1:], "."); r > 0 {
				major, _ := strconv.Atoi(versionStr[p : p+q])
				minor, _ := strconv.Atoi(versionStr[p+q+1 : p+q+r+1])
				useIt := major > minJavaMajor || (major == minJavaMajor && minor >= minJavaMinor)
				if !useIt {
					c.logIfVerbose("Not using " + java + ": unsupported version")
				}
				return useIt
			}
		}
	}

	c.logIfVerbose("Not using " + java + ": failed to parse version")
	return false
}

func (c *config) locateGAPIC() error {
	gapic := c.gapic
	if gapic == "" {
		gapic = filepath.Join(c.cwd, "lib", "gapic.jar")
	}
	if abs, err := filepath.Abs(gapic); err == nil {
		gapic = abs
	}
	if _, err := os.Stat(gapic); !os.IsNotExist(err) {
		c.gapic = gapic
		return nil
	}

	return fmt.Errorf("GAPIC JAR '%s' not found", gapic)
}
