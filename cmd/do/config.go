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
	"runtime"
	"strings"

	"github.com/google/gapid/core/fault"
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
	JavaHome       file.Path `desc:"Path to JDK root"`
	AndroidSDKRoot file.Path `desc:"Path to Android SDK"`
	AndroidNDKRoot file.Path `desc:"Path to Android NDK"`
	CMakePath      file.Path `desc:"Path to CMake executable"`
	NinjaPath      file.Path `desc:"Path to ninja executable"`
	PythonPath     file.Path `desc:"Path to python executable"`
	MSYS2Path      file.Path `desc:"Path to msys2 root" os:"windows"`
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
	if !found || options.Interactive {
		cfg = interactiveConfig(ctx, cfg, options)
	}
	for {
		err := validateConfig(ctx, cfg)
		if err == nil {
			return cfg
		}
		fmt.Printf("Configuration is not valid: %v\n", err)
		cfg = interactiveConfig(ctx, cfg, options)
	}
}

func validateConfig(ctx context.Context, cfg Config) error {
	for _, test := range []struct {
		name string
		path file.Path
	}{
		{"cmake", cfg.CMakePath},
		{"ninja", cfg.NinjaPath},
		{"python", cfg.PythonPath},
	} {
		if !test.path.Exists() {
			return fault.Const("Could not find " + test.name)
		}
		if test.path.IsDir() {
			return fault.Const("The provided " + test.name + " is a directory,\nplease provide the path to the " + test.name + " executable")
		}
	}
	return nil
}

func interactiveStructConfig(ctx context.Context, v reflect.Value, options ConfigOptions) {
	t := v.Type()
	for i, c := 0, t.NumField(); i < c; i++ {
		f, t := v.Field(i), t.Field(i)
		desc := t.Tag.Get("desc")
		if desc == "" {
			desc = t.Name
		}
		os := t.Tag.Get("os")
		if os != "" && os != runtime.GOOS {
			continue
		}
		v := f.Addr().Interface()
		switch v := v.(type) {
		case enum:
			options := v.Options()
			fmt.Printf(" • %s. One of: %v [Default: %v]\n", desc, strings.Join(options, ", "), v)
			if in := readLine(); in != "" {
				if !v.Set(in) {
					fmt.Printf("Must be one of: %v\n", strings.Join(options, ", "))
					i--
					continue
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
				if val, ok := parseBool(in); !ok {
					fmt.Printf("Must be yes/true or no/false\n")
					i--
					continue
				} else {
					*v = val
				}
			}
		case *file.Path:
			fmt.Printf(" • %s [Default: %v]\n", desc, v.System())
			if in := readLine(); in != "" {
				*v = file.Abs(in)
			}
		default:
			if t.Type.Kind() == reflect.Interface {
				if !f.IsNil() {
					interactiveStructConfig(ctx, f.Elem(), options)
				}
				continue
			}
			panic(fmt.Sprintf("Unknown type %T in config struct", v))
		}
	}
}

func interactiveConfig(ctx context.Context, cfg Config, options ConfigOptions) Config {
	v := reflect.ValueOf(&cfg).Elem()
	interactiveStructConfig(ctx, v, options)
	writeConfig(cfg)
	return cfg
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
