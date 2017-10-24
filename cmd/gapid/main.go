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
	"errors"
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
	minJavaMajor  = 1
	minJavaMinor  = 8
)

type config struct {
	cwd     string
	vm      string
	vmArgs  []string
	gapic   string
	args    []string
	help    bool
	console bool
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
			fmt.Println(" --vm              Path to the JVM to use")
			fmt.Println(" --vmarg           Extra argument for the JVM (repeatable)")
			fmt.Println(" --console         Run GAPID inside a terminal console")
		}()
	}

	if err := c.locateCWD(); err != nil {
		fmt.Println(err)
		return 1
	}

	if err := c.locateVM(); err != nil {
		fmt.Println(err)
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
		cmd.Env = append(cmd.Env, "SWT_GTK3=0", "LIBOVERLAY_SCROLLBAR=0")
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
		case args[i] == "--vm" && i < len(args)-1:
			i++
			c.vm = args[i]
		case args[i] == "--vmarg" && i < len(args)-1:
			i++
			c.vmArgs = append(c.vmArgs, args[i])
		case args[i] == "--console":
			c.console = true
		default:
			c.help = c.help || args[i] == "--help"
			c.args = append(c.args, args[i])
		}
	}

	c.console = c.console || c.help

	if runtime.GOOS == "darwin" {
		c.vmArgs = append(c.vmArgs, "-XstartOnFirstThread")
	}

	return c
}

func (c *config) locateCWD() error {
	cwd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}
	c.cwd, err = filepath.EvalSymlinks(cwd)
	return err
}

func (c *config) locateVM() error {
	if c.vm != "" {
		if checkVM(c.vm, false) {
			return nil
		}

		if runtime.GOOS == "windows" && checkVM(c.vm+".exe", false) {
			c.vm += ".exe"
			return nil
		}

		if java := c.javaInHome(c.vm); checkVM(java, false) {
			c.vm = java
			return nil
		}
		return fmt.Errorf("JVM '%s' not found", c.vm)
	}

	if java := c.javaInHome(filepath.Join(c.cwd, "jre")); checkVM(java, true) {
		c.vm = java
		return nil
	}

	if java, err := exec.LookPath(c.javaExecutable()); err == nil && checkVM(java, true) {
		c.vm = java
		return nil
	}

	if home := os.Getenv("JAVA_HOME"); home != "" {
		if java := c.javaInHome(home); checkVM(java, true) {
			c.vm = java
			return nil
		}
	}

	if runtime.GOOS == "linux" {
		if java := "/usr/lib/jvm/java-8-openjdk-amd64/jre/bin/java"; checkVM(java, true) {
			c.vm = java
			return nil
		}
	}

	return errors.New("No suitable JVM found")
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

func checkVM(java string, checkVersion bool) bool {
	if stat, err := os.Stat(java); err != nil || stat.IsDir() {
		return false
	}

	if !checkVersion {
		return true
	}

	version, err := exec.Command(java, "-version").CombinedOutput()
	if err != nil {
		return false
	}

	// Looks for the pattern: <product> version "<major>.<minor>.<micro><build>"
	// Not using regular expressions to avoid binary bloat.
	versionStr := string(version)
	if p := strings.Index(versionStr, versionPrefix); p >= 0 {
		p += len(versionPrefix)
		if q := strings.Index(versionStr[p:], "."); q > 0 {
			if r := strings.Index(versionStr[p+q+1:], "."); r > 0 {
				major, _ := strconv.Atoi(versionStr[p : p+q])
				minor, _ := strconv.Atoi(versionStr[p+q+1 : p+q+r+1])
				return major > minJavaMajor || (major == minJavaMajor && minor >= minJavaMinor)
			}
		}
	}

	return false
}

func (c *config) locateGAPIC() error {
	gapic := filepath.Join(c.cwd, "lib", "gapic.jar")
	if _, err := os.Stat(gapic); !os.IsNotExist(err) {
		c.gapic = gapic
		return nil
	}

	jar := "gapic.jar"
	switch runtime.GOOS {
	case "linux", "windows":
		jar = "gapic-" + runtime.GOOS + ".jar"
	case "darwin":
		jar = "gapic-osx.jar"
	}

	buildGapic := filepath.Join(c.cwd, "..", "java", jar)
	if _, err := os.Stat(buildGapic); !os.IsNotExist(err) {
		c.gapic = buildGapic
		return nil
	}

	return fmt.Errorf("GAPIC JAR '%s' not found", gapic)
}
