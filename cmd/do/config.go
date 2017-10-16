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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/google/gapid/core/os/file"
)

type enum interface {
	Options() []string
	String() string
	Set(string) bool
}

type Flavor string

func (Flavor) Options() []string { return []string{"release", "debug"} }
func (f Flavor) String() string  { return string(f) }
func (f *Flavor) Set(v string) bool {
	for _, o := range f.Options() {
		if o == v {
			*f = Flavor(o)
			return true
		}
	}
	return false
}

// Config is the structure that holds all the configuration settings.
type Config struct {
	Flavor         Flavor    `desc:"Build flavor"`
	OutRoot        file.Path `desc:"Build output directory"`
	JavaHome       file.Path `desc:"Path to JDK root" type:"dir"`
	AndroidSDKRoot file.Path `desc:"Path to Android SDK" type:"dir"`
	AndroidNDKRoot file.Path `desc:"Path to Android NDK r15" type:"dir"`
	CMakePath      file.Path `desc:"Path to CMake executable" type:"file"`
	NinjaPath      file.Path `desc:"Path to ninja executable" type:"file"`
	PythonPath     file.Path `desc:"Path to python executable" type:"file"`
	MSYS2Path      file.Path `desc:"Path to msys2 root" type:"dir" os:"windows"`
	ArmLinuxGapii  bool      `desc:"Build additional gapii for armlinux"`
}

func defaults() Config {
	u, _ := user.Current()
	cfg := Config{}
	cfg.Flavor = "release"
	cfg.OutRoot = file.Abs(u.HomeDir).Join("gapid")
	cfg.JavaHome = file.Abs(os.Getenv("JAVA_HOME"))
	cfg.AndroidSDKRoot = file.Abs(os.Getenv("ANDROID_HOME"))
	cfg.AndroidNDKRoot = file.Abs(os.Getenv("ANDROID_NDK_ROOT"))
	cfg.CMakePath, _ = file.FindExecutable("cmake")
	cfg.NinjaPath, _ = file.FindExecutable("ninja")
	cfg.PythonPath, _ = file.FindExecutable("python")
	return cfg
}

func (cfg Config) out() file.Path         { return cfg.OutRoot.Join(cfg.Flavor.String()) }
func (cfg Config) bin() file.Path         { return cfg.out().Join("bin") }
func (cfg Config) pkg() file.Path         { return cfg.out().Join("pkg") }
func (cfg Config) versionFile() file.Path { return cfg.out().Join("do-version.txt") }
func (cfg Config) cacheFile() file.Path   { return cfg.out().Join("CMakeCache.txt") }

func (cfg Config) loadBuildVersion() (int, int) {
	data, err := ioutil.ReadFile(cfg.versionFile().System())
	if err != nil {
		return 0, 0
	}
	var major, minor int
	fmt.Sscanf(string(data), "%d.%d", &major, &minor)
	return major, minor
}

func (cfg Config) storeBuildVersion() {
	str := fmt.Sprintf("%d.%d", versionMajor, versionMinor)
	ioutil.WriteFile(cfg.versionFile().System(), []byte(str), 0666)
}

func readConfig() (Config, bool) {
	def := defaults()
	data, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return def, false
	}
	cfg := def
	if err := json.Unmarshal(data, &cfg); err != nil {
		return def, false
	}
	return cfg, true
}

func writeConfig(cfg Config) {
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(cfgPath, data, 0666); err != nil {
		panic(err)
	}
}

func fetchValidConfig(ctx context.Context, options ConfigOptions) Config {
	cfg, found := readConfig()
	if options.Reset {
		cfg = defaults()
	}

	askForAll := !found || options.Interactive

	v := reflect.ValueOf(&cfg).Elem()
	t := v.Type()
	for i, c := 0, t.NumField(); i < c; i++ {
		f, t := v.Field(i), t.Field(i)
		if os := t.Tag.Get("os"); os != "" && os != runtime.GOOS {
			continue
		}
		v := f.Addr().Interface()
		if !askForAll {
			err := vaildateField(v, t)
			if err != nil {
				fmt.Println(err)
			} else {
				continue
			}
		}
		retrys := 0
		for {
			if retrys == 10 {
				fmt.Println("Aborting after 10 failed attempts")
				os.Exit(1)
			}
			retrys++
			if err := inputField(v, t); err != nil {
				fmt.Println(err)
				continue
			}
			if err := vaildateField(v, t); err != nil {
				fmt.Println(err)
				continue
			}
			break
		}
	}
	writeConfig(cfg)
	return cfg
}

func inputField(v interface{}, t reflect.StructField) error {
	desc := t.Tag.Get("desc")
	if desc == "" {
		desc = t.Name
	}
	switch v := v.(type) {
	case enum:
		options := v.Options()
		fmt.Printf(" • %s. One of: %v [Default: %v]\n", desc, strings.Join(options, ", "), v)
		if in := readLine(); in != "" {
			if !v.Set(in) {
				return fmt.Errorf("Must be one of: %v", strings.Join(options, ", "))
			}
		}
	case *string:
		fmt.Printf(" • %s [Default: %q]\n", desc, *v)
		if in := readLine(); in != "" {
			*v = in
		}
	case *bool:
		fmt.Printf(" • %s [Default: %v]\n", desc, *v)
		if in := readLine(); in != "" {
			val, ok := parseBool(in)
			if !ok {
				return fmt.Errorf("Must be yes/true or no/false")
			}
			*v = val
		}
	case *file.Path:
		fmt.Printf(" • %s [Default: %v]\n", desc, v.System())
		if in := readLine(); in != "" {
			*v = file.Abs(in)
		}
	default:
		panic(fmt.Sprintf("Unknown type %T in config struct", v))
	}
	return nil
}

type validator struct{}

// ValidateAndroidNDKRoot checks the AndroidNDKRoot field.
func (validator) ValidateAndroidNDKRoot(path file.Path) error {
	text, err := ioutil.ReadFile(path.Join("source.properties").System())
	if err == nil {
		re := regexp.MustCompile(`Pkg\.Revision = ([0-9]+).([0-9]+).([0-9]+)`)
		for _, line := range strings.Split(string(text), "\n") {
			groups := re.FindStringSubmatch(line)
			if len(groups) < 4 {
				continue
			}
			major, minor := groups[1], groups[2]
			if major != "15" && major != "16" {
				return fmt.Errorf("Found NDK %v.%v. Must be r15 or r16", major, minor)
			}
			return nil
		}
	}
	return fmt.Errorf("Couldn't determine version of the NDK. Must be r15")
}

func vaildateField(v interface{}, t reflect.StructField) error {
	m, ok := reflect.TypeOf(validator{}).MethodByName("Validate" + t.Name)
	if ok {
		err := m.Func.Call([]reflect.Value{
			reflect.ValueOf(validator{}),
			reflect.ValueOf(v).Elem()},
		)[0].Interface()
		if err != nil {
			return err.(error)
		}
	}

	switch v := v.(type) {
	case *file.Path:
		switch t.Tag.Get("type") {
		case "file":
			if !v.Exists() {
				return fmt.Errorf("Path does not exist")
			}
			if v.IsDir() {
				return fmt.Errorf("The provided path is a directory, please provide the path to the executable")
			}
		case "dir":
			if !v.Exists() {
				return fmt.Errorf("Path does not exist")
			}
			if !v.IsDir() {
				return fmt.Errorf("The provided path is not a directory")
			}
		}
	}
	return nil
}

func readLine() string {
	r := bufio.NewReader(os.Stdin)
	l, _ := r.ReadString('\n')
	return strings.Trim(l, "\n\r")
}

func parseBool(str string) (val, ok bool) {
	switch strings.ToLower(str) {
	case "yes", "y", "true":
		return true, true
	case "no", "n", "false":
		return false, true
	}
	return false, false
}
