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
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/core/os/device/remotessh"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

func (f CommandFilterFlags) commandFilter(ctx context.Context, client service.Service, p *path.Capture) (*path.CommandFilter, error) {
	filter := &path.CommandFilter{}
	if f.Context >= 0 {
		contexts, err := client.Get(ctx, p.Contexts().Path(), nil)
		if err != nil {
			return nil, log.Err(ctx, err, "Failed to load the contexts")
		}
		filter.Context = contexts.(*service.Contexts).List[f.Context].ID
	}
	return filter, nil
}

type clientCloser struct {
	client.Client
	preClose []func()
}

func (c clientCloser) Close() error {
	for i := range c.preClose { // Close in reverse order to append
		c.preClose[len(c.preClose)-i-1]()
	}
	return c.Client.Close()
}

func getGapis(ctx context.Context, gapisFlags GapisFlags, gapirFlags GapirFlags) (client.Client, error) {
	args := strings.Fields(gapisFlags.Args)

	args = append(args, "--enable-local-files")

	if app.Flags.Analytics != "" {
		args = append(args, "--analytics", app.Flags.Analytics)
	}
	if gapirFlags.Args != "" {
		// Pass the arguments for gapir further to gapis. Add flag to tag the
		// gapir argument string for gapis.
		args = append(args, "--gapir-args", gapirFlags.Args)
	}
	args = append(args, "--idle-timeout", "1m")

	var token auth.Token
	if gapisFlags.Port == 0 {
		token = auth.GenToken()
	} else {
		token = auth.Token(gapisFlags.Token)
	}
	client, err := client.Connect(ctx, client.Config{
		Port:  gapisFlags.Port,
		Args:  args,
		Token: token,
	})
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}

	close := []func(){}

	openFile := func(path string) (*os.File, error) {
		if path == "" {
			return nil, nil
		}
		file, err := os.Create(path)
		if err != nil {
			return nil, log.Errf(ctx, err, "Failed to open file at '%v'", path)
		}
		close = append(close, func() { file.Close() })
		return file, nil
	}

	var pprof, trace *os.File
	if gapisFlags.Profile.Pprof != "" {
		pprof, err = openFile(gapisFlags.Profile.Pprof)
		if err != nil {
			return nil, err
		}
	}
	if gapisFlags.Profile.Trace != "" {
		trace, err = openFile(gapisFlags.Profile.Trace)
		if err != nil {
			return nil, err
		}
	}
	if pprof != nil || trace != nil {
		stop, err := client.Profile(ctx, pprof, trace, 1)
		if err != nil {
			log.E(ctx, "Profile failed: %v", err)
			return nil, err
		}
		close = append(close, func() { stop() })
	}

	// We start this goroutine to send a heartbeat to gapis.
	// It has an idle-timeout of 1m, so for long requests,
	// pinging every 1s should prevent it from closing down unexpectedly.
	crash.Go(func() {
		hb := time.NewTicker(time.Millisecond * 1000)
		for {
			select {
			case <-ctx.Done():
				return
			case <-hb.C:
				if err := client.Ping(ctx); err != nil {
					return
				}
			}
		}
	})

	if h := log.GetHandler(ctx); h != nil {
		crash.Go(func() { client.GetLogStream(ctx, h) })
	}

	return clientCloser{client, close}, nil
}

func getDevice(ctx context.Context, client client.Client, capture *path.Capture, flags GapirFlags) (*path.Device, error) {
	if flags.Device == "none" {
		return nil, nil
	}
	ctx = log.V{"device": flags.Device}.Bind(ctx)
	paths, err := client.GetDevicesForReplay(ctx, capture)
	if err != nil {
		return nil, log.Err(ctx, err, "Failed query list of devices for replay")
	}

	filteredByFlags := make([]*path.Device, 0, len(paths))
	filteredSerialsOrNames := map[*path.Device]string{}
	getSerialOrName := func(d *device.Instance) string {
		if len(d.GetSerial()) > 0 {
			return d.GetSerial()
		}
		return d.GetName()
	}
	for i := 0; i < len(paths); i++ {
		p := paths[i]
		o, err := client.Get(ctx, p.Path(), nil)
		if err != nil {
			return nil, log.Err(ctx, err, "Couldn't resolve device")
		}
		d := o.(*device.Instance)
		switch flags.Device {
		case "":
			// empty flag
			filteredByFlags = append(filteredByFlags, p)
			filteredSerialsOrNames[p] = getSerialOrName(d)
		case "host":
			if d == host.Instance(ctx) {
				filteredByFlags = append(filteredByFlags, p)
				filteredSerialsOrNames[p] = getSerialOrName(d)
				break
			}
		case "android":
			if d.GetConfiguration().GetOS().GetKind() == device.OSKind_Android {
				filteredByFlags = append(filteredByFlags, p)
				filteredSerialsOrNames[p] = getSerialOrName(d)
			}
		default:
			// serial number
			// TODO: Regex matching instead of exact match?
			if d.GetSerial() == flags.Device {
				filteredByFlags = append(filteredByFlags, p)
				filteredSerialsOrNames[p] = getSerialOrName(d)
				break
			}
		}
	}

	if len(filteredByFlags) > 1 {
		log.I(ctx, "Found multiple usable devices (by serial, or name if serial is empty:")
		for _, sn := range filteredSerialsOrNames {
			log.I(ctx, "\t%s", sn)
		}
	}

	if len(filteredByFlags) > 0 {
		selected := filteredByFlags[0]
		log.I(ctx, "Selected device (by serial, or name if serial is empty): %s",
			filteredSerialsOrNames[selected])
		return selected, nil
	}

	log.W(ctx, "No compatible devices found. Attempting to use the first device anyway...")

	paths, err = client.GetDevices(ctx)
	if err != nil {
		return nil, log.Err(ctx, err, "Failed query list of devices")
	}

	if len(paths) > 0 {
		return paths[0], nil
	}

	return nil, log.Err(ctx, nil, "No devices found")
}

func getDesktopTraceDevice(ctx context.Context, flags GapiiFlags) (bind.Device, error) {
	switch flags.Device {
	case "none":
		return nil, nil
	case "host", "":
		return bind.Host(ctx), nil
	default:
		f, err := os.Open(flags.Ssh.Config)
		if err != nil {
			return nil, err
		}

		devices, err := remotessh.Devices(ctx, f)
		if err != nil {
			return nil, err
		}
		for _, d := range devices {
			if d.Instance().Name == flags.Device {
				return d, nil
			}
		}
	}
	return nil, log.Errf(ctx, nil, "Could not find compatible device %s", flags.Device)
}

func getADBDevice(ctx context.Context, pattern string) (adb.Device, error) {
	devices, err := adb.Devices(ctx)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("No devices found")
	}
	log.I(ctx, "Device list:")
	for _, test := range devices {
		log.I(ctx, "  %v", test.Instance().Serial)
	}
	matchingDevices := []adb.Device{}
	if pattern == "" {
		matchingDevices = devices
	} else {
		re := regexp.MustCompile("(?i)" + pattern)
		for _, test := range devices {
			if re.MatchString(test.Instance().Serial) {
				matchingDevices = append(matchingDevices, test)
			}
		}
	}
	if len(matchingDevices) == 0 {
		return nil, fmt.Errorf("No devices matching %q found", pattern)
	} else if len(matchingDevices) > 1 {
		fmt.Fprintln(os.Stderr, "Matching devices:")
		for _, test := range matchingDevices {
			fmt.Fprint(os.Stderr, "    ")
			fmt.Fprintln(os.Stderr, test.Instance().Serial)
		}
		return nil, fmt.Errorf("Multiple devices matching %q found", pattern)
	}
	return matchingDevices[0], nil
}

func getEvents(ctx context.Context, client service.Service, p *path.Events) ([]*service.Event, error) {
	b, err := client.Get(ctx, p.Path(), nil)
	if err != nil {
		return nil, log.Errf(ctx, err, "Couldn't get events at: %v", p)
	}
	return b.(*service.Events).List, nil
}

func getCommand(ctx context.Context, client service.Service, p *path.Command) (*api.Command, error) {
	boxedCmd, err := client.Get(ctx, p.Path(), nil)
	if err != nil {
		return nil, log.Errf(ctx, err, "Couldn't load command at: %v", p)
	}
	return boxedCmd.(*api.Command), nil
}

var constantSetCache = map[string]*service.ConstantSet{}

func getConstantSet(ctx context.Context, client service.Service, p *path.ConstantSet) (*service.ConstantSet, error) {
	key := fmt.Sprintf("%v", p)
	if cs, ok := constantSetCache[key]; ok {
		return cs, nil
	}
	boxedConstants, err := client.Get(ctx, p.Path(), nil)
	if err != nil {
		return nil, log.Errf(ctx, err, "Couldn't local constant set at: %v", p)
	}
	out := boxedConstants.(*service.ConstantSet)
	constantSetCache[key] = out
	return out, nil
}

func printCommand(ctx context.Context, client service.Service, p *path.Command, c *api.Command, of ObservationFlags) error {
	indices := make([]string, len(p.Indices))
	for i, v := range p.Indices {
		indices[i] = fmt.Sprintf("%d", v)
	}

	params := make([]string, len(c.Parameters))
	for i, p := range c.Parameters {
		v := p.Value.Get()
		if p.Constants != nil {
			constants, err := getConstantSet(ctx, client, p.Constants)
			if err != nil {
				return log.Err(ctx, err, "Couldn't fetch constant set")
			}
			v = constants.Sprint(v)
		}
		params[i] = fmt.Sprintf("%v: %v", p.Name, v)
	}
	fmt.Printf("%v %v(%v)", indices, c.Name, strings.Join(params, ", "))
	if c.Result != nil {
		v := c.Result.Value.Get()
		if c.Result.Constants != nil {
			constants, err := getConstantSet(ctx, client, c.Result.Constants)
			if err != nil {
				return log.Err(ctx, err, "Couldn't fetch constant set")
			}
			v = constants.Sprint(v)
		}
		fmt.Printf(" → %v", v)
	}

	fmt.Fprintln(os.Stdout, "")

	if of.Ranges || of.Data {
		mp := p.MemoryAfter(0, 0, math.MaxUint64)
		mp.ExcludeData = true
		mp.ExcludeObserved = true
		boxedMemory, err := client.Get(ctx, mp.Path(), nil)
		if err != nil {
			return log.Err(ctx, err, "Couldn't fetch memory observations")
		}
		m := boxedMemory.(*service.Memory)
		for _, read := range m.Reads {
			fmt.Printf("   R: [%v - %v]\n",
				memory.BytePtr(read.Base),
				memory.BytePtr(read.Base+read.Size-1))
			if of.Data {
				printMemoryData(ctx, client, p, read)
			}
		}
		for _, write := range m.Writes {
			fmt.Printf("   W: [%v - %v]\n",
				memory.BytePtr(write.Base),
				memory.BytePtr(write.Base+write.Size-1))
			if of.Data {
				printMemoryData(ctx, client, p, write)
			}
		}
	}
	return nil
}

func printMemoryData(ctx context.Context, client service.Service, p *path.Command, rng *service.MemoryRange) error {
	mp := p.MemoryAfter(0, rng.Base, rng.Size)
	mp.ExcludeObserved = true
	boxedMemory, err := client.Get(ctx, mp.Path(), nil)
	if err != nil {
		return log.Err(ctx, err, "Couldn't fetch memory observations")
	}
	memory := boxedMemory.(*service.Memory)
	fmt.Printf("%x\n", memory.Data)
	return nil
}

func getAndPrintCommand(ctx context.Context, client service.Service, p *path.Command, of ObservationFlags) error {
	cmd, err := getCommand(ctx, client, p)
	if err != nil {
		return err
	}
	return printCommand(ctx, client, p, cmd, of)
}

func filterDevices(ctx context.Context, flags *DeviceFlags, gapis client.Client) ([]*path.Device, error) {
	if flags.Device != "" && flags.Serial != "" {
		return nil, fmt.Errorf("You may only specify one of -device or -serial")
	}

	if flags.Device == "host" {
		serverInfo, err := gapis.GetServerInfo(ctx)
		if err != nil {
			return nil, err
		}
		return []*path.Device{serverInfo.ServerLocalDevice}, nil
	}

	devices, err := gapis.GetDevices(ctx)
	if err != nil {
		return nil, err
	}

	out := []*path.Device{}
	for _, dev := range devices {
		dd, err := gapis.Get(ctx, dev.Path(), nil)
		if err != nil {
			return nil, err
		}
		d := dd.(*device.Instance)

		if flags.Device != "" {
			if d.Name != flags.Device {
				continue
			}
		}

		if flags.Serial != "" {
			if d.Serial != flags.Serial {
				continue
			}
		}
		if flags.Os != "" {
			expectedOS := device.OSKind_UnknownOS
			switch strings.ToLower(flags.Os) {
			case "android":
				expectedOS = device.OSKind_Android
			case "linux":
				expectedOS = device.OSKind_Linux
			case "windows", "win":
				expectedOS = device.OSKind_Windows
			case "osx", "macos":
				expectedOS = device.OSKind_OSX
			default:
				return nil, fmt.Errorf("Unknown OS %+v", flags.Os)
			}
			if d.Configuration.OS.Kind != expectedOS {
				continue
			}
		}
		out = append(out, dev)
	}
	return out, nil
}
