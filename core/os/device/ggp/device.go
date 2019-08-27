// Copyright (C) 2019 Google Inc.
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

package ggp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	bd "github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/device/remotessh"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
)

// Binding represents an attached ggp ssh client
type Binding struct {
	remotessh.Device
	Inst string
}

const (
	// Frequency at which to print scan errors
	printScanErrorsEveryNSeconds = 120
)

var _ bind.Device = &Binding{}

var (
	// Registry of all the discovered devices.
	registry = bind.NewRegistry()

	// cache is a map of device names to fully resolved bindings.
	cache      = map[string]*Binding{}
	cacheMutex sync.Mutex // Guards cache.
)

// GGP is the path to the ggp executable.
var GGP file.Path

// GGPExecutablePath returns the path to the ggp
// executable
func GGPExecutablePath() (file.Path, error) {
	if !GGP.IsEmpty() {
		return GGP, nil
	}
	ggpExe := "ggp"
	search := []string{ggpExe}

	if ggpSDKPath := os.Getenv("GGP_SDK_PATH"); ggpSDKPath != "" {
		search = append(search,
			filepath.Join(ggpSDKPath, "dev", "bin", ggpExe))
	}
	for _, path := range search {
		if p, err := file.FindExecutable(path); err == nil {
			GGP = p
			return GGP, nil
		}
	}

	return file.Path{}, fmt.Errorf(
		"ggp could not be found from GGP_SDK_PATH or PATH\n"+
			"GGP_SDK_PATH: %v\n"+
			"PATH: %v\n"+
			"search: %v",
		os.Getenv("GGP_SDK_PATH"), os.Getenv("PATH"), search)
}

func ggpDefaultRootConfigPath() (string, error) {
	if runtime.GOOS == "windows" {
		return os.Getenv("APPDATA"), nil
	}
	if p := os.Getenv("XDG_CONFIG_HOME"); p != "" {
		return path.Clean(p), nil
	}
	if p := os.Getenv("HOME"); p != "" {
		return path.Join(path.Clean(p), ".config"), nil
	}
	return "", fmt.Errorf("Can not find environment")
}

type GGPConfiguration struct {
	remotessh.Configuration
	Inst string
}

func getConfigs(ctx context.Context) ([]GGPConfiguration, error) {
	configs := []GGPConfiguration{}

	ggpPath, err := GGPExecutablePath()
	if err != nil {
		return nil, log.Errf(ctx, err, "Could not find ggp executable to list instances")
	}
	cli := ggpPath.System()

	cmd := shell.Command(cli, "instance", "list")
	instanceListOutBuf := &bytes.Buffer{}
	instanceListErrBuf := &bytes.Buffer{}

	if err := cmd.Capture(instanceListOutBuf, instanceListErrBuf).Run(ctx); err != nil {
		return nil, err
	}

	t, err := ParseListOutput(instanceListOutBuf)
	if err != nil {
		return nil, log.Errf(ctx, err, "parse instance list")
	}
	for _, inf := range t.Rows {
		if inf[3] != "RESERVED" && inf[3] != "IN_USE" {
			continue
		}
		sshInitOutBuf := &bytes.Buffer{}
		sshInitErrBuf := &bytes.Buffer{}

		if err := shell.Command(cli, "ssh", "init", "--instance", inf[1], "-s").Capture(sshInitOutBuf, sshInitErrBuf).Run(ctx); err != nil {
			log.W(ctx, "'ggp ssh init --instance %v -s' finished with error: %v", inf[1], err)
			continue
		}
		if sshInitErrBuf.Len() != 0 {
			log.W(ctx, "'ggp ssh init --instance %v -s' finished with error: %v", inf[1], sshInitErrBuf.String())
			continue
		}
		envs := []string{
			"YETI_DISABLE_GUEST_ORC=1",
			"YETI_DISABLE_STREAMER=1",
		}
		cfg := remotessh.Configuration{
			Name: inf[0],
			Env:  envs,
		}
		if err := json.Unmarshal(sshInitOutBuf.Bytes(), &cfg); err != nil {
			log.W(ctx, "Failed at unmarshaling 'ggp ssh init --instance %v -s' output, fallback to use default config, err: %v", inf[1], err)
		}
		configs = append(configs, GGPConfiguration{cfg, inf[1]})
	}
	return configs, nil
}

// Monitor updates the registry with devices that are added and removed at the
// specified interval. Monitor returns once the context is cancelled.
func Monitor(ctx context.Context, r *bind.Registry, interval time.Duration) error {
	unlisten := registry.Listen(bind.NewDeviceListener(r.AddDevice, r.RemoveDevice))
	defer unlisten()

	for _, d := range registry.Devices() {
		r.AddDevice(ctx, d)
	}

	var lastErrorPrinted time.Time

	for {
		configs, err := getConfigs(ctx)
		if err != nil {
			return err
		}
		if err := scanDevices(ctx, configs); err != nil {
			if time.Since(lastErrorPrinted).Seconds() > printScanErrorsEveryNSeconds {
				log.E(ctx, "Couldn't scan devices: %v", err)
				lastErrorPrinted = time.Now()
			}
		} else {
			lastErrorPrinted = time.Time{}
		}

		select {
		case <-task.ShouldStop(ctx):
			return nil
		case <-time.After(interval):
		}
	}
}

// Devices returns the list of attached GGP devices.
func Devices(ctx context.Context) ([]bind.Device, error) {
	configs, err := getConfigs(ctx)

	if err != nil {
		return nil, err
	}

	if err := scanDevices(ctx, configs); err != nil {
		return nil, err
	}
	devs := registry.Devices()
	out := make([]bind.Device, len(devs))
	for i, d := range devs {
		out[i] = d
	}
	return out, nil
}

func deviceStillConnected(ctx context.Context, d *Binding) bool {
	return d.Status(ctx) == bind.Status_Online
}

func scanDevices(ctx context.Context, configurations []GGPConfiguration) error {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	allConfigs := make(map[string]bool)

	for _, cfg := range configurations {
		allConfigs[cfg.Name] = true

		// If this device already exists, see if we
		// can/have to remove it
		if cached, ok := cache[cfg.Name]; ok {
			if !deviceStillConnected(ctx, cached) {
				delete(cache, cfg.Name)
				registry.RemoveDevice(ctx, cached)
			}
		} else {
			if device, err := remotessh.GetConnectedDevice(ctx, cfg.Configuration); err == nil {
				dev := Binding{
					Device: device,
					Inst:   cfg.Inst,
				}
				device.Instance().Configuration.OS.Kind = bd.Stadia
				registry.AddDevice(ctx, dev)
				cache[cfg.Name] = &dev
			}
		}
	}

	for name, dev := range cache {
		if _, ok := allConfigs[name]; !ok {
			delete(cache, name)
			registry.RemoveDevice(ctx, *dev)
		}
	}
	return nil
}

// ListExecutables lists all executables.
// On GGP, executables may not have the executable bit set,
// so treat any file as executable
func (b Binding) ListExecutables(ctx context.Context, inPath string) ([]string, error) {
	if inPath == "" {
		inPath = b.GetURIRoot()
	}
	// 'find' may partially succeed. Redirect the error messages to /dev/null, only
	// process the found files.
	files, _ := b.Shell("find", `"`+inPath+`"`, "-mindepth", "1", "-maxdepth", "1", "-type", "f", "-printf", `%f\\n`, "2>/dev/null").Call(ctx)
	scanner := bufio.NewScanner(strings.NewReader(files))
	out := []string{}
	for scanner.Scan() {
		_, file := path.Split(scanner.Text())
		out = append(out, file)
	}
	return out, nil
}

// DefaultReplayCacheDir returns the default replay resource cache directory
// on a GGP device
func (b Binding) DefaultReplayCacheDir() string {
	return "/mnt/developer/ggp/gapid/replay_cache"
}
