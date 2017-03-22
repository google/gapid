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

package android

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/gapid/core/os/device"
)

// https://android.googlesource.com/platform/bionic/+/master/libc/include/sys/system_properties.h#38
// Actually 32, but that includes null-terminator.
const maxPropName = 31

// InstalledPackage describes a package installed on a device.
type InstalledPackage struct {
	Name            string          // Name of the package.
	Device          Device          // The device this package is installed on.
	ActivityActions ActivityActions // The activity actions this package supports.
	ServiceActions  ServiceActions  // The service actions this package supports.
	ABI             *device.ABI     // The ABI of the package or empty
	Debuggable      bool            // Whether the package is debuggable or not
	VersionCode     int             // The version code as reported by the manifest.
	VersionName     string          // The version name as reported by the manifest.
	MinSDK          int             // The minimum SDK reported by the manifest.
	TargetSdk       int             // The target SDK reported by the manifest.
}

// InstalledPackages is a list of installed packages.
type InstalledPackages []*InstalledPackage

func (l InstalledPackages) Len() int           { return len(l) }
func (l InstalledPackages) Less(i, j int) bool { return l[i].Name < l[j].Name }
func (l InstalledPackages) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

// ErrProcessNotFound is returned by InstalledPackage.Pid when no running process of the package is found.
var ErrProcessNotFound = fmt.Errorf("Process not found")

// FindByPartialName returns a list of installed packages whose name contains or
// equals s (case insensitive).
func (l InstalledPackages) FindByPartialName(s string) InstalledPackages {
	s = strings.ToLower(s)
	found := make(InstalledPackages, 0, 1)
	for _, p := range l {
		if strings.Contains(strings.ToLower(p.Name), s) {
			found = append(found, p)
		}
	}
	return found
}

// FindByName returns the installed package whose name exactly matches s.
func (l InstalledPackages) FindByName(s string) *InstalledPackage {
	for _, p := range l {
		if p.Name == s {
			return p
		}
	}
	return nil
}

// FindSingleByPartialName returns the single installed package whose name
// contains or equals s (case insensitive). If none or more than one packages
// partially matches s then an error is returned. If a package exactly matches s
// then that is returned regardless of any partial matches.
func (l InstalledPackages) FindSingleByPartialName(s string) (*InstalledPackage, error) {
	found := l.FindByPartialName(s)
	if len(found) == 0 {
		return nil, fmt.Errorf("No packages found containing the name '%v'", s)
	}

	if len(found) > 1 {
		names := make([]string, len(found))
		for i, p := range found {
			if p.Name == s { // Exact match
				return p, nil
			}
			names[i] = p.Name
		}
		return nil, fmt.Errorf("%v packages found containing the name '%v':\n%v",
			len(found), s, strings.Join(names, "\n"))
	}

	return found[0], nil
}

// WrapProperties returns the list of wrap-properties for the given installed
// package.
func (p *InstalledPackage) WrapProperties(ctx context.Context) ([]string, error) {
	list, err := p.Device.Shell("getprop", p.wrapPropName()).Call(ctx)
	return strings.Fields(list), err
}

// SetWrapProperties sets the list of wrap-properties for the given installed
// package.
func (p *InstalledPackage) SetWrapProperties(ctx context.Context, props ...string) error {
	arg := strings.Join(props, " ")
	res, err := p.Device.Shell("setprop", p.wrapPropName(), arg).Call(ctx)
	if res != "" {
		return fmt.Errorf("setprop returned error:\n%s", res)
	}
	return err
}

// ClearCache deletes all data associated with a package.
func (p *InstalledPackage) ClearCache(ctx context.Context) error {
	return p.Device.Shell("pm", "clear", p.Name).Run(ctx)
}

// Stop stops any activities belonging to the package from running on the device.
func (p *InstalledPackage) Stop(ctx context.Context) error {
	return p.Device.Shell("am", "force-stop", p.Name).Run(ctx)
}

// Path returns the absolute path of the installed package on the device.
func (p *InstalledPackage) Path(ctx context.Context) (string, error) {
	out, err := p.Device.Shell("pm", "path", p.Name).Call(ctx)
	if err != nil {
		return "", err
	}
	prefix := "package:"
	if !strings.HasPrefix(out, prefix) {
		return "", fmt.Errorf("Unexpected output: '%s'", out)
	}
	path := out[len(prefix):]
	return path, err
}

// FileDir returns the absolute path of the installed packages files directory.
func (p *InstalledPackage) FileDir(ctx context.Context) (string, error) {
	out, err := p.Device.Shell("run-as", p.Name, "pwd").Call(ctx)
	if err != nil {
		return "", err
	}
	path := out + "/files"
	return path, err
}

// Pid returns the PID of the oldest (if pgrep exists) running process belonging to the given package.
func (p *InstalledPackage) Pid(ctx context.Context) (int, error) {
	// First, try pgrep.
	out, err := p.Device.Shell("pgrep", "-o", "-f", p.Name).Call(ctx)
	if err == nil {
		if out == "" {
			// Empty pgrep output. Process not found.
			return -1, ErrProcessNotFound
		}
		if regexp.MustCompile("^[0-9]+$").MatchString(out) {
			pid, _ := strconv.Atoi(out)
			return pid, nil
		}
	}

	// pgrep not found or other error, fall back to trying ps.
	out, err = p.Device.Shell("ps").Call(ctx)
	if err != nil {
		return -1, err
	}

	matches := regexp.MustCompile(
		`(?m)^\S+\s+([0-9]+)\s+[0-9]+\s+[0-9]+\s+[^\n\r]*\s+(\S+)\s*$`).FindAllStringSubmatch(out, -1)
	if matches != nil {
		// If we're here, we're getting sensible output from ps.
		for _, match := range matches {
			if match[2] == p.Name {
				pid, _ := strconv.Atoi(match[1])
				return pid, nil
			}
		}
		// Process not found.
		return -1, ErrProcessNotFound
	}

	return -1, fmt.Errorf("failed to get pid for package (pgrep and ps both missing or misbehaving)")
}

// Pull pulls the installed package from the device to the specified local directory.
func (p *InstalledPackage) Pull(ctx context.Context, target string) error {
	path, err := p.Path(ctx)
	if err != nil {
		return err
	}
	return p.Device.Pull(ctx, path, target)
}

// Uninstall uninstalls the package from the device.
func (p *InstalledPackage) Uninstall(ctx context.Context) error {
	return p.Device.Shell("pm", "uninstall", p.Name).Run(ctx)
}

// String returns the package name.
func (p *InstalledPackage) String() string {
	return p.Name
}

func (p *InstalledPackage) wrapPropName() string {
	name := "wrap." + p.Name
	if len(name) > maxPropName {
		// The property name must not end in dot
		name = strings.TrimRight(name[:maxPropName], ".")
	}
	return name
}
