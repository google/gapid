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

package adb

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android"
	"github.com/google/gapid/core/os/device"
)

// InstalledPackages returns the sorted list of installed packages on the
// device.
func (b *binding) InstalledPackages(ctx context.Context) (android.InstalledPackages, error) {
	str, err := b.Shell("dumpsys", "package").Call(ctx)
	if err != nil {
		return nil, log.Errf(ctx, err, "Failed to get installed packages")
	}
	return b.parsePackages(str)
}

// InstalledPackage returns information about a single installed package on the
// device.
func (b *binding) InstalledPackage(ctx context.Context, name string) (*android.InstalledPackage, error) {
	str, err := b.Shell("dumpsys", "package", name).Call(ctx)
	if err != nil {
		return nil, log.Errf(ctx, err, "Failed to get installed packages")
	}
	packages, err := b.parsePackages(str)
	if err != nil {
		return nil, err
	}
	switch len(packages) {
	case 0:
		return nil, fmt.Errorf("Package '%v' not found", name)
	case 1:
		return packages[0], nil
	default:
		return nil, fmt.Errorf("%v packages found with name '%v'", len(packages), name)
	}
}

// The minSdk field was added more recently [see https://goo.gl/UN7oFv]
var reVersionCodeMinSDKTargetSDK = regexp.MustCompile("^(?:versionCode=([0-9]+))(?: minSdk=([0-9]+))? (?:targetSdk=([0-9]+))?.*$")

func (b *binding) parsePackages(str string) (android.InstalledPackages, error) {
	tree := parseTabbedTree(str)
	packageMap := map[string]*android.InstalledPackage{}
	parseActions := func(group *treeNode, cb func(pkg *android.InstalledPackage, name, owner string)) error {
		actions := group.find("Non-Data Actions:")
		if actions == nil {
			return fmt.Errorf("Could not find Non-Data Actions in dumpsys")
		}
		for _, action := range actions.children {
			for _, entry := range action.children {
				// 43178558 com.google.foo/.FooActivity filter 431d7db8
				// 43178558 com.google.foo/.FooActivity
				fields := strings.Fields(entry.text)
				if len(fields) < 2 {
					return fmt.Errorf("Could not parse package: '%v'", entry.text)
				}
				component := fields[1]
				parts := strings.SplitN(component, "/", 2)
				pkgName := parts[0]
				p, ok := packageMap[pkgName]
				if !ok {
					p = &android.InstalledPackage{
						Name:            pkgName,
						Device:          b,
						ActivityActions: android.ActivityActions{},
						ServiceActions:  android.ServiceActions{},
						ABI:             device.UnknownABI,
					}
					packageMap[pkgName] = p
				}
				actionName := strings.TrimRight(action.text, ":")
				actionOwner := parts[1]
				if strings.HasPrefix(actionOwner, ".") {
					actionOwner = pkgName + actionOwner
				}
				cb(p, actionName, actionOwner)
			}
		}
		return nil
	}

	if activities := tree.find("Activity Resolver Table:"); activities != nil {
		err := parseActions(activities, func(pkg *android.InstalledPackage, name, owner string) {
			pkg.ActivityActions = append(pkg.ActivityActions, &android.ActivityAction{
				Package:  pkg,
				Name:     name,
				Activity: owner,
			})
		})
		if err != nil {
			return nil, err
		}
	}

	if services := tree.find("Service Resolver Table:"); services != nil {
		err := parseActions(services, func(pkg *android.InstalledPackage, name, owner string) {
			pkg.ServiceActions = append(pkg.ServiceActions, &android.ServiceAction{
				Package: pkg,
				Name:    name,
				Service: owner,
			})
		})
		if err != nil {
			return nil, err
		}
	}

	// Read the "Packages:" section if it is present and use it to set ABI
	packSection := tree.find("Packages:")
	if packSection != nil {
		for _, pack := range packSection.children {
			// Package [com.google.foo] (ffffffc):
			fields := strings.Fields(pack.text)
			if len(fields) != 3 {
				continue
			}
			name := strings.Trim(fields[1], "[]")
			ip, ok := packageMap[name]
			if !ok {
				// We didn't find an action for this package
				ip = &android.InstalledPackage{
					Name:            name,
					Device:          b,
					ActivityActions: android.ActivityActions{},
					ServiceActions:  android.ServiceActions{},
					ABI:             device.UnknownABI,
				}
				packageMap[name] = ip
			}

			for _, attr := range pack.children {
				av := strings.TrimSpace(attr.text)

				splits := strings.SplitN(av, "=", 2)
				if len(splits) < 2 {
					continue
				}

				switch {
				case strings.HasPrefix(av, "flags="):
					ip.Debuggable = strings.Contains(av, " DEBUGGABLE ")
				case strings.HasPrefix(av, "versionCode="):
					match := reVersionCodeMinSDKTargetSDK.FindStringSubmatch(av)
					if len(match) == 4 {
						ip.VersionCode, _ = strconv.Atoi(match[1])
						ip.MinSDK, _ = strconv.Atoi(match[2])
						ip.TargetSdk, _ = strconv.Atoi(match[3])
					}
				case strings.HasPrefix(av, "versionName="):
					ip.VersionName = splits[1]
				case strings.HasPrefix(av, "primaryCpuAbi="):
					// primaryCpuAbi=arm64-v8a
					// primaryCpuAbi=null
					if splits[1] == "null" {
						break // This means the package manager will select the platform ABI
					}
					ip.ABI = device.AndroidABIByName(splits[1])
				}
			}
		}
	}
	packages := make(android.InstalledPackages, 0, len(packageMap))
	for _, p := range packageMap {
		packages = append(packages, p)
	}
	sort.Sort(packages)
	return packages, nil
}

type treeNode struct {
	text     string
	children []*treeNode
	parent   *treeNode
	depth    int
}

func (t *treeNode) find(name string) *treeNode {
	if t == nil {
		return nil
	}
	for _, c := range t.children {
		if c.text == name {
			return c
		}
	}
	return nil
}

func parseTabbedTree(str string) *treeNode {
	head := &treeNode{depth: -1}
	extraDepth := 0
	for _, line := range strings.Split(str, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}

		// Calculate the line's depth
		depth := 0
		for _, r := range line {
			if r == ' ' {
				depth++
			} else {
				break
			}
		}
		line = line[depth:]

		if line == "" {
			// A line containing only whitespace probably comes from an extra newline
			// character being printed before the intended line. So, add the depth that
			// would have resulted from this empty line to the next line. Example with
			// front spaces shown as '_':
			// ...
			// Known Packages:
			// _ Package [foo]:
			// __
			//Package categories:
			// ___ category bar
			// ...
			// In the above example "Package categories" was meant to be indented by
			// two spaces, which are present on the previous line.
			extraDepth += depth
			continue
		}

		depth += extraDepth
		extraDepth = 0

		// Find the insertion point
		for {
			if head.depth >= depth {
				head = head.parent
			} else {
				node := &treeNode{text: line, depth: depth, parent: head}
				head.children = append(head.children, node)
				head = node
				break
			}
		}
	}
	for head.parent != nil {
		head = head.parent
	}
	return head
}
